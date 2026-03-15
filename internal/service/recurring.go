package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/apperror"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/schedule"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var recurringTracer = otel.Tracer("service.recurring")

type recurringRepository interface {
	Create(ctx context.Context, item *model.RecurringTransaction) error
	GetByIDForUser(ctx context.Context, id, userID int64) (*model.RecurringTransaction, error)
	ListByUser(ctx context.Context, userID int64) ([]model.RecurringTransaction, error)
	Update(ctx context.Context, item *model.RecurringTransaction) error
	Delete(ctx context.Context, id, userID int64) (bool, error)
}

type RecurringInput struct {
	CategoryID  int64
	Type        model.TransactionType
	AmountCents int64
	Note        string
	Frequency   model.RecurringFrequency
	StartDate   time.Time
	EndDate     *time.Time
	Active      bool
}

type RecurringService struct {
	items      recurringRepository
	categories categoryLookupRepository
}

func NewRecurringService(items recurringRepository, categories categoryLookupRepository) *RecurringService {
	return &RecurringService{
		items:      items,
		categories: categories,
	}
}

func (s *RecurringService) Create(ctx context.Context, userID int64, input RecurringInput) (*model.RecurringTransaction, error) {
	ctx, span := recurringTracer.Start(ctx, "service.recurring.create")
	defer span.End()
	span.SetAttributes(attribute.Int64("app.user.id", userID))

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Msg("service recurring create started")
	category, nextRunDate, active, err := s.validateRecurringInput(ctx, userID, nil, input)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "validation failed")
		logger.Warn().Err(err).Msg("service recurring create validation failed")
		return nil, err
	}

	item := model.RecurringTransaction{
		UserID:       userID,
		CategoryID:   input.CategoryID,
		CategoryName: category.Name,
		Type:         input.Type,
		AmountCents:  input.AmountCents,
		Note:         strings.TrimSpace(input.Note),
		Frequency:    input.Frequency,
		StartDate:    schedule.NormalizeDate(input.StartDate),
		EndDate:      normalizeOptionalDate(input.EndDate),
		NextRunDate:  nextRunDate,
		Active:       active,
	}

	if err := s.items.Create(ctx, &item); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "repository create failed")
		logger.Error().Err(err).Msg("service recurring create repository failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to create recurring transaction", err)
	}

	span.SetAttributes(attribute.Int64("app.recurring.id", item.ID))
	logger.Info().Int64("user_id", userID).Int64("recurring_id", item.ID).Bool("active", item.Active).Msg("service recurring create completed")
	return &item, nil
}

func (s *RecurringService) List(ctx context.Context, userID int64) ([]model.RecurringTransaction, error) {
	ctx, span := recurringTracer.Start(ctx, "service.recurring.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("app.user.id", userID))

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Msg("service recurring list started")
	items, err := s.items.ListByUser(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "repository list failed")
		logger.Error().Err(err).Msg("service recurring list failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to list recurring transactions", err)
	}
	span.SetAttributes(attribute.Int("app.recurring.count", len(items)))
	logger.Info().Int64("user_id", userID).Int("count", len(items)).Msg("service recurring list completed")
	return items, nil
}

