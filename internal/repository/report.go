package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/logging"
)

type ReportRepository struct {
	db *pgxpool.Pool
}

func NewReportRepository(db *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) GetMonthlyTotals(ctx context.Context, userID int64, start, end string) (int64, int64, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Str("start", start).Str("end", end).Msg("repository report monthly totals started")
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount_cents ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount_cents ELSE 0 END), 0)
		FROM transactions
		WHERE user_id = $1 AND transaction_date >= $2 AND transaction_date < $3
	`

	var income, expense int64
	if err := r.db.QueryRow(ctx, query, userID, start, end).Scan(&income, &expense); err != nil {
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository report monthly totals failed")
		return 0, 0, err
	}

	logger.Info().Int64("user_id", userID).Int64("income_cents", income).Int64("expense_cents", expense).Msg("repository report monthly totals completed")
	return income, expense, nil
}

func (r *ReportRepository) GetMonthlySpendingByCategory(ctx context.Context, userID int64, start, end string) ([]model.CategorySpending, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Str("start", start).Str("end", end).Msg("repository report monthly spending started")
	query := `
		SELECT c.id, c.name, COALESCE(SUM(t.amount_cents), 0) AS amount_cents
		FROM transactions t
		JOIN categories c ON c.id = t.category_id
		WHERE t.user_id = $1
			AND t.type = 'expense'
			AND t.transaction_date >= $2
			AND t.transaction_date < $3
		GROUP BY c.id, c.name
		ORDER BY amount_cents DESC, c.name ASC
	`

	rows, err := r.db.Query(ctx, query, userID, start, end)
	if err != nil {
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository report monthly spending query failed")
		return nil, err
	}
	defer rows.Close()

	items := make([]model.CategorySpending, 0)
	for rows.Next() {
		var item model.CategorySpending
		if err := rows.Scan(&item.CategoryID, &item.CategoryName, &item.AmountCents); err != nil {
			logger.Error().Err(err).Int64("user_id", userID).Msg("repository report monthly spending scan failed")
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository report monthly spending rows failed")
		return nil, err
	}
	logger.Info().Int64("user_id", userID).Int("count", len(items)).Msg("repository report monthly spending completed")
	return items, nil
}

func (r *ReportRepository) GetRecentTransactions(ctx context.Context, userID int64, limit int) ([]model.Transaction, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int("limit", limit).Msg("repository report recent transactions started")
	query := `
		SELECT
			t.id,
			t.user_id,
			t.category_id,
			c.name,
			t.type,
			t.amount_cents,
			t.note,
			t.transaction_date,
			t.source,
			t.recurring_transaction_id,
			t.created_at,
			t.updated_at
		FROM transactions t
		JOIN categories c ON c.id = t.category_id
		WHERE t.user_id = $1
		ORDER BY t.transaction_date DESC, t.id DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository report recent transactions query failed")
		return nil, err
	}
	defer rows.Close()

	transactions := make([]model.Transaction, 0)
	for rows.Next() {
		transaction, err := scanTransaction(rows)
		if err != nil {
			logger.Error().Err(err).Int64("user_id", userID).Msg("repository report recent transactions scan failed")
			return nil, err
		}
		transactions = append(transactions, transaction)
	}

	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository report recent transactions rows failed")
		return nil, err
	}
	logger.Info().Int64("user_id", userID).Int("count", len(transactions)).Msg("repository report recent transactions completed")
	return transactions, nil
}
