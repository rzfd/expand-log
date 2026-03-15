package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/logging"
)

type BudgetRepository struct {
	db *pgxpool.Pool
}

func NewBudgetRepository(db *pgxpool.Pool) *BudgetRepository {
	return &BudgetRepository{db: db}
}

func (r *BudgetRepository) Create(ctx context.Context, budget *model.Budget) error {
	ctx, span := startRepositorySpan(ctx, "repository.budget.create",
		attribute.Int64("app.user.id", budget.UserID),
	)
	defer span.End()

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", budget.UserID).Msg("repository budget create started")
	query := `
		INSERT INTO budgets (user_id, category_id, year, month, amount_cents)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	dbCtx, dbSpan := startDBSpan(ctx, "insert", attribute.String("db.table", "budgets"))
	setDBStatement(dbSpan, query)
	err := r.db.QueryRow(
		dbCtx,
		query,
		budget.UserID,
		budget.CategoryID,
		budget.Year,
		budget.Month,
		budget.AmountCents,
	).Scan(&budget.ID, &budget.CreatedAt, &budget.UpdatedAt)
	if err != nil {
		markSpanError(dbSpan, err, "insert budget failed")
	}
	dbSpan.End()
	if err != nil {
		markSpanError(span, err, "create budget failed")
		logger.Error().Err(err).Int64("user_id", budget.UserID).Msg("repository budget create failed")
		return err
	}
	logger.Info().Int64("user_id", budget.UserID).Int64("budget_id", budget.ID).Msg("repository budget create completed")
	return nil
}

func (r *BudgetRepository) GetByIDForUser(ctx context.Context, id, userID int64) (*model.Budget, error) {
	ctx, span := startRepositorySpan(ctx, "repository.budget.get_by_id_for_user",
		attribute.Int64("app.user.id", userID),
		attribute.Int64("app.budget.id", id),
	)
	defer span.End()

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("budget_id", id).Msg("repository budget get by id started")
	query := `
		SELECT
			b.id,
			b.user_id,
			b.category_id,
			c.name,
			b.year,
			b.month,
			b.amount_cents,
			COALESCE((
				SELECT SUM(t.amount_cents)
				FROM transactions t
				WHERE t.user_id = b.user_id
					AND t.category_id = b.category_id
					AND t.type = 'expense'
					AND EXTRACT(YEAR FROM t.transaction_date) = b.year
					AND EXTRACT(MONTH FROM t.transaction_date) = b.month
			), 0) AS spent_cents,
			b.created_at,
			b.updated_at
		FROM budgets b
		JOIN categories c ON c.id = b.category_id
		WHERE b.id = $1 AND b.user_id = $2
	`

	var budget model.Budget
	dbCtx, dbSpan := startDBSpan(ctx, "select", attribute.String("db.table", "budgets"))
	setDBStatement(dbSpan, query)
	err := r.db.QueryRow(dbCtx, query, id, userID).Scan(
		&budget.ID,
		&budget.UserID,
		&budget.CategoryID,
		&budget.CategoryName,
		&budget.Year,
		&budget.Month,
		&budget.AmountCents,
		&budget.SpentCents,
		&budget.CreatedAt,
		&budget.UpdatedAt,
	)
	defer dbSpan.End()
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			span.AddEvent("budget_not_found")
			logger.Info().Int64("user_id", userID).Int64("budget_id", id).Msg("repository budget get by id not found")
			return nil, nil
		}
		markSpanError(dbSpan, err, "select budget failed")
		markSpanError(span, err, "get budget failed")
		logger.Error().Err(err).Int64("user_id", userID).Int64("budget_id", id).Msg("repository budget get by id failed")
		return nil, err
	}

	logger.Info().Int64("user_id", userID).Int64("budget_id", budget.ID).Msg("repository budget get by id completed")
	return &budget, nil
}

func (r *BudgetRepository) ListByUser(ctx context.Context, userID int64, year, month int) ([]model.Budget, error) {
	ctx, span := startRepositorySpan(ctx, "repository.budget.list_by_user",
		attribute.Int64("app.user.id", userID),
	)
	defer span.End()

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int("year", year).Int("month", month).Msg("repository budget list by user started")
	query := `
		SELECT
			b.id,
			b.user_id,
			b.category_id,
			c.name,
			b.year,
			b.month,
			b.amount_cents,
			COALESCE(spent.total_spent, 0) AS spent_cents,
			b.created_at,
			b.updated_at
		FROM budgets b
		JOIN categories c ON c.id = b.category_id
		LEFT JOIN (
			SELECT category_id, SUM(amount_cents) AS total_spent
			FROM transactions
			WHERE user_id = $1
				AND type = 'expense'
				AND EXTRACT(YEAR FROM transaction_date) = $2
				AND EXTRACT(MONTH FROM transaction_date) = $3
			GROUP BY category_id
		) spent ON spent.category_id = b.category_id
		WHERE b.user_id = $1 AND b.year = $2 AND b.month = $3
		ORDER BY c.name ASC
	`

	dbCtx, dbSpan := startDBSpan(ctx, "select", attribute.String("db.table", "budgets"))
	setDBStatement(dbSpan, query)
	rows, err := r.db.Query(dbCtx, query, userID, year, month)
	if err != nil {
		markSpanError(dbSpan, err, "query budgets failed")
		dbSpan.End()
		markSpanError(span, err, "list budgets failed")
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository budget list by user query failed")
		return nil, err
	}
	defer rows.Close()
	defer dbSpan.End()

	items := make([]model.Budget, 0)
	for rows.Next() {
		var item model.Budget
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.CategoryID,
			&item.CategoryName,
			&item.Year,
			&item.Month,
			&item.AmountCents,
			&item.SpentCents,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			markSpanError(dbSpan, err, "scan budgets failed")
			markSpanError(span, err, "scan budgets failed")
			logger.Error().Err(err).Int64("user_id", userID).Msg("repository budget list by user scan failed")
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		markSpanError(dbSpan, err, "rows budgets failed")
		markSpanError(span, err, "rows budgets failed")
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository budget list by user rows failed")
		return nil, err
	}
	dbSpan.SetAttributes(attribute.Int("app.budget.count", len(items)))
	logger.Info().Int64("user_id", userID).Int("count", len(items)).Msg("repository budget list by user completed")
	return items, nil
}

func (r *BudgetRepository) Update(ctx context.Context, budget *model.Budget) error {
	ctx, span := startRepositorySpan(ctx, "repository.budget.update",
		attribute.Int64("app.user.id", budget.UserID),
		attribute.Int64("app.budget.id", budget.ID),
	)
	defer span.End()

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", budget.UserID).Int64("budget_id", budget.ID).Msg("repository budget update started")
	query := `
		UPDATE budgets
		SET category_id = $1,
			year = $2,
			month = $3,
			amount_cents = $4,
			updated_at = NOW()
		WHERE id = $5 AND user_id = $6
		RETURNING updated_at
	`

	dbCtx, dbSpan := startDBSpan(ctx, "update", attribute.String("db.table", "budgets"))
	setDBStatement(dbSpan, query)
	err := r.db.QueryRow(
		dbCtx,
		query,
		budget.CategoryID,
		budget.Year,
		budget.Month,
		budget.AmountCents,
		budget.ID,
		budget.UserID,
	).Scan(&budget.UpdatedAt)
	if err != nil {
		markSpanError(dbSpan, err, "update budget failed")
	}
	dbSpan.End()
	if err != nil {
		markSpanError(span, err, "update budget failed")
		logger.Error().Err(err).Int64("user_id", budget.UserID).Int64("budget_id", budget.ID).Msg("repository budget update failed")
		return err
	}
	logger.Info().Int64("user_id", budget.UserID).Int64("budget_id", budget.ID).Msg("repository budget update completed")
	return nil
}

func (r *BudgetRepository) Delete(ctx context.Context, id, userID int64) (bool, error) {
	ctx, span := startRepositorySpan(ctx, "repository.budget.delete",
		attribute.Int64("app.user.id", userID),
		attribute.Int64("app.budget.id", id),
	)
	defer span.End()

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("budget_id", id).Msg("repository budget delete started")
	dbCtx, dbSpan := startDBSpan(ctx, "delete", attribute.String("db.table", "budgets"))
	statement := `DELETE FROM budgets WHERE id = $1 AND user_id = $2`
	setDBStatement(dbSpan, statement)
	result, err := r.db.Exec(dbCtx, statement, id, userID)
	if err != nil {
		markSpanError(dbSpan, err, "delete budget failed")
		dbSpan.End()
		markSpanError(span, err, "delete budget failed")
		logger.Error().Err(err).Int64("user_id", userID).Int64("budget_id", id).Msg("repository budget delete failed")
		return false, err
	}
	dbSpan.SetAttributes(attribute.Int64("db.rows_affected", result.RowsAffected()))
	dbSpan.End()
	deleted := result.RowsAffected() > 0
	logger.Info().Int64("user_id", userID).Int64("budget_id", id).Bool("deleted", deleted).Msg("repository budget delete completed")
	return deleted, nil
}
