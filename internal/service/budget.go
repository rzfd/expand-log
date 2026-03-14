package service

import (
	"context"
	"net/http"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/apperror"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/repository"
)

type budgetRepository interface {
	Create(ctx context.Context, budget *model.Budget) error
	GetByIDForUser(ctx context.Context, id, userID int64) (*model.Budget, error)
	ListByUser(ctx context.Context, userID int64, year, month int) ([]model.Budget, error)
	Update(ctx context.Context, budget *model.Budget) error
	Delete(ctx context.Context, id, userID int64) (bool, error)
}

type BudgetInput struct {
	CategoryID  int64
	Year        int
	Month       int
	AmountCents int64
}

type BudgetService struct {
	budgets    budgetRepository
	categories categoryLookupRepository
}

func NewBudgetService(budgets budgetRepository, categories categoryLookupRepository) *BudgetService {
	return &BudgetService{
		budgets:    budgets,
		categories: categories,
	}
}

func (s *BudgetService) Create(ctx context.Context, userID int64, input BudgetInput) (*model.Budget, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Msg("service budget create started")
	category, err := s.validateBudgetInput(ctx, userID, input)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("service budget create validation failed")
		return nil, err
	}

	item := model.Budget{
		UserID:       userID,
		CategoryID:   input.CategoryID,
		CategoryName: category.Name,
		Year:         input.Year,
		Month:        input.Month,
		AmountCents:  input.AmountCents,
	}

	if err := s.budgets.Create(ctx, &item); err != nil {
		if repository.IsUniqueViolation(err) {
			logger.Warn().Err(err).Msg("service budget create unique violation")
			return nil, apperror.New(http.StatusConflict, "budget_exists", "budget already exists for that category and month")
		}
		logger.Error().Err(err).Msg("service budget create repository failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to create budget", err)
	}

	logger.Info().Int64("user_id", userID).Int64("budget_id", item.ID).Msg("service budget create completed")
	return s.budgets.GetByIDForUser(ctx, item.ID, userID)
}

func (s *BudgetService) List(ctx context.Context, userID int64, year, month int) ([]model.Budget, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int("year", year).Int("month", month).Msg("service budget list started")
	if year == 0 || month == 0 {
		now := currentUTC()
		if year == 0 {
			year = now.Year()
		}
		if month == 0 {
			month = int(now.Month())
		}
	}

	if month < 1 || month > 12 || !isBudgetYearAllowed(year, currentUTC()) {
		logger.Warn().Int("year", year).Int("month", month).Msg("service budget list invalid year month")
		return nil, apperror.New(http.StatusBadRequest, "validation_error", "year and month must be valid")
	}

	items, err := s.budgets.ListByUser(ctx, userID, year, month)
	if err != nil {
		logger.Error().Err(err).Msg("service budget list repository failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to list budgets", err)
	}
	logger.Info().Int64("user_id", userID).Int("count", len(items)).Int("year", year).Int("month", month).Msg("service budget list completed")
	return items, nil
}

func (s *BudgetService) Update(ctx context.Context, userID, budgetID int64, input BudgetInput) (*model.Budget, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("budget_id", budgetID).Msg("service budget update started")
	existing, err := s.budgets.GetByIDForUser(ctx, budgetID, userID)
	if err != nil {
		logger.Error().Err(err).Msg("service budget update load failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to load budget", err)
	}
	if existing == nil {
		logger.Warn().Msg("service budget update budget not found")
		return nil, apperror.New(http.StatusNotFound, "not_found", "budget not found")
	}

	category, err := s.validateBudgetInput(ctx, userID, input)
	if err != nil {
		logger.Warn().Err(err).Msg("service budget update validation failed")
		return nil, err
	}

	existing.CategoryID = input.CategoryID
	existing.CategoryName = category.Name
	existing.Year = input.Year
	existing.Month = input.Month
	existing.AmountCents = input.AmountCents

	if err := s.budgets.Update(ctx, existing); err != nil {
		if repository.IsUniqueViolation(err) {
			logger.Warn().Err(err).Msg("service budget update unique violation")
			return nil, apperror.New(http.StatusConflict, "budget_exists", "budget already exists for that category and month")
		}
		logger.Error().Err(err).Msg("service budget update repository failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to update budget", err)
	}

	logger.Info().Int64("user_id", userID).Int64("budget_id", budgetID).Msg("service budget update completed")
	return s.budgets.GetByIDForUser(ctx, budgetID, userID)
}

func (s *BudgetService) Delete(ctx context.Context, userID, budgetID int64) error {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("budget_id", budgetID).Msg("service budget delete started")
	deleted, err := s.budgets.Delete(ctx, budgetID, userID)
	if err != nil {
		logger.Error().Err(err).Msg("service budget delete repository failed")
		return apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to delete budget", err)
	}
	if !deleted {
		logger.Warn().Msg("service budget delete budget not found")
		return apperror.New(http.StatusNotFound, "not_found", "budget not found")
	}
	logger.Info().Int64("user_id", userID).Int64("budget_id", budgetID).Msg("service budget delete completed")
	return nil
}

func (s *BudgetService) validateBudgetInput(ctx context.Context, userID int64, input BudgetInput) (*model.Category, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("category_id", input.CategoryID).Msg("service budget validate input started")
	if input.CategoryID <= 0 {
		logger.Warn().Msg("service budget validate input invalid category id")
		return nil, apperror.New(http.StatusBadRequest, "validation_error", "category_id must be greater than zero")
	}
	if input.Month < 1 || input.Month > 12 || !isBudgetYearAllowed(input.Year, currentUTC()) {
		logger.Warn().Int("year", input.Year).Int("month", input.Month).Msg("service budget validate input invalid year month")
		return nil, apperror.New(http.StatusBadRequest, "validation_error", "year must be within the allowed budgeting window")
	}
	if err := validateAmountBounds(input.AmountCents); err != nil {
		logger.Warn().Err(err).Int64("amount_cents", input.AmountCents).Msg("service budget validate input invalid amount")
		return nil, err
	}

	category, err := s.categories.GetByIDForUser(ctx, input.CategoryID, userID)
	if err != nil {
		logger.Error().Err(err).Msg("service budget validate input category lookup failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to load category", err)
	}
	if category == nil {
		logger.Warn().Msg("service budget validate input category not found")
		return nil, apperror.New(http.StatusBadRequest, "validation_error", "category not found")
	}
	if category.Type != model.TransactionTypeExpense {
		logger.Warn().Str("category_type", string(category.Type)).Msg("service budget validate input category not expense")
		return nil, apperror.New(http.StatusBadRequest, "validation_error", "budget category must be an expense category")
	}

	logger.Info().Int64("category_id", category.ID).Msg("service budget validate input completed")
	return category, nil
}
