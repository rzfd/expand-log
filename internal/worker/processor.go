package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/schedule"
)

var workerTracer = otel.Tracer("worker.recurring")

type recurringProcessorRepository interface {
	BeginTx(ctx context.Context) (pgx.Tx, error)
	GetNextDueForUpdate(ctx context.Context, tx pgx.Tx, runAt string) (*model.RecurringTransaction, error)
	InsertGeneratedTransaction(ctx context.Context, tx pgx.Tx, item model.RecurringTransaction, runDate string) (bool, error)
	AdvanceSchedule(ctx context.Context, tx pgx.Tx, id int64, nextRunDate *string, active bool) error
}

type Processor struct {
	repo      recurringProcessorRepository
	batchSize int
	now       func() time.Time
}

func NewProcessor(repo recurringProcessorRepository, batchSize int) *Processor {
	return &Processor{
		repo:      repo,
		batchSize: batchSize,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (p *Processor) ProcessDue(ctx context.Context) (int, error) {
	ctx, span := workerTracer.Start(ctx, "worker.process_due")
	defer span.End()
	span.SetAttributes(attribute.Int("worker.batch_size", p.batchSize))

	logger := logging.FromContext(ctx)
	logger.Info().Int("batch_size", p.batchSize).Msg("worker process due started")
	created := 0
	runAt := schedule.NormalizeDate(p.now()).Format("2006-01-02")

	for i := 0; i < p.batchSize; i++ {
		logger.Info().Int("iteration", i+1).Str("run_at", runAt).Msg("worker process due iteration")
		tx, err := p.repo.BeginTx(ctx)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "begin transaction failed")
			logger.Error().Err(err).Int("iteration", i+1).Msg("worker process due begin tx failed")
			return created, fmt.Errorf("begin tx: %w", err)
		}

		item, err := p.repo.GetNextDueForUpdate(ctx, tx, runAt)
		if err != nil {
			_ = tx.Rollback(ctx)
			if errors.Is(err, pgx.ErrNoRows) {
				span.SetAttributes(attribute.Int("worker.created", created))
				logger.Info().Int("created", created).Msg("worker process due no eligible records")
				return created, nil
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, "load next recurring failed")
			logger.Error().Err(err).Int("iteration", i+1).Msg("worker process due load next failed")
			return created, fmt.Errorf("get next due recurring transaction: %w", err)
		}

		if item.NextRunDate == nil {
			_ = tx.Rollback(ctx)
			span.SetStatus(codes.Error, "next run date is nil")
			logger.Error().Int64("recurring_id", item.ID).Msg("worker process due nil next run date")
			return created, fmt.Errorf("recurring transaction %d has nil next_run_date", item.ID)
		}

		runDate := schedule.NormalizeDate(*item.NextRunDate)
		inserted, err := p.repo.InsertGeneratedTransaction(ctx, tx, *item, runDate.Format("2006-01-02"))
		if err != nil {
			_ = tx.Rollback(ctx)
			span.RecordError(err)
			span.SetStatus(codes.Error, "insert generated transaction failed")
			logger.Error().Err(err).Int64("recurring_id", item.ID).Msg("worker process due insert generated failed")
			return created, fmt.Errorf("insert generated transaction: %w", err)
		}

		nextRunDate, active, err := nextSchedule(ctx, item.Frequency, runDate, item.EndDate)
		if err != nil {
			_ = tx.Rollback(ctx)
			span.RecordError(err)
			span.SetStatus(codes.Error, "compute next schedule failed")
			logger.Error().Err(err).Int64("recurring_id", item.ID).Msg("worker process due next schedule failed")
			return created, err
		}

		var nextRunString *string
		if nextRunDate != nil {
			formatted := nextRunDate.Format("2006-01-02")
			nextRunString = &formatted
		}

		if err := p.repo.AdvanceSchedule(ctx, tx, item.ID, nextRunString, active); err != nil {
			_ = tx.Rollback(ctx)
			span.RecordError(err)
			span.SetStatus(codes.Error, "advance schedule failed")
			logger.Error().Err(err).Int64("recurring_id", item.ID).Msg("worker process due advance schedule failed")
			return created, fmt.Errorf("advance recurring schedule: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			_ = tx.Rollback(ctx)
			span.RecordError(err)
			span.SetStatus(codes.Error, "commit transaction failed")
			logger.Error().Err(err).Int64("recurring_id", item.ID).Msg("worker process due commit failed")
			return created, fmt.Errorf("commit tx: %w", err)
		}

		if inserted {
			created++
		}
		logger.Info().Int64("recurring_id", item.ID).Bool("inserted", inserted).Bool("active", active).Msg("worker process due iteration completed")
	}

	logger.Info().Int("created", created).Msg("worker process due completed")
	span.SetAttributes(attribute.Int("worker.created", created))
	return created, nil
}

func nextSchedule(ctx context.Context, frequency model.RecurringFrequency, runDate time.Time, endDate *time.Time) (*time.Time, bool, error) {
	logging.FromContext(ctx).Info().Str("frequency", string(frequency)).Str("run_date", runDate.Format("2006-01-02")).Msg("worker next schedule started")
	nextRun, err := schedule.NextRunDate(frequency, runDate)
	if err != nil {
		logging.FromContext(ctx).Warn().Err(err).Msg("worker next schedule failed")
		return nil, false, err
	}

	if endDate != nil && nextRun.After(schedule.NormalizeDate(*endDate)) {
		logging.FromContext(ctx).Info().Msg("worker next schedule completed inactive")
		return nil, false, nil
	}

	logging.FromContext(ctx).Info().Msg("worker next schedule completed")
	return &nextRun, true, nil
}

type Runner struct {
	processor *Processor
	interval  time.Duration
}

func NewRunner(processor *Processor, interval time.Duration) *Runner {
	return &Runner{
		processor: processor,
		interval:  interval,
	}
}

func (r *Runner) Start(ctx context.Context) {
	logging.FromContext(ctx).Info().Msg("worker runner start")
	r.runOnce(ctx)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logging.FromContext(ctx).Info().Msg("worker runner stopped from context")
			return
		case <-ticker.C:
			r.runOnce(ctx)
		}
	}
}

func (r *Runner) runOnce(ctx context.Context) {
	cycleStart := r.processor.now()
	runID := fmt.Sprintf("%d", cycleStart.UnixNano())
	cycleCtx := logging.WithField(ctx, "run_id", runID)
	cycleCtx, span := workerTracer.Start(cycleCtx, "worker.run_once")
	defer span.End()
	span.SetAttributes(attribute.String("worker.run_id", runID))
	logging.FromContext(cycleCtx).Info().Msg("worker runner cycle started")
	created, err := r.processor.ProcessDue(cycleCtx)
	elapsedMS := time.Since(cycleStart).Milliseconds()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "process due failed")
		logging.FromContext(cycleCtx).Error().Err(err).Int64("elapsed_ms", elapsedMS).Msg("worker run failed")
		return
	}
	span.SetAttributes(
		attribute.Int("worker.created", created),
		attribute.Int64("worker.elapsed_ms", elapsedMS),
	)

	if created > 0 {
		logging.FromContext(cycleCtx).Info().Int("created", created).Msg("worker generated recurring transactions")
	}
	logging.FromContext(cycleCtx).Info().Int("created", created).Int64("elapsed_ms", elapsedMS).Msg("worker runner cycle completed")
}