func (s *RecurringService) Update(ctx context.Context, userID, itemID int64, input RecurringInput) (*model.RecurringTransaction, error) {
	ctx, span := recurringTracer.Start(ctx, "service.recurring.update")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("app.user.id", userID),
		attribute.Int64("app.recurring.id", itemID),
	)

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("recurring_id", itemID).Msg("service recurring update started")
	existing, err := s.items.GetByIDForUser(ctx, itemID, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "load recurring failed")
		logger.Error().Err(err).Msg("service recurring update load failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to load recurring transaction", err)
	}
	if existing == nil {
		span.AddEvent("recurring_not_found")
		span.SetStatus(codes.Error, "recurring transaction not found")
		logger.Warn().Msg("service recurring update not found")
		return nil, apperror.New(http.StatusNotFound, "not_found", "recurring transaction not found")
	}

	category, nextRunDate, active, err := s.validateRecurringInput(ctx, userID, existing, input)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "validation failed")
		logger.Warn().Err(err).Msg("service recurring update validation failed")
		return nil, err
	}

	existing.CategoryID = input.CategoryID
	existing.CategoryName = category.Name
	existing.Type = input.Type
	existing.AmountCents = input.AmountCents
	existing.Note = strings.TrimSpace(input.Note)
	existing.Frequency = input.Frequency
	existing.StartDate = schedule.NormalizeDate(input.StartDate)
	existing.EndDate = normalizeOptionalDate(input.EndDate)
	existing.NextRunDate = nextRunDate
	existing.Active = active

	if err := s.items.Update(ctx, existing); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "repository update failed")
		logger.Error().Err(err).Msg("service recurring update repository failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to update recurring transaction", err)
	}

	logger.Info().Int64("user_id", userID).Int64("recurring_id", itemID).Bool("active", existing.Active).Msg("service recurring update completed")
	return s.items.GetByIDForUser(ctx, itemID, userID)
}

func (s *RecurringService) Delete(ctx context.Context, userID, itemID int64) error {
	ctx, span := recurringTracer.Start(ctx, "service.recurring.delete")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("app.user.id", userID),
		attribute.Int64("app.recurring.id", itemID),
	)

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("recurring_id", itemID).Msg("service recurring delete started")
	deleted, err := s.items.Delete(ctx, itemID, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "repository delete failed")
		logger.Error().Err(err).Msg("service recurring delete repository failed")
		return apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to delete recurring transaction", err)
	}
	if !deleted {
		span.AddEvent("recurring_not_found")
		span.SetStatus(codes.Error, "recurring transaction not found")
		logger.Warn().Msg("service recurring delete not found")
		return apperror.New(http.StatusNotFound, "not_found", "recurring transaction not found")
	}
	logger.Info().Int64("user_id", userID).Int64("recurring_id", itemID).Msg("service recurring delete completed")
	return nil
}

