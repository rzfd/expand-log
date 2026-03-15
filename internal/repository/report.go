package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"

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
	ctx, span := startRepositorySpan(ctx, "repository.report.get_monthly_totals",
		attribute.Int64("app.user.id", userID),
	)
	defer span.End()

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
	dbCtx, dbSpan := startDBSpan(ctx, "select", attribute.String("db.table", "transactions"))
	setDBStatement(dbSpan, query)
	if err := r.db.QueryRow(dbCtx, query, userID, start, end).Scan(&income, &expense); err != nil {
		markSpanError(dbSpan, err, "query monthly totals failed")
		dbSpan.End()
		markSpanError(span, err, "get monthly totals failed")
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository report monthly totals failed")
		return 0, 0, err
	}
	dbSpan.SetAttributes(
		attribute.Int64("app.report.income_cents", income),
		attribute.Int64("app.report.expense_cents", expense),
	)
	dbSpan.End()

	logger.Info().Int64("user_id", userID).Int64("income_cents", income).Int64("expense_cents", expense).Msg("repository report monthly totals completed")
	return income, expense, nil
}

func (r *ReportRepository) GetMonthlySpendingByCategory(ctx context.Context, userID int64, start, end string) ([]model.CategorySpending, error) {
	ctx, span := startRepositorySpan(ctx, "repository.report.get_monthly_spending_by_category",
		attribute.Int64("app.user.id", userID),
	)
	defer span.End()

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

	dbCtx, dbSpan := startDBSpan(ctx, "select", attribute.String("db.table", "transactions"))
	setDBStatement(dbSpan, query)
	rows, err := r.db.Query(dbCtx, query, userID, start, end)
	if err != nil {
		markSpanError(dbSpan, err, "query monthly spending failed")
		dbSpan.End()
		markSpanError(span, err, "get monthly spending failed")
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository report monthly spending query failed")
		return nil, err
	}
	defer rows.Close()
	defer dbSpan.End()

	items := make([]model.CategorySpending, 0)
	for rows.Next() {
		var item model.CategorySpending
		if err := rows.Scan(&item.CategoryID, &item.CategoryName, &item.AmountCents); err != nil {
			markSpanError(dbSpan, err, "scan monthly spending failed")
			markSpanError(span, err, "scan monthly spending failed")
			logger.Error().Err(err).Int64("user_id", userID).Msg("repository report monthly spending scan failed")
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		markSpanError(dbSpan, err, "rows monthly spending failed")
		markSpanError(span, err, "rows monthly spending failed")
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository report monthly spending rows failed")
		return nil, err
	}
	dbSpan.SetAttributes(attribute.Int("app.report.category_count", len(items)))
	logger.Info().Int64("user_id", userID).Int("count", len(items)).Msg("repository report monthly spending completed")
	return items, nil
}

func (r *ReportRepository) GetRecentTransactions(ctx context.Context, userID int64, limit int) ([]model.Transaction, error) {
	ctx, span := startRepositorySpan(ctx, "repository.report.get_recent_transactions",
		attribute.Int64("app.user.id", userID),
	)
	defer span.End()

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

	dbCtx, dbSpan := startDBSpan(ctx, "select", attribute.String("db.table", "transactions"))
	setDBStatement(dbSpan, query)
	rows, err := r.db.Query(dbCtx, query, userID, limit)
	if err != nil {
		markSpanError(dbSpan, err, "query recent transactions failed")
		dbSpan.End()
		markSpanError(span, err, "get recent transactions failed")
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository report recent transactions query failed")
		return nil, err
	}
	defer rows.Close()
	defer dbSpan.End()

	transactions := make([]model.Transaction, 0)
	for rows.Next() {
		transaction, err := scanTransaction(rows)
		if err != nil {
			markSpanError(dbSpan, err, "scan recent transactions failed")
			markSpanError(span, err, "scan recent transactions failed")
			logger.Error().Err(err).Int64("user_id", userID).Msg("repository report recent transactions scan failed")
			return nil, err
		}
		transactions = append(transactions, transaction)
	}

	if err := rows.Err(); err != nil {
		markSpanError(dbSpan, err, "rows recent transactions failed")
		markSpanError(span, err, "rows recent transactions failed")
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository report recent transactions rows failed")
		return nil, err
	}
	dbSpan.SetAttributes(attribute.Int("app.transaction.count", len(transactions)))
	logger.Info().Int64("user_id", userID).Int("count", len(transactions)).Msg("repository report recent transactions completed")
	return transactions, nil
}
