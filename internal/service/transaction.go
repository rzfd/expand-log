package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/apperror"
	"github.com/rzfd/expand/internal/pkg/logging"
)

type transactionRepository interface {
	Create(ctx context.Context, transaction *model.Transaction) error
	GetByIDForUser(ctx context.Context, id, userID int64) (*model.Transaction, error)
	ListByUser(ctx context.Context, userID int64, filter model.TransactionFilter) ([]model.Transaction, int, error)
	Update(ctx context.Context, transaction *model.Transaction) error
	Delete(ctx context.Context, id, userID int64) (bool, error)
	HasRecentManualTransaction(ctx context.Context, userID int64, since time.Time) (bool, error)
}

type categoryLookupRepository interface {
	GetByIDForUser(ctx context.Context, id, userID int64) (*model.Category, error)
}

type TransactionInput struct {
	CategoryID      int64
	Type            model.TransactionType
	AmountCents     int64
	Note            string
	TransactionDate time.Time
}

type TransactionService struct {
	transactions transactionRepository
	categories   categoryLookupRepository
}

func NewTransactionService(transactions transactionRepository, categories categoryLookupRepository) *TransactionService {
	return &TransactionService{
		transactions: transactions,
		categories:   categories,
	}
}

func (s *TransactionService) Create(ctx context.Context, userID int64, input TransactionInput) (*model.Transaction, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Msg("service transaction create started")
	category, err := s.validateTransactionInput(ctx, userID, input)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("service transaction create validation failed")
		return nil, err
	}
	cutoff := currentUTC().Add(-time.Minute)
	hasRecent, err := s.transactions.HasRecentManualTransaction(ctx, userID, cutoff)
	if err != nil {
		logger.Error().Err(err).Int64("user_id", userID).Msg("service transaction create rate limit check failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to validate transaction rate limit", err)
	}
	if hasRecent {
		logger.Warn().Int64("user_id", userID).Msg("service transaction create rate limited")
		return nil, newRateLimitError("you can only create one manual transaction per minute")
	}

	transaction := model.Transaction{
		UserID:          userID,
		CategoryID:      input.CategoryID,
		CategoryName:    category.Name,
		Type:            input.Type,
		AmountCents:     input.AmountCents,
		Note:            strings.TrimSpace(input.Note),
		TransactionDate: input.TransactionDate.UTC(),
		Source:          "manual",
	}

	if err := s.transactions.Create(ctx, &transaction); err != nil {
		logger.Error().Err(err).Int64("user_id", userID).Msg("service transaction create repository failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to create transaction", err)
	}

	logger.Info().Int64("user_id", userID).Int64("transaction_id", transaction.ID).Msg("service transaction create completed")
	return s.transactions.GetByIDForUser(ctx, transaction.ID, userID)
}

func (s *TransactionService) GetByID(ctx context.Context, userID, transactionID int64) (*model.Transaction, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("transaction_id", transactionID).Msg("service transaction get by id started")
	transaction, err := s.transactions.GetByIDForUser(ctx, transactionID, userID)
	if err != nil {
		logger.Error().Err(err).Msg("service transaction get by id repository failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to load transaction", err)
	}
	if transaction == nil {
		logger.Warn().Msg("service transaction get by id not found")
		return nil, apperror.New(http.StatusNotFound, "not_found", "transaction not found")
	}
	logger.Info().Int64("user_id", userID).Int64("transaction_id", transaction.ID).Msg("service transaction get by id completed")
	return transaction, nil
}

func (s *TransactionService) List(ctx context.Context, userID int64, filter model.TransactionFilter) ([]model.Transaction, int, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Msg("service transaction list started")
	items, total, err := s.transactions.ListByUser(ctx, userID, filter)
	if err != nil {
		logger.Error().Err(err).Msg("service transaction list repository failed")
		return nil, 0, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to list transactions", err)
	}
	logger.Info().Int64("user_id", userID).Int("count", len(items)).Int("total", total).Msg("service transaction list completed")
	return items, total, nil
}