func (s *RecurringService) validateRecurringInput(ctx context.Context, userID int64, existing *model.RecurringTransaction, input RecurringInput) (*model.Category, *time.Time, bool, error) {
	ctx, span := recurringTracer.Start(ctx, "service.recurring.validate_input")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("app.user.id", userID),
		attribute.Int64("app.category.id", input.CategoryID),
	)

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("category_id", input.CategoryID).Msg("service recurring validate input started")
	if input.CategoryID <= 0 {
		span.SetStatus(codes.Error, "invalid category id")
		logger.Warn().Msg("service recurring validate input invalid category id")
		return nil, nil, false, apperror.New(http.StatusBadRequest, "validation_error", "category_id must be greater than zero")
	}
	if !input.Type.IsValid() {
		span.SetStatus(codes.Error, "invalid type")
		logger.Warn().Str("type", string(input.Type)).Msg("service recurring validate input invalid type")
		return nil, nil, false, apperror.New(http.StatusBadRequest, "validation_error", "type must be either income or expense")
	}
	if !input.Frequency.IsValid() {
		span.SetStatus(codes.Error, "invalid frequency")
		logger.Warn().Str("frequency", string(input.Frequency)).Msg("service recurring validate input invalid frequency")
		return nil, nil, false, apperror.New(http.StatusBadRequest, "validation_error", "frequency must be daily, weekly, or monthly")
	}
	if input.AmountCents <= 0 {
		span.SetStatus(codes.Error, "invalid amount")
		logger.Warn().Int64("amount_cents", input.AmountCents).Msg("service recurring validate input invalid amount")
		return nil, nil, false, newValidationError("amount must be greater than zero")
	}
	if err := validateAmountBounds(input.AmountCents); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "amount out of range")
		logger.Warn().Err(err).Int64("amount_cents", input.AmountCents).Msg("service recurring validate input amount out of range")
		return nil, nil, false, err
	}
	if input.StartDate.IsZero() {
		span.SetStatus(codes.Error, "missing start date")
		logger.Warn().Msg("service recurring validate input missing start date")
		return nil, nil, false, newValidationError("start_date is required")
	}
	if err := validateNoteLength(input.Note, maxRecurringNoteLength, "note"); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "note too long")
		logger.Warn().Err(err).Msg("service recurring validate input note too long")
		return nil, nil, false, err
	}

	startDate := schedule.NormalizeDate(input.StartDate)
	now := schedule.NormalizeDate(currentUTC())
	if startDate.Before(now.AddDate(-maxRecurringPastYears, 0, 0)) {
		span.SetStatus(codes.Error, "start date too far in past")
		logger.Warn().Time("start_date", startDate).Msg("service recurring validate input start date too far in past")
		return nil, nil, false, newValidationError("start_date is too far in the past")
	}
	endDate := normalizeOptionalDate(input.EndDate)
	if endDate != nil && endDate.Before(startDate) {
		span.SetStatus(codes.Error, "invalid end date")
		logger.Warn().Msg("service recurring validate input invalid end date")
		return nil, nil, false, newValidationError("end_date must be on or after start_date")
	}
	if endDate != nil && endDate.After(startDate.AddDate(maxRecurringFutureYears, 0, 0)) {
		span.SetStatus(codes.Error, "end date too far in future")
		logger.Warn().Time("end_date", *endDate).Msg("service recurring validate input end date too far in future")
		return nil, nil, false, newValidationError("end_date is too far in the future")
	}

	category, err := s.categories.GetByIDForUser(ctx, input.CategoryID, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "category lookup failed")
		logger.Error().Err(err).Msg("service recurring validate input category lookup failed")
		return nil, nil, false, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to load category", err)
	}
	if category == nil {
		span.SetAttributes(attribute.Bool("app.category.found", false))
		span.SetStatus(codes.Error, "category not found")
		logger.Warn().Msg("service recurring validate input category not found")
		return nil, nil, false, apperror.New(http.StatusBadRequest, "validation_error", "category not found")
	}
	span.SetAttributes(attribute.Bool("app.category.found", true))
	if category.Type != input.Type {
		span.SetStatus(codes.Error, "category type mismatch")
		logger.Warn().Str("category_type", string(category.Type)).Str("input_type", string(input.Type)).Msg("service recurring validate input type mismatch")
		return nil, nil, false, apperror.New(http.StatusBadRequest, "validation_error", "recurring transaction type must match category type")
	}

	if !input.Active {
		logger.Info().Int64("category_id", category.ID).Msg("service recurring validate input completed inactive")
		return category, nil, false, nil
	}

	if existing != nil && existing.NextRunDate != nil {
		if input.Active && startDate.After(schedule.NormalizeDate(*existing.NextRunDate)) {
			span.SetStatus(codes.Error, "retroactive change blocked")
			logger.Warn().Time("start_date", startDate).Time("existing_next_run_date", *existing.NextRunDate).Msg("service recurring validate input retroactive change blocked")
			return nil, nil, false, newValidationError("start_date cannot be after current next_run_date for active recurring transaction")
		}
		candidate := schedule.NormalizeDate(*existing.NextRunDate)
		if candidate.Before(startDate) {
			candidate = startDate
		}
		if endDate != nil && candidate.After(*endDate) {
			logger.Info().Int64("category_id", category.ID).Msg("service recurring validate input completed next run after end date")
			return category, nil, false, nil
		}
		logger.Info().Int64("category_id", category.ID).Msg("service recurring validate input completed with existing next run")
		return category, &candidate, true, nil
	}

	logger.Info().Int64("category_id", category.ID).Msg("service recurring validate input completed")
	return category, &startDate, true, nil
}

func normalizeOptionalDate(input *time.Time) *time.Time {
	if input == nil {
		logging.FromContext(nil).Info().Msg("service recurring normalize optional date empty")
		return nil
	}
	normalized := schedule.NormalizeDate(*input)
	logging.FromContext(nil).Info().Msg("service recurring normalize optional date completed")
	return &normalized
}
