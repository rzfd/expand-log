package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/apperror"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/money"
	"github.com/rzfd/expand/internal/pkg/pagination"
	"github.com/rzfd/expand/internal/pkg/schedule"
)

const dateLayout = "2006-01-02"

func parseIDParam(c echo.Context, name string) (int64, error) {
	_, span := handlerTracer.Start(c.Request().Context(), "handler.parse_id_param")
	defer span.End()
	span.SetAttributes(attribute.String("app.param.name", name))

	logger := logging.FromContext(c.Request().Context())
	value := c.Param(name)
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "invalid id param")
		logger.Warn().Err(err).Str("param", name).Str("value", value).Msg("parse id param failed")
		return 0, apperror.New(http.StatusBadRequest, "validation_error", fmt.Sprintf("%s must be a positive integer", name))
	}
	span.SetAttributes(attribute.Int64("app.param.value", id))
	logger.Info().Str("param", name).Int64("value", id).Msg("parse id param completed")
	return id, nil
}

func parseOptionalDate(ctx context.Context, value, field string) (*time.Time, error) {
	ctx, span := handlerTracer.Start(ctx, "handler.parse_optional_date")
	defer span.End()
	span.SetAttributes(attribute.String("app.field", field))

	logger := logging.FromContext(ctx)
	value = strings.TrimSpace(value)
	if value == "" {
		logger.Info().Str("field", field).Msg("parse optional date empty")
		return nil, nil
	}

	parsed, err := time.Parse(dateLayout, value)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid date format")
		logger.Warn().Err(err).Str("field", field).Str("value", value).Msg("parse optional date failed")
		return nil, apperror.New(http.StatusBadRequest, "validation_error", fmt.Sprintf("%s must use YYYY-MM-DD format", field))
	}

	normalized := schedule.NormalizeDate(parsed)
	span.SetAttributes(attribute.String("app.date.value", normalized.Format(dateLayout)))
	logger.Info().Str("field", field).Msg("parse optional date completed")
	return &normalized, nil
}

func parseRequiredDate(ctx context.Context, value, field string) (time.Time, error) {
	ctx, span := handlerTracer.Start(ctx, "handler.parse_required_date")
	defer span.End()
	span.SetAttributes(attribute.String("app.field", field))

	parsed, err := parseOptionalDate(ctx, value, field)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "parse required date failed")
		return time.Time{}, err
	}
	if parsed == nil {
		span.SetStatus(codes.Error, "required date missing")
		logging.FromContext(ctx).Warn().Str("field", field).Msg("parse required date missing")
		return time.Time{}, apperror.New(http.StatusBadRequest, "validation_error", fmt.Sprintf("%s is required", field))
	}
	span.SetAttributes(attribute.String("app.date.value", parsed.Format(dateLayout)))
	return *parsed, nil
}

func parseAmount(ctx context.Context, value, field string) (int64, error) {
	ctx, span := handlerTracer.Start(ctx, "handler.parse_amount")
	defer span.End()
	span.SetAttributes(attribute.String("app.field", field))

	logger := logging.FromContext(ctx)
	parsed, err := money.ParseDecimal(value)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "parse amount failed")
		logger.Warn().Err(err).Str("field", field).Str("value", value).Msg("parse amount failed")
		return 0, apperror.New(http.StatusBadRequest, "validation_error", fmt.Sprintf("%s: %s", field, err.Error()))
	}
	span.SetAttributes(attribute.Int64("app.amount_cents", parsed))
	logger.Info().Str("field", field).Int64("amount_cents", parsed).Msg("parse amount completed")
	return parsed, nil
}

