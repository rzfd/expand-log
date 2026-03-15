package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/logging"
)

type RecurringTransactionRepository struct {
	db *pgxpool.Pool
}

func NewRecurringTransactionRepository(db *pgxpool.Pool) *RecurringTransactionRepository {
	return &RecurringTransactionRepository{db: db}
}

func (r *RecurringTransactionRepository) Create(ctx context.Context, item *model.RecurringTransaction) error {
	ctx, span := startRepositorySpan(ctx, "repository.recurring.create",
		attribute.Int64("app.user.id", item.UserID),
	)
	defer span.End()

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", item.UserID).Msg("repository recurring create started")
	query := `
		INSERT INTO recurring_transactions (
			user_id, category_id, type, amount_cents, note, frequency, start_date, end_date, next_run_date, active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`

	dbCtx, dbSpan := startDBSpan(ctx, "insert", attribute.String("db.table", "recurring_transactions"))
	setDBStatement(dbSpan, query)
	err := r.db.QueryRow(
		dbCtx,
		query,
		item.UserID,
		item.CategoryID,
		item.Type,
		item.AmountCents,
		item.Note,
		item.Frequency,
		item.StartDate,
		item.EndDate,
		item.NextRunDate,
		item.Active,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		markSpanError(dbSpan, err, "insert recurring failed")
	}
	dbSpan.End()
	if err != nil {
		markSpanError(span, err, "create recurring failed")
		logger.Error().Err(err).Int64("user_id", item.UserID).Msg("repository recurring create failed")
		return err
	}
	logger.Info().Int64("user_id", item.UserID).Int64("recurring_id", item.ID).Msg("repository recurring create completed")
	return nil
}

func (r *RecurringTransactionRepository) GetByIDForUser(ctx context.Context, id, userID int64) (*model.RecurringTransaction, error) {
	ctx, span := startRepositorySpan(ctx, "repository.recurring.get_by_id_for_user",
		attribute.Int64("app.user.id", userID),
		attribute.Int64("app.recurring.id", id),
	)
	defer span.End()

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("recurring_id", id).Msg("repository recurring get by id started")
	query := `
		SELECT rt.id, rt.user_id, rt.category_id, c.name, rt.type, rt.amount_cents, rt.note, rt.frequency, rt.start_date, rt.end_date, rt.next_run_date, rt.active, rt.created_at, rt.updated_at
		FROM recurring_transactions rt
		JOIN categories c ON c.id = rt.category_id
		WHERE rt.id = $1 AND rt.user_id = $2
	`

	item, err := r.getOne(ctx, r.db, query, id, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			span.AddEvent("recurring_not_found")
			logger.Info().Int64("user_id", userID).Int64("recurring_id", id).Msg("repository recurring get by id not found")
			return nil, nil
		}
		markSpanError(span, err, "get recurring failed")
		logger.Error().Err(err).Int64("user_id", userID).Int64("recurring_id", id).Msg("repository recurring get by id failed")
		return nil, err
	}
	logger.Info().Int64("user_id", userID).Int64("recurring_id", item.ID).Msg("repository recurring get by id completed")
	return item, nil
}

