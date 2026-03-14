package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/logging"
)

type TransactionRepository struct {
	db *pgxpool.Pool
}

func NewTransactionRepository(db *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(ctx context.Context, transaction *model.Transaction) error {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", transaction.UserID).Msg("repository transaction create started")
	query := `
		INSERT INTO transactions (
			user_id, category_id, recurring_transaction_id, type, amount_cents, note, transaction_date, source
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		transaction.UserID,
		transaction.CategoryID,
		transaction.RecurringTransactionID,
		transaction.Type,
		transaction.AmountCents,
		transaction.Note,
		transaction.TransactionDate,
		transaction.Source,
	).Scan(&transaction.ID, &transaction.CreatedAt, &transaction.UpdatedAt)
	if err != nil {
		logger.Error().Err(err).Int64("user_id", transaction.UserID).Msg("repository transaction create failed")
		return err
	}
	logger.Info().Int64("user_id", transaction.UserID).Int64("transaction_id", transaction.ID).Msg("repository transaction create completed")
	return nil
}

func (r *TransactionRepository) GetByIDForUser(ctx context.Context, id, userID int64) (*model.Transaction, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("transaction_id", id).Msg("repository transaction get by id started")
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
		WHERE t.id = $1 AND t.user_id = $2
	`

	var transaction model.Transaction
	var recurringID sql.NullInt64
	err := r.db.QueryRow(ctx, query, id, userID).Scan(
		&transaction.ID,
		&transaction.UserID,
		&transaction.CategoryID,
		&transaction.CategoryName,
		&transaction.Type,
		&transaction.AmountCents,
		&transaction.Note,
		&transaction.TransactionDate,
		&transaction.Source,
		&recurringID,
		&transaction.CreatedAt,
		&transaction.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Info().Int64("user_id", userID).Int64("transaction_id", id).Msg("repository transaction get by id not found")
			return nil, nil
		}
		logger.Error().Err(err).Int64("user_id", userID).Int64("transaction_id", id).Msg("repository transaction get by id failed")
		return nil, err
	}

	if recurringID.Valid {
		transaction.RecurringTransactionID = &recurringID.Int64
	}

	logger.Info().Int64("user_id", userID).Int64("transaction_id", transaction.ID).Msg("repository transaction get by id completed")
	return &transaction, nil
}

func (r *TransactionRepository) ListByUser(ctx context.Context, userID int64, filter model.TransactionFilter) ([]model.Transaction, int, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Msg("repository transaction list by user started")
	whereClause, args, nextIndex := buildTransactionFilters(userID, filter)

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM transactions t WHERE %s`, whereClause)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository transaction list count failed")
		return nil, 0, err
	}

	args = append(args, filter.PageSize, filter.Offset)
	listQuery := fmt.Sprintf(`
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
		WHERE %s
		ORDER BY t.transaction_date DESC, t.id DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, nextIndex, nextIndex+1)

	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository transaction list query failed")
		return nil, 0, err
	}
	defer rows.Close()

	transactions := make([]model.Transaction, 0)
	for rows.Next() {
		transaction, err := scanTransaction(rows)
		if err != nil {
			logger.Error().Err(err).Int64("user_id", userID).Msg("repository transaction list scan failed")
			return nil, 0, err
		}
		transactions = append(transactions, transaction)
	}

	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository transaction list rows failed")
		return nil, 0, err
	}
	logger.Info().Int64("user_id", userID).Int("count", len(transactions)).Int("total", total).Msg("repository transaction list by user completed")
	return transactions, total, nil
}

