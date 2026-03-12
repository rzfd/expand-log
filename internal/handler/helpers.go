package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/apperror"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/money"
	"github.com/rzfd/expand/internal/pkg/pagination"
	"github.com/rzfd/expand/internal/pkg/schedule"
)

const dateLayout = "2006-01-02"

func parseIDParam(c echo.Context, name string) (int64, error) {
	logger := logging.FromContext(c.Request().Context())
	value := c.Param(name)
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		logger.Warn().Err(err).Str("param", name).Str("value", value).Msg("parse id param failed")
		return 0, apperror.New(http.StatusBadRequest, "validation_error", fmt.Sprintf("%s must be a positive integer", name))
	}
	logger.Info().Str("param", name).Int64("value", id).Msg("parse id param completed")
	return id, nil
}

func parseOptionalDate(value, field string) (*time.Time, error) {
	logger := logging.FromContext(nil)
	value = strings.TrimSpace(value)
	if value == "" {
		logger.Info().Str("field", field).Msg("parse optional date empty")
		return nil, nil
	}

	parsed, err := time.Parse(dateLayout, value)
	if err != nil {
		logger.Warn().Err(err).Str("field", field).Str("value", value).Msg("parse optional date failed")
		return nil, apperror.New(http.StatusBadRequest, "validation_error", fmt.Sprintf("%s must use YYYY-MM-DD format", field))
	}

	normalized := schedule.NormalizeDate(parsed)
	logger.Info().Str("field", field).Msg("parse optional date completed")
	return &normalized, nil
}

func parseRequiredDate(value, field string) (time.Time, error) {
	parsed, err := parseOptionalDate(value, field)
	if err != nil {
		return time.Time{}, err
	}
	if parsed == nil {
		logging.FromContext(nil).Warn().Str("field", field).Msg("parse required date missing")
		return time.Time{}, apperror.New(http.StatusBadRequest, "validation_error", fmt.Sprintf("%s is required", field))
	}
	return *parsed, nil
}

func parseAmount(value, field string) (int64, error) {
	logger := logging.FromContext(nil)
	parsed, err := money.ParseDecimal(value)
	if err != nil {
		logger.Warn().Err(err).Str("field", field).Str("value", value).Msg("parse amount failed")
		return 0, apperror.New(http.StatusBadRequest, "validation_error", fmt.Sprintf("%s: %s", field, err.Error()))
	}
	logger.Info().Str("field", field).Int64("amount_cents", parsed).Msg("parse amount completed")
	return parsed, nil
}

func parseOptionalAmount(value, field string) (*int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := parseAmount(value, field)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseTransactionType(value string) (*model.TransactionType, error) {
	logger := logging.FromContext(nil)
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
	logger := logging.FromContext(c.Request().Context())
	year, err := parseOptionalInt(c.QueryParam("year"))
	if err != nil {
		logger.Warn().Err(err).Str("year", c.QueryParam("year")).Msg("parse year failed")
		return 0, 0, apperror.New(http.StatusBadRequest, "validation_error", "year must be a valid integer")
	}
	month, err := parseOptionalInt(c.QueryParam("month"))
	if err != nil {
		logger.Warn().Err(err).Str("month", c.QueryParam("month")).Msg("parse month failed")
		return 0, 0, apperror.New(http.StatusBadRequest, "validation_error", "month must be a valid integer")
	}
	logger.Info().Int("year", year).Int("month", month).Msg("parse year month completed")
	return year, month, nil
}

func parseOptionalInt64(value string, field string) (*int64, error) {
	logger := logging.FromContext(nil)
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

func parseOptionalInt(value string) (int, error) {
	logger := logging.FromContext(nil)
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

func parsePagination(c echo.Context) pagination.Params {
	params := pagination.Parse(c.QueryParam("page"), c.QueryParam("page_size"))
	logging.FromContext(c.Request().Context()).
		Info().
		Int("page", params.Page).
		Int("page_size", params.PageSize).
		Msg("parse pagination completed")
	return params
}
