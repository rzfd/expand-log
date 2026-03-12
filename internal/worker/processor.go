package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/schedule"
)

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
	logger := logging.FromContext(ctx)
	logger.Info().Int("batch_size", p.batchSize).Msg("worker process due started")
	created := 0
	runAt := schedule.NormalizeDate(p.now()).Format("2006-01-02")

	for i := 0; i < p.batchSize; i++ {
		logger.Info().Int("iteration", i+1).Str("run_at", runAt).Msg("worker process due iteration")
		tx, err := p.repo.BeginTx(ctx)
		if err != nil {
			logger.Error().Err(err).Int("iteration", i+1).Msg("worker process due begin tx failed")
			return created, fmt.Errorf("begin tx: %w", err)
		}

		item, err := p.repo.GetNextDueForUpdate(ctx, tx, runAt)
		if err != nil {
			_ = tx.Rollback(ctx)
			if errors.Is(err, pgx.ErrNoRows) {
				logger.Info().Int("created", created).Msg("worker process due no eligible records")
				return created, nil
			}
			logger.Error().Err(err).Int("iteration", i+1).Msg("worker process due load next failed")
			return created, fmt.Errorf("get next due recurring transaction: %w", err)
		}

		if item.NextRunDate == nil {
			_ = tx.Rollback(ctx)
			logger.Error().Int64("recurring_id", item.ID).Msg("worker process due nil next run date")
			return created, fmt.Errorf("recurring transaction %d has nil next_run_date", item.ID)
		}

		runDate := schedule.NormalizeDate(*item.NextRunDate)
		inserted, err := p.repo.InsertGeneratedTransaction(ctx, tx, *item, runDate.Format("2006-01-02"))
		if err != nil {
			_ = tx.Rollback(ctx)
			logger.Error().Err(err).Int64("recurring_id", item.ID).Msg("worker process due insert generated failed")
			return created, fmt.Errorf("insert generated transaction: %w", err)
		}

		nextRunDate, active, err := nextSchedule(item.Frequency, runDate, item.EndDate)
		if err != nil {
			_ = tx.Rollback(ctx)
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
			logger.Error().Err(err).Int64("recurring_id", item.ID).Msg("worker process due advance schedule failed")
			return created, fmt.Errorf("advance recurring schedule: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			_ = tx.Rollback(ctx)
			logger.Error().Err(err).Int64("recurring_id", item.ID).Msg("worker process due commit failed")
			return created, fmt.Errorf("commit tx: %w", err)
		}

		if inserted {
			created++
		}
		logger.Info().Int64("recurring_id", item.ID).Bool("inserted", inserted).Bool("active", active).Msg("worker process due iteration completed")
	}

	logger.Info().Int("created", created).Msg("worker process due completed")
	return created, nil
}

func nextSchedule(frequency model.RecurringFrequency, runDate time.Time, endDate *time.Time) (*time.Time, bool, error) {
	logging.FromContext(nil).Info().Str("frequency", string(frequency)).Str("run_date", runDate.Format("2006-01-02")).Msg("worker next schedule started")
	nextRun, err := schedule.NextRunDate(frequency, runDate)
	if err != nil {
		logging.FromContext(nil).Warn().Err(err).Msg("worker next schedule failed")
		return nil, false, err
	}

	if endDate != nil && nextRun.After(schedule.NormalizeDate(*endDate)) {
		logging.FromContext(nil).Info().Msg("worker next schedule completed inactive")
		return nil, false, nil
	}

	logging.FromContext(nil).Info().Msg("worker next schedule completed")
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
	logging.FromContext(ctx).Info().Msg("worker runner cycle started")
	created, err := r.processor.ProcessDue(ctx)
	if err != nil {
		logging.FromContext(ctx).Error().Err(err).Msg("worker run failed")
		return
	}

	if created > 0 {
		logging.FromContext(ctx).Info().Int("created", created).Msg("worker generated recurring transactions")
	}
	logging.FromContext(ctx).Info().Int("created", created).Msg("worker runner cycle completed")
}
