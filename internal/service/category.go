package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/apperror"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var categoryTracer = otel.Tracer("service.category")

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
	ctx, span := categoryTracer.Start(ctx, "service.category.create")
	defer span.End()
	span.SetAttributes(attribute.Int64("app.user.id", userID))

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Msg("service category create started")
	name = strings.TrimSpace(name)
	if err := validateCategoryInput(name, transactionType); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "validation failed")
		logger.Warn().Err(err).Int64("user_id", userID).Msg("service category create validation failed")
		return nil, err
	}

	category := model.Category{
		UserID: userID,
		Name:   name,
		Type:   transactionType,
	}

	if err := s.categories.Create(ctx, &category); err != nil {
		if repository.IsUniqueViolation(err) {
			span.RecordError(err)
			span.SetStatus(codes.Error, "category exists")
			logger.Warn().Err(err).Int64("user_id", userID).Msg("service category create duplicate name")
			return nil, apperror.New(http.StatusConflict, "category_exists", "category already exists for this type")
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "repository create failed")
		logger.Error().Err(err).Int64("user_id", userID).Msg("service category create repository failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to create category", err)
	}

	span.SetAttributes(attribute.Int64("app.category.id", category.ID))
	logger.Info().Int64("user_id", userID).Int64("category_id", category.ID).Msg("service category create completed")
	return &category, nil
}

func (s *CategoryService) List(ctx context.Context, userID int64) ([]model.Category, error) {
	ctx, span := categoryTracer.Start(ctx, "service.category.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("app.user.id", userID))

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Msg("service category list started")
	items, err := s.categories.ListByUser(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "repository list failed")
		logger.Error().Err(err).Int64("user_id", userID).Msg("service category list failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to list categories", err)
	}
	span.SetAttributes(attribute.Int("app.category.count", len(items)))
	logger.Info().Int64("user_id", userID).Int("count", len(items)).Msg("service category list completed")
	return items, nil
}

func (s *CategoryService) Update(ctx context.Context, userID, categoryID int64, name string, transactionType model.TransactionType) (*model.Category, error) {
	ctx, span := categoryTracer.Start(ctx, "service.category.update")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("app.user.id", userID),
		attribute.Int64("app.category.id", categoryID),
	)

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("category_id", categoryID).Msg("service category update started")
	name = strings.TrimSpace(name)
	if err := validateCategoryInput(name, transactionType); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "validation failed")
		logger.Warn().Err(err).Msg("service category update validation failed")
		return nil, err
	}

	category, err := s.categories.GetByIDForUser(ctx, categoryID, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "load category failed")
		logger.Error().Err(err).Msg("service category update load failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to load category", err)
	}
	if category == nil {
		span.AddEvent("category_not_found")
		span.SetStatus(codes.Error, "category not found")
		logger.Warn().Msg("service category update category not found")
		return nil, apperror.New(http.StatusNotFound, "not_found", "category not found")
	}

	category.Name = name
	category.Type = transactionType
	if err := s.categories.Update(ctx, category); err != nil {
		if repository.IsUniqueViolation(err) {
			span.RecordError(err)
			span.SetStatus(codes.Error, "category exists")
			logger.Warn().Err(err).Int64("user_id", userID).Int64("category_id", categoryID).Msg("service category update duplicate name")
			return nil, apperror.New(http.StatusConflict, "category_exists", "category already exists for this type")
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "repository update failed")
		logger.Error().Err(err).Msg("service category update repository failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to update category", err)
	}

	logger.Info().Int64("user_id", userID).Int64("category_id", category.ID).Msg("service category update completed")
	return category, nil
}

func (s *CategoryService) Delete(ctx context.Context, userID, categoryID int64) error {
	ctx, span := categoryTracer.Start(ctx, "service.category.delete")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("app.user.id", userID),
		attribute.Int64("app.category.id", categoryID),
	)

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("category_id", categoryID).Msg("service category delete started")
	deleted, err := s.categories.Delete(ctx, categoryID, userID)
	if err != nil {
		if repository.IsForeignKeyViolation(err) {
			span.RecordError(err)
			span.SetStatus(codes.Error, "category in use")
			logger.Warn().Err(err).Msg("service category delete category in use")
			return apperror.New(http.StatusConflict, "category_in_use", "category cannot be deleted while it is still referenced")
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "repository delete failed")
		logger.Error().Err(err).Msg("service category delete repository failed")
		return apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to delete category", err)
	}
	if !deleted {
		span.AddEvent("category_not_found")
		span.SetStatus(codes.Error, "category not found")
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
		return newValidationError("name is required")
	}
	if len(name) < minCategoryNameLength {
		logger.Warn().Msg("service category validate input name too short")
		return newValidationError("name must be at least 2 characters")
	}
	if len(name) > maxCategoryNameLength {
		logger.Warn().Msg("service category validate input name too long")
		return newValidationError("name must be at most 50 characters")
	}
	reserved := map[string]struct{}{
		"all":     {},
		"default": {},
		"misc":    {},
	}
	if _, blocked := reserved[strings.ToLower(name)]; blocked {
		logger.Warn().Msg("service category validate input reserved name")
		return newValidationError("name is reserved")
	}
	if !transactionType.IsValid() {
		logger.Warn().Msg("service category validate input invalid type")
		return newValidationError("type must be either income or expense")
	}
	logger.Info().Msg("service category validate input completed")
	return nil
}
