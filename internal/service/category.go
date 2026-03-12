package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/apperror"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/repository"
)

type categoryRepository interface {
	Create(ctx context.Context, category *model.Category) error
	ListByUser(ctx context.Context, userID int64) ([]model.Category, error)
	GetByIDForUser(ctx context.Context, id, userID int64) (*model.Category, error)
	Update(ctx context.Context, category *model.Category) error
	Delete(ctx context.Context, id, userID int64) (bool, error)
}

type CategoryService struct {
	categories categoryRepository
}

func NewCategoryService(categories categoryRepository) *CategoryService {
	return &CategoryService{categories: categories}
}

func (s *CategoryService) Create(ctx context.Context, userID int64, name string, transactionType model.TransactionType) (*model.Category, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Msg("service category create started")
	name = strings.TrimSpace(name)
	if err := validateCategoryInput(name, transactionType); err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("service category create validation failed")
		return nil, err
	}

	category := model.Category{
		UserID: userID,
		Name:   name,
		Type:   transactionType,
	}

	if err := s.categories.Create(ctx, &category); err != nil {
		logger.Error().Err(err).Int64("user_id", userID).Msg("service category create repository failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to create category", err)
	}

	logger.Info().Int64("user_id", userID).Int64("category_id", category.ID).Msg("service category create completed")
	return &category, nil
}

func (s *CategoryService) List(ctx context.Context, userID int64) ([]model.Category, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Msg("service category list started")
	items, err := s.categories.ListByUser(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Int64("user_id", userID).Msg("service category list failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to list categories", err)
	}
	logger.Info().Int64("user_id", userID).Int("count", len(items)).Msg("service category list completed")
	return items, nil
}

func (s *CategoryService) Update(ctx context.Context, userID, categoryID int64, name string, transactionType model.TransactionType) (*model.Category, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("category_id", categoryID).Msg("service category update started")
	name = strings.TrimSpace(name)
	if err := validateCategoryInput(name, transactionType); err != nil {
		logger.Warn().Err(err).Msg("service category update validation failed")
		return nil, err
	}

	category, err := s.categories.GetByIDForUser(ctx, categoryID, userID)
	if err != nil {
		logger.Error().Err(err).Msg("service category update load failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to load category", err)
	}
	if category == nil {
		logger.Warn().Msg("service category update category not found")
		return nil, apperror.New(http.StatusNotFound, "not_found", "category not found")
	}

	category.Name = name
	category.Type = transactionType
	if err := s.categories.Update(ctx, category); err != nil {
		logger.Error().Err(err).Msg("service category update repository failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to update category", err)
	}

	logger.Info().Int64("user_id", userID).Int64("category_id", category.ID).Msg("service category update completed")
	return category, nil
}

func (s *CategoryService) Delete(ctx context.Context, userID, categoryID int64) error {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("category_id", categoryID).Msg("service category delete started")
	deleted, err := s.categories.Delete(ctx, categoryID, userID)
	if err != nil {
		if repository.IsForeignKeyViolation(err) {
			logger.Warn().Err(err).Msg("service category delete category in use")
			return apperror.New(http.StatusConflict, "category_in_use", "category cannot be deleted while it is still referenced")
		}
		logger.Error().Err(err).Msg("service category delete repository failed")
		return apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to delete category", err)
	}
	if !deleted {
		logger.Warn().Msg("service category delete category not found")
		return apperror.New(http.StatusNotFound, "not_found", "category not found")
	}
	logger.Info().Int64("user_id", userID).Int64("category_id", categoryID).Msg("service category delete completed")
	return nil
}

func validateCategoryInput(name string, transactionType model.TransactionType) error {
	logger := logging.FromContext(nil)
	logger.Info().Msg("service category validate input started")
	if name == "" {
		logger.Warn().Msg("service category validate input empty name")
		return apperror.New(http.StatusBadRequest, "validation_error", "name is required")
	}
	if len(name) > 100 {
		logger.Warn().Msg("service category validate input name too long")
		return apperror.New(http.StatusBadRequest, "validation_error", "name must be at most 100 characters")
	}
	if !transactionType.IsValid() {
		logger.Warn().Msg("service category validate input invalid type")
		return apperror.New(http.StatusBadRequest, "validation_error", "type must be either income or expense")
	}
	logger.Info().Msg("service category validate input completed")
	return nil
}
