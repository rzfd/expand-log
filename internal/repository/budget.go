package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

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
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", budget.UserID).Msg("repository budget create started")
	query := `
		INSERT INTO budgets (user_id, category_id, year, month, amount_cents)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		budget.UserID,
		budget.CategoryID,
		budget.Year,
		budget.Month,
		budget.AmountCents,
	).Scan(&budget.ID, &budget.CreatedAt, &budget.UpdatedAt)
	if err != nil {
		logger.Error().Err(err).Int64("user_id", budget.UserID).Msg("repository budget create failed")
		return err
	}
	logger.Info().Int64("user_id", budget.UserID).Int64("budget_id", budget.ID).Msg("repository budget create completed")
	return nil
}

func (r *BudgetRepository) GetByIDForUser(ctx context.Context, id, userID int64) (*model.Budget, error) {
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
	err := r.db.QueryRow(ctx, query, id, userID).Scan(
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
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Info().Int64("user_id", userID).Int64("budget_id", id).Msg("repository budget get by id not found")
			return nil, nil
		}
		logger.Error().Err(err).Int64("user_id", userID).Int64("budget_id", id).Msg("repository budget get by id failed")
		return nil, err
	}

	logger.Info().Int64("user_id", userID).Int64("budget_id", budget.ID).Msg("repository budget get by id completed")
	return &budget, nil
}

func (r *BudgetRepository) ListByUser(ctx context.Context, userID int64, year, month int) ([]model.Budget, error) {
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

	rows, err := r.db.Query(ctx, query, userID, year, month)
	if err != nil {
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository budget list by user query failed")
		return nil, err
	}
	defer rows.Close()

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
			logger.Error().Err(err).Int64("user_id", userID).Msg("repository budget list by user scan failed")
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository budget list by user rows failed")
		return nil, err
	}
	logger.Info().Int64("user_id", userID).Int("count", len(items)).Msg("repository budget list by user completed")
	return items, nil
}

func (r *BudgetRepository) Update(ctx context.Context, budget *model.Budget) error {
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

	err := r.db.QueryRow(
		ctx,
		query,
		budget.CategoryID,
		budget.Year,
		budget.Month,
		budget.AmountCents,
		budget.ID,
		budget.UserID,
	).Scan(&budget.UpdatedAt)
	if err != nil {
		logger.Error().Err(err).Int64("user_id", budget.UserID).Int64("budget_id", budget.ID).Msg("repository budget update failed")
		return err
	}
	logger.Info().Int64("user_id", budget.UserID).Int64("budget_id", budget.ID).Msg("repository budget update completed")
	return nil
}

func (r *BudgetRepository) Delete(ctx context.Context, id, userID int64) (bool, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("budget_id", id).Msg("repository budget delete started")
	result, err := r.db.Exec(ctx, `DELETE FROM budgets WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		logger.Error().Err(err).Int64("user_id", userID).Int64("budget_id", id).Msg("repository budget delete failed")
		return false, err
	}
	deleted := result.RowsAffected() > 0
	logger.Info().Int64("user_id", userID).Int64("budget_id", id).Bool("deleted", deleted).Msg("repository budget delete completed")
	return deleted, nil
}
