package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	authmiddleware "github.com/rzfd/expand/internal/middleware"
	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/response"
	"github.com/rzfd/expand/internal/service"
)

type RecurringHandler struct {
	items *service.RecurringService
}

type recurringRequest struct {
	CategoryID int64   `json:"category_id"`
	Type       string  `json:"type"`
	Amount     string  `json:"amount"`
	Note       string  `json:"note"`
	Frequency  string  `json:"frequency"`
	StartDate  string  `json:"start_date"`
	EndDate    *string `json:"end_date"`
	Active     *bool   `json:"active"`
}

func NewRecurringHandler(items *service.RecurringService) *RecurringHandler {
	return &RecurringHandler{items: items}
}

func (h *RecurringHandler) List(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("recurring list started")
	userID, ok := authmiddleware.UserIDFromContext(c)
	if !ok {
		logger.Warn().Msg("recurring list unauthorized")
		return unauthorized(c)
	}

	items, err := h.items.List(c.Request().Context(), userID)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("recurring list failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", userID).Int("count", len(items)).Msg("recurring list completed")
	return response.OK(c, newRecurringResponses(items))
}

func (h *RecurringHandler) Create(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("recurring create started")
	userID, ok := authmiddleware.UserIDFromContext(c)
	if !ok {
		logger.Warn().Msg("recurring create unauthorized")
		return unauthorized(c)
	}

	input, err := h.parseRequest(c)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("recurring create invalid request")
		return response.Error(c, err)
	}

	item, err := h.items.Create(c.Request().Context(), userID, input)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("recurring create failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", userID).Int64("recurring_id", item.ID).Msg("recurring create completed")
	return response.Created(c, newRecurringResponse(*item))
}

func (h *RecurringHandler) Update(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("recurring update started")
	userID, ok := authmiddleware.UserIDFromContext(c)
	if !ok {
		logger.Warn().Msg("recurring update unauthorized")
		return unauthorized(c)
	}

	id, err := parseIDParam(c, "id")
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("recurring update invalid id")
		return response.Error(c, err)
	}

	input, err := h.parseRequest(c)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Int64("recurring_id", id).Msg("recurring update invalid request")
		return response.Error(c, err)
	}

	item, err := h.items.Update(c.Request().Context(), userID, id, input)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Int64("recurring_id", id).Msg("recurring update failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", userID).Int64("recurring_id", item.ID).Msg("recurring update completed")
	return response.OK(c, newRecurringResponse(*item))
}

func (h *RecurringHandler) Delete(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("recurring delete started")
	userID, ok := authmiddleware.UserIDFromContext(c)
	if !ok {
		logger.Warn().Msg("recurring delete unauthorized")
		return unauthorized(c)
	}

	id, err := parseIDParam(c, "id")
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("recurring delete invalid id")
		return response.Error(c, err)
	}

	if err := h.items.Delete(c.Request().Context(), userID, id); err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Int64("recurring_id", id).Msg("recurring delete failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", userID).Int64("recurring_id", id).Msg("recurring delete completed")
	return c.NoContent(http.StatusNoContent)
}

func (h *RecurringHandler) parseRequest(c echo.Context) (service.RecurringInput, error) {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("recurring parse request started")
	var request recurringRequest
	if err := c.Bind(&request); err != nil {
		logger.Warn().Err(err).Msg("recurring parse request bind failed")
		return service.RecurringInput{}, badRequestBody()
	}

	amount, err := parseAmount(request.Amount, "amount")
	if err != nil {
		logger.Warn().Err(err).Msg("recurring parse request invalid amount")
		return service.RecurringInput{}, err
	}
	startDate, err := parseRequiredDate(request.StartDate, "start_date")
	if err != nil {
		logger.Warn().Err(err).Msg("recurring parse request invalid start date")
		return service.RecurringInput{}, err
	}

	var endDate *time.Time
	if request.EndDate != nil {
		endDate, err = parseOptionalDate(*request.EndDate, "end_date")
		if err != nil {
			logger.Warn().Err(err).Msg("recurring parse request invalid end date")
			return service.RecurringInput{}, err
		}
	}

	active := true
	if request.Active != nil {
		active = *request.Active
	}

	result := service.RecurringInput{
		CategoryID:  request.CategoryID,
		Type:        model.TransactionType(request.Type),
		AmountCents: amount,
		Note:        request.Note,
		Frequency:   model.RecurringFrequency(request.Frequency),
		StartDate:   startDate,
		EndDate:     endDate,
		Active:      active,
	}
	logger.Info().Bool("active", result.Active).Msg("recurring parse request completed")
	return result, nil
}