func (r *RecurringTransactionRepository) ListByUser(ctx context.Context, userID int64) ([]model.RecurringTransaction, error) {
	ctx, span := startRepositorySpan(ctx, "repository.recurring.list_by_user",
		attribute.Int64("app.user.id", userID),
	)
	defer span.End()

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Msg("repository recurring list by user started")
	query := `
		SELECT rt.id, rt.user_id, rt.category_id, c.name, rt.type, rt.amount_cents, rt.note, rt.frequency, rt.start_date, rt.end_date, rt.next_run_date, rt.active, rt.created_at, rt.updated_at
		FROM recurring_transactions rt
		JOIN categories c ON c.id = rt.category_id
		WHERE rt.user_id = $1
		ORDER BY rt.created_at DESC, rt.id DESC
	`

	dbCtx, dbSpan := startDBSpan(ctx, "select", attribute.String("db.table", "recurring_transactions"))
	setDBStatement(dbSpan, query)
	rows, err := r.db.Query(dbCtx, query, userID)
	if err != nil {
		markSpanError(dbSpan, err, "query recurring failed")
		dbSpan.End()
		markSpanError(span, err, "list recurring failed")
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository recurring list by user query failed")
		return nil, err
	}
	defer rows.Close()
	defer dbSpan.End()

	items := make([]model.RecurringTransaction, 0)
	for rows.Next() {
		item, err := scanRecurringTransaction(rows)
		if err != nil {
			markSpanError(dbSpan, err, "scan recurring failed")
			markSpanError(span, err, "scan recurring failed")
			logger.Error().Err(err).Int64("user_id", userID).Msg("repository recurring list by user scan failed")
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		markSpanError(dbSpan, err, "rows recurring failed")
		markSpanError(span, err, "rows recurring failed")
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository recurring list by user rows failed")
		return nil, err
	}
	dbSpan.SetAttributes(attribute.Int("app.recurring.count", len(items)))
	logger.Info().Int64("user_id", userID).Int("count", len(items)).Msg("repository recurring list by user completed")
	return items, nil
}

func (r *RecurringTransactionRepository) Update(ctx context.Context, item *model.RecurringTransaction) error {
	ctx, span := startRepositorySpan(ctx, "repository.recurring.update",
		attribute.Int64("app.user.id", item.UserID),
		attribute.Int64("app.recurring.id", item.ID),
	)
	defer span.End()

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", item.UserID).Int64("recurring_id", item.ID).Msg("repository recurring update started")
	query := `
		UPDATE recurring_transactions
		SET category_id = $1,
			type = $2,
			amount_cents = $3,
			note = $4,
			frequency = $5,
			start_date = $6,
			end_date = $7,
			next_run_date = $8,
			active = $9,
			updated_at = NOW()
		WHERE id = $10 AND user_id = $11
		RETURNING updated_at
	`

	dbCtx, dbSpan := startDBSpan(ctx, "update", attribute.String("db.table", "recurring_transactions"))
	setDBStatement(dbSpan, query)
	err := r.db.QueryRow(
		dbCtx,
		query,
		item.CategoryID,
		item.Type,
		item.AmountCents,
		item.Note,
		item.Frequency,
		item.StartDate,
		item.EndDate,
		item.NextRunDate,
		item.Active,
		item.ID,
		item.UserID,
	).Scan(&item.UpdatedAt)
	if err != nil {
		markSpanError(dbSpan, err, "update recurring failed")
	}
	dbSpan.End()
	if err != nil {
		markSpanError(span, err, "update recurring failed")
		logger.Error().Err(err).Int64("user_id", item.UserID).Int64("recurring_id", item.ID).Msg("repository recurring update failed")
		return err
	}
	logger.Info().Int64("user_id", item.UserID).Int64("recurring_id", item.ID).Msg("repository recurring update completed")
	return nil
}

func (r *RecurringTransactionRepository) Delete(ctx context.Context, id, userID int64) (bool, error) {
	ctx, span := startRepositorySpan(ctx, "repository.recurring.delete",
		attribute.Int64("app.user.id", userID),
		attribute.Int64("app.recurring.id", id),
	)
	defer span.End()

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("recurring_id", id).Msg("repository recurring delete started")
	dbCtx, dbSpan := startDBSpan(ctx, "delete", attribute.String("db.table", "recurring_transactions"))
	statement := `DELETE FROM recurring_transactions WHERE id = $1 AND user_id = $2`
	setDBStatement(dbSpan, statement)
	result, err := r.db.Exec(dbCtx, statement, id, userID)
	if err != nil {
		markSpanError(dbSpan, err, "delete recurring failed")
		dbSpan.End()
		markSpanError(span, err, "delete recurring failed")
		logger.Error().Err(err).Int64("user_id", userID).Int64("recurring_id", id).Msg("repository recurring delete failed")
		return false, err
	}
	dbSpan.SetAttributes(attribute.Int64("db.rows_affected", result.RowsAffected()))
	dbSpan.End()
	deleted := result.RowsAffected() > 0
	logger.Info().Int64("user_id", userID).Int64("recurring_id", id).Bool("deleted", deleted).Msg("repository recurring delete completed")
	return deleted, nil
}

func (r *RecurringTransactionRepository) GetNextDueForUpdate(ctx context.Context, tx pgx.Tx, runAt string) (*model.RecurringTransaction, error) {
	ctx, span := startRepositorySpan(ctx, "repository.recurring.get_next_due_for_update")
	defer span.End()

	logger := logging.FromContext(ctx)
	logger.Info().Str("run_at", runAt).Msg("repository recurring get next due started")
	query := `
		SELECT rt.id, rt.user_id, rt.category_id, c.name, rt.type, rt.amount_cents, rt.note, rt.frequency, rt.start_date, rt.end_date, rt.next_run_date, rt.active, rt.created_at, rt.updated_at
		FROM recurring_transactions rt
		JOIN categories c ON c.id = rt.category_id
		WHERE rt.active = TRUE
			AND rt.next_run_date IS NOT NULL
			AND rt.next_run_date <= $1
		ORDER BY rt.next_run_date ASC, rt.id ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`

	item, err := r.getOne(ctx, tx, query, runAt)
	if err != nil {
		markSpanError(span, err, "get next due recurring failed")
		logger.Error().Err(err).Str("run_at", runAt).Msg("repository recurring get next due failed")
		return nil, err
	}
	logger.Info().Str("run_at", runAt).Int64("recurring_id", item.ID).Msg("repository recurring get next due completed")
	return item, nil
}

func (r *RecurringTransactionRepository) InsertGeneratedTransaction(ctx context.Context, tx pgx.Tx, item model.RecurringTransaction, runDate string) (bool, error) {
	ctx, span := startRepositorySpan(ctx, "repository.recurring.insert_generated_transaction",
		attribute.Int64("app.recurring.id", item.ID),
	)
	defer span.End()

	logger := logging.FromContext(ctx)
	logger.Info().Int64("recurring_id", item.ID).Str("run_date", runDate).Msg("repository recurring insert generated transaction started")
	query := `
		INSERT INTO transactions (
			user_id, category_id, recurring_transaction_id, type, amount_cents, note, transaction_date, source
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'recurring')
		ON CONFLICT (recurring_transaction_id, transaction_date) DO NOTHING
	`

	dbCtx, dbSpan := startDBSpan(ctx, "insert", attribute.String("db.table", "transactions"))
	setDBStatement(dbSpan, query)
	result, err := tx.Exec(dbCtx, query, item.UserID, item.CategoryID, item.ID, item.Type, item.AmountCents, item.Note, runDate)
	if err != nil {
		markSpanError(dbSpan, err, "insert generated transaction failed")
		dbSpan.End()
		markSpanError(span, err, "insert generated transaction failed")
		logger.Error().Err(err).Int64("recurring_id", item.ID).Str("run_date", runDate).Msg("repository recurring insert generated transaction failed")
		return false, err
	}
	inserted := result.RowsAffected() > 0
	dbSpan.SetAttributes(
		attribute.Int64("db.rows_affected", result.RowsAffected()),
		attribute.Bool("app.transaction.inserted", inserted),
	)
	dbSpan.End()
	logger.Info().Int64("recurring_id", item.ID).Str("run_date", runDate).Bool("inserted", inserted).Msg("repository recurring insert generated transaction completed")
	return inserted, nil
}

func (r *RecurringTransactionRepository) AdvanceSchedule(ctx context.Context, tx pgx.Tx, id int64, nextRunDate *string, active bool) error {
	ctx, span := startRepositorySpan(ctx, "repository.recurring.advance_schedule",
		attribute.Int64("app.recurring.id", id),
	)
	defer span.End()

	logger := logging.FromContext(ctx)
	logger.Info().Int64("recurring_id", id).Bool("active", active).Msg("repository recurring advance schedule started")
	query := `
		UPDATE recurring_transactions
		SET next_run_date = $1, active = $2, updated_at = NOW()
		WHERE id = $3
	`
	dbCtx, dbSpan := startDBSpan(ctx, "update", attribute.String("db.table", "recurring_transactions"))
	setDBStatement(dbSpan, query)
	result, err := tx.Exec(dbCtx, query, nextRunDate, active, id)
	if err != nil {
		markSpanError(dbSpan, err, "advance recurring schedule failed")
		dbSpan.End()
		markSpanError(span, err, "advance recurring schedule failed")
		logger.Error().Err(err).Int64("recurring_id", id).Msg("repository recurring advance schedule failed")
		return err
	}
	dbSpan.SetAttributes(attribute.Int64("db.rows_affected", result.RowsAffected()))
	dbSpan.End()
	logger.Info().Int64("recurring_id", id).Bool("active", active).Msg("repository recurring advance schedule completed")
	return err
}

func (r *RecurringTransactionRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	ctx, span := startRepositorySpan(ctx, "repository.recurring.begin_tx")
	defer span.End()

	logging.FromContext(ctx).Info().Msg("repository recurring begin tx started")
	tx, err := r.db.Begin(ctx)
	if err != nil {
		markSpanError(span, err, "begin tx failed")
		logging.FromContext(ctx).Error().Err(err).Msg("repository recurring begin tx failed")
		return nil, err
	}
	logging.FromContext(ctx).Info().Msg("repository recurring begin tx completed")
	return tx, nil
}

func (r *RecurringTransactionRepository) getOne(ctx context.Context, querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, query string, args ...any) (*model.RecurringTransaction, error) {
	logging.FromContext(ctx).Info().Msg("repository recurring get one started")
	dbCtx, dbSpan := startDBSpan(ctx, "select", attribute.String("db.table", "recurring_transactions"))
	setDBStatement(dbSpan, query)
	row := querier.QueryRow(dbCtx, query, args...)
	item, err := scanRecurringTransaction(row)
	defer dbSpan.End()
	if err != nil {
		markSpanError(dbSpan, err, "select recurring failed")
		logging.FromContext(ctx).Warn().Err(err).Msg("repository recurring get one failed")
		return nil, err
	}
	dbSpan.SetAttributes(attribute.Int64("app.recurring.id", item.ID))
	logging.FromContext(ctx).Info().Int64("recurring_id", item.ID).Msg("repository recurring get one completed")
	return &item, nil
}

func scanRecurringTransaction(scanner interface {
	Scan(dest ...any) error
}) (model.RecurringTransaction, error) {
	logging.FromContext(nil).Info().Msg("repository recurring scan started")
	var item model.RecurringTransaction
	var endDate sql.NullTime
	var nextRunDate sql.NullTime

	err := scanner.Scan(
		&item.ID,
		&item.UserID,
		&item.CategoryID,
		&item.CategoryName,
		&item.Type,
		&item.AmountCents,
		&item.Note,
		&item.Frequency,
		&item.StartDate,
		&endDate,
		&nextRunDate,
		&item.Active,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		logging.FromContext(nil).Warn().Err(err).Msg("repository recurring scan failed")
		return model.RecurringTransaction{}, err
	}

	if endDate.Valid {
		item.EndDate = &endDate.Time
	}
	if nextRunDate.Valid {
		item.NextRunDate = &nextRunDate.Time
	}

	logging.FromContext(nil).Info().Int64("recurring_id", item.ID).Msg("repository recurring scan completed")
	return item, nil
}