func (r *TransactionRepository) Update(ctx context.Context, transaction *model.Transaction) error {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", transaction.UserID).Int64("transaction_id", transaction.ID).Msg("repository transaction update started")
	query := `
		UPDATE transactions
		SET category_id = $1,
			type = $2,
			amount_cents = $3,
			note = $4,
			transaction_date = $5,
			updated_at = NOW()
		WHERE id = $6 AND user_id = $7
		RETURNING updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		transaction.CategoryID,
		transaction.Type,
		transaction.AmountCents,
		transaction.Note,
		transaction.TransactionDate,
		transaction.ID,
		transaction.UserID,
	).Scan(&transaction.UpdatedAt)
	if err != nil {
		logger.Error().Err(err).Int64("user_id", transaction.UserID).Int64("transaction_id", transaction.ID).Msg("repository transaction update failed")
		return err
	}
	logger.Info().Int64("user_id", transaction.UserID).Int64("transaction_id", transaction.ID).Msg("repository transaction update completed")
	return nil
}

func (r *TransactionRepository) Delete(ctx context.Context, id, userID int64) (bool, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("transaction_id", id).Msg("repository transaction delete started")
	result, err := r.db.Exec(ctx, `DELETE FROM transactions WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		logger.Error().Err(err).Int64("user_id", userID).Int64("transaction_id", id).Msg("repository transaction delete failed")
		return false, err
	}
	deleted := result.RowsAffected() > 0
	logger.Info().Int64("user_id", userID).Int64("transaction_id", id).Bool("deleted", deleted).Msg("repository transaction delete completed")
	return deleted, nil
}

func (r *TransactionRepository) HasRecentManualTransaction(ctx context.Context, userID int64, since time.Time) (bool, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Time("since", since).Msg("repository transaction recent manual check started")
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM transactions
			WHERE user_id = $1
				AND source = 'manual'
				AND created_at >= $2
		)
	`
	var exists bool
	if err := r.db.QueryRow(ctx, query, userID, since).Scan(&exists); err != nil {
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository transaction recent manual check failed")
		return false, err
	}
	logger.Info().Int64("user_id", userID).Bool("exists", exists).Msg("repository transaction recent manual check completed")
	return exists, nil
}

func buildTransactionFilters(userID int64, filter model.TransactionFilter) (string, []any, int) {
	logging.FromContext(nil).Info().Int64("user_id", userID).Msg("repository transaction build filters started")
	clauses := []string{"t.user_id = $1"}
	args := []any{userID}
	index := 2

	if filter.StartDate != nil {
		clauses = append(clauses, fmt.Sprintf("t.transaction_date >= $%d", index))
		args = append(args, *filter.StartDate)
		index++
	}
	if filter.EndDate != nil {
		clauses = append(clauses, fmt.Sprintf("t.transaction_date <= $%d", index))
		args = append(args, *filter.EndDate)
		index++
	}
	if filter.Type != nil {
		clauses = append(clauses, fmt.Sprintf("t.type = $%d", index))
		args = append(args, *filter.Type)
		index++
	}
	if filter.CategoryID != nil {
		clauses = append(clauses, fmt.Sprintf("t.category_id = $%d", index))
		args = append(args, *filter.CategoryID)
		index++
	}
	if filter.MinAmountCents != nil {
		clauses = append(clauses, fmt.Sprintf("t.amount_cents >= $%d", index))
		args = append(args, *filter.MinAmountCents)
		index++
	}
	if filter.MaxAmountCents != nil {
		clauses = append(clauses, fmt.Sprintf("t.amount_cents <= $%d", index))
		args = append(args, *filter.MaxAmountCents)
		index++
	}

	whereClause := strings.Join(clauses, " AND ")
	logging.FromContext(nil).Info().Int64("user_id", userID).Int("arg_count", len(args)).Msg("repository transaction build filters completed")
	return whereClause, args, index
}

func scanTransaction(scanner interface {
	Scan(dest ...any) error
}) (model.Transaction, error) {
	logging.FromContext(nil).Info().Msg("repository transaction scan started")
	var transaction model.Transaction
	var recurringID sql.NullInt64

	err := scanner.Scan(
		&transaction.ID,
		&transaction.UserID,
		&transaction.CategoryID,
		&transaction.CategoryName,
		&transaction.Type,
		&transaction.AmountCents,
		&transaction.Note,
		&transaction.TransactionDate,
		&transaction.Source,
		&recurringID,
		&transaction.CreatedAt,
		&transaction.UpdatedAt,
	)
	if err != nil {
		logging.FromContext(nil).Warn().Err(err).Msg("repository transaction scan failed")
		return model.Transaction{}, err
	}

	if recurringID.Valid {
		transaction.RecurringTransactionID = &recurringID.Int64
	}

	logging.FromContext(nil).Info().Int64("transaction_id", transaction.ID).Msg("repository transaction scan completed")
	return transaction, nil
}