func parseOptionalAmount(ctx context.Context, value, field string) (*int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := parseAmount(ctx, value, field)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseTransactionType(ctx context.Context, value string) (*model.TransactionType, error) {
	logger := logging.FromContext(ctx)
	value = strings.TrimSpace(value)
	if value == "" {
		logger.Info().Msg("parse transaction type empty")
		return nil, nil
	}
	transactionType := model.TransactionType(value)
	if !transactionType.IsValid() {
		logger.Warn().Str("value", value).Msg("parse transaction type failed")
		return nil, apperror.New(http.StatusBadRequest, "validation_error", "type must be either income or expense")
	}
	logger.Info().Str("type", string(transactionType)).Msg("parse transaction type completed")
	return &transactionType, nil
}

func parseYearMonth(c echo.Context) (int, int, error) {
	ctx, span := handlerTracer.Start(c.Request().Context(), "handler.parse_year_month")
	defer span.End()

	logger := logging.FromContext(c.Request().Context())
	yearRaw := strings.TrimSpace(c.QueryParam("year"))
	monthRaw := strings.TrimSpace(c.QueryParam("month"))
	year, err := parseOptionalInt(ctx, yearRaw)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "parse year failed")
		logger.Warn().Err(err).Str("year", yearRaw).Msg("parse year failed")
		return 0, 0, apperror.New(http.StatusBadRequest, "validation_error", "year must be a valid integer")
	}
	month, err := parseOptionalInt(ctx, monthRaw)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "parse month failed")
		logger.Warn().Err(err).Str("month", monthRaw).Msg("parse month failed")
		return 0, 0, apperror.New(http.StatusBadRequest, "validation_error", "month must be a valid integer")
	}
	if yearRaw != "" && (year < 2000 || year > 9999) {
		span.SetStatus(codes.Error, "year out of range")
		logger.Warn().Int("year", year).Msg("parse year out of range")
		return 0, 0, apperror.New(http.StatusBadRequest, "validation_error", "year must be between 2000 and 9999")
	}
	if monthRaw != "" && (month < 1 || month > 12) {
		span.SetStatus(codes.Error, "month out of range")
		logger.Warn().Int("month", month).Msg("parse month out of range")
		return 0, 0, apperror.New(http.StatusBadRequest, "validation_error", "month must be between 1 and 12")
	}
	span.SetAttributes(
		attribute.Int("app.year", year),
		attribute.Int("app.month", month),
	)
	logger.Info().Int("year", year).Int("month", month).Msg("parse year month completed")
	return year, month, nil
}

func parseOptionalInt64(ctx context.Context, value string, field string) (*int64, error) {
	logger := logging.FromContext(ctx)
	value = strings.TrimSpace(value)
	if value == "" {
		logger.Info().Str("field", field).Msg("parse optional int64 empty")
		return nil, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		logger.Warn().Err(err).Str("field", field).Str("value", value).Msg("parse optional int64 failed")
		return nil, apperror.New(http.StatusBadRequest, "validation_error", fmt.Sprintf("%s must be a valid integer", field))
	}
	logger.Info().Str("field", field).Int64("value", parsed).Msg("parse optional int64 completed")
	return &parsed, nil
}

func parseOptionalInt(ctx context.Context, value string) (int, error) {
	logger := logging.FromContext(ctx)
	value = strings.TrimSpace(value)
	if value == "" {
		logger.Info().Msg("parse optional int empty")
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		logger.Warn().Err(err).Str("value", value).Msg("parse optional int failed")
		return 0, err
	}
	logger.Info().Int("value", parsed).Msg("parse optional int completed")
	return parsed, nil
}

func parsePagination(c echo.Context) (pagination.Params, error) {
	_, span := handlerTracer.Start(c.Request().Context(), "handler.parse_pagination")
	defer span.End()

	logger := logging.FromContext(c.Request().Context())
	pageRaw := strings.TrimSpace(c.QueryParam("page"))
	pageSizeRaw := strings.TrimSpace(c.QueryParam("page_size"))

	if pageRaw != "" {
		page, err := strconv.Atoi(pageRaw)
		if err != nil || page <= 0 {
			if err != nil {
				span.RecordError(err)
			}
			span.SetStatus(codes.Error, "invalid page")
			logger.Warn().Str("page", pageRaw).Err(err).Msg("parse pagination invalid page")
			return pagination.Params{}, apperror.New(http.StatusBadRequest, "validation_error", "page must be a positive integer")
		}
	}
	if pageSizeRaw != "" {
		pageSize, err := strconv.Atoi(pageSizeRaw)
		if err != nil || pageSize <= 0 {
			if err != nil {
				span.RecordError(err)
			}
			span.SetStatus(codes.Error, "invalid page size")
			logger.Warn().Str("page_size", pageSizeRaw).Err(err).Msg("parse pagination invalid page size")
			return pagination.Params{}, apperror.New(http.StatusBadRequest, "validation_error", "page_size must be a positive integer")
		}
		if pageSize > 100 {
			span.SetStatus(codes.Error, "page size too large")
			logger.Warn().Int("page_size", pageSize).Msg("parse pagination page size too large")
			return pagination.Params{}, apperror.New(http.StatusBadRequest, "validation_error", "page_size must be less than or equal to 100")
		}
	}

	params := pagination.Parse(pageRaw, pageSizeRaw)
	span.SetAttributes(
		attribute.Int("app.page", params.Page),
		attribute.Int("app.page_size", params.PageSize),
	)
	logger.
		Info().
		Int("page", params.Page).
		Int("page_size", params.PageSize).
		Msg("parse pagination completed")
	return params, nil
}