func (s *TransactionService) Update(ctx context.Context, userID, transactionID int64, input TransactionInput) (*model.Transaction, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("transaction_id", transactionID).Msg("service transaction update started")
	existing, err := s.transactions.GetByIDForUser(ctx, transactionID, userID)
	if err != nil {
		logger.Error().Err(err).Msg("service transaction update load failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to load transaction", err)
	}
	if existing == nil {
		logger.Warn().Msg("service transaction update not found")
		return nil, apperror.New(http.StatusNotFound, "not_found", "transaction not found")
	}
	if existing.Source == "recurring" {
		logger.Warn().Int64("user_id", userID).Int64("transaction_id", transactionID).Msg("service transaction update blocked for recurring source")
		return nil, newValidationError("recurring-generated transactions cannot be edited manually")
	}

	category, err := s.validateTransactionInput(ctx, userID, input)
	if err != nil {
		logger.Warn().Err(err).Msg("service transaction update validation failed")
		return nil, err
	}

	existing.CategoryID = input.CategoryID
	existing.CategoryName = category.Name
	existing.Type = input.Type
	existing.AmountCents = input.AmountCents
	existing.Note = strings.TrimSpace(input.Note)
	existing.TransactionDate = input.TransactionDate.UTC()

	if err := s.transactions.Update(ctx, existing); err != nil {
		logger.Error().Err(err).Msg("service transaction update repository failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to update transaction", err)
	}

	logger.Info().Int64("user_id", userID).Int64("transaction_id", transactionID).Msg("service transaction update completed")
	return s.transactions.GetByIDForUser(ctx, transactionID, userID)
}

func (s *TransactionService) Delete(ctx context.Context, userID, transactionID int64) error {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("transaction_id", transactionID).Msg("service transaction delete started")
	deleted, err := s.transactions.Delete(ctx, transactionID, userID)
	if err != nil {
		logger.Error().Err(err).Msg("service transaction delete repository failed")
		return apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to delete transaction", err)
	}
	if !deleted {
		logger.Warn().Msg("service transaction delete not found")
		return apperror.New(http.StatusNotFound, "not_found", "transaction not found")
	}
	logger.Info().Int64("user_id", userID).Int64("transaction_id", transactionID).Msg("service transaction delete completed")
	return nil
}

func (s *TransactionService) validateTransactionInput(ctx context.Context, userID int64, input TransactionInput) (*model.Category, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("category_id", input.CategoryID).Msg("service transaction validate input started")
	if input.CategoryID <= 0 {
		logger.Warn().Msg("service transaction validate input invalid category id")
		return nil, apperror.New(http.StatusBadRequest, "validation_error", "category_id must be greater than zero")
	}
	if !input.Type.IsValid() {
		logger.Warn().Str("type", string(input.Type)).Msg("service transaction validate input invalid type")
		return nil, apperror.New(http.StatusBadRequest, "validation_error", "type must be either income or expense")
	}
	if input.AmountCents <= 0 {
		logger.Warn().Int64("amount_cents", input.AmountCents).Msg("service transaction validate input invalid amount")
		return nil, newValidationError("amount must be greater than zero")
	}
	if err := validateAmountBounds(input.AmountCents); err != nil {
		logger.Warn().Err(err).Int64("amount_cents", input.AmountCents).Msg("service transaction validate input amount out of range")
		return nil, err
	}
	if input.TransactionDate.IsZero() {
		logger.Warn().Msg("service transaction validate input missing transaction date")
		return nil, newValidationError("transaction_date is required")
	}
	if err := validateNoteLength(input.Note, maxTransactionNoteLength, "note"); err != nil {
		logger.Warn().Err(err).Msg("service transaction validate input note too long")
		return nil, err
	}
	now := currentUTC()
	if input.TransactionDate.UTC().After(now.AddDate(0, 0, maxTransactionFutureDays)) {
		logger.Warn().Time("transaction_date", input.TransactionDate).Msg("service transaction validate input future date too far")
		return nil, newValidationError("transaction_date is too far in the future")
	}
	if input.TransactionDate.UTC().Before(now.AddDate(-maxTransactionPastYears, 0, 0)) {
		logger.Warn().Time("transaction_date", input.TransactionDate).Msg("service transaction validate input past date too far")
		return nil, newValidationError("transaction_date is too far in the past")
	}

	category, err := s.categories.GetByIDForUser(ctx, input.CategoryID, userID)
	if err != nil {
		logger.Error().Err(err).Msg("service transaction validate input category lookup failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to load category", err)
	}
	if category == nil {
		logger.Warn().Msg("service transaction validate input category not found")
		return nil, apperror.New(http.StatusBadRequest, "validation_error", "category not found")
	}
	if category.Type != input.Type {
		logger.Warn().Str("category_type", string(category.Type)).Str("input_type", string(input.Type)).Msg("service transaction validate input type mismatch")
		return nil, apperror.New(http.StatusBadRequest, "validation_error", "transaction type must match category type")
	}

	logger.Info().Int64("category_id", category.ID).Msg("service transaction validate input completed")
	return category, nil
}
