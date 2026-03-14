package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	authmiddleware "github.com/rzfd/expand/internal/middleware"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/response"
	"github.com/rzfd/expand/internal/service"
)

type BudgetHandler struct {
	budgets *service.BudgetService
}

type budgetRequest struct {
	CategoryID int64  `json:"category_id"`
	Year       int    `json:"year"`
	Month      int    `json:"month"`
	Amount     string `json:"amount"`
}

func NewBudgetHandler(budgets *service.BudgetService) *BudgetHandler {
	return &BudgetHandler{budgets: budgets}
}

func (h *BudgetHandler) List(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("budget list started")
	userID, ok := authmiddleware.UserIDFromContext(c)
	if !ok {
		logger.Warn().Msg("budget list unauthorized")
		return unauthorized(c)
	}

	year, month, err := parseYearMonth(c)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("budget list invalid year month")
		return response.Error(c, err)
	}

	items, err := h.budgets.List(c.Request().Context(), userID, year, month)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Int("year", year).Int("month", month).Msg("budget list failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", userID).Int("count", len(items)).Int("year", year).Int("month", month).Msg("budget list completed")
	return response.OK(c, newBudgetResponses(items))
}

func (h *BudgetHandler) Create(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("budget create started")
	userID, ok := authmiddleware.UserIDFromContext(c)
	if !ok {
		logger.Warn().Msg("budget create unauthorized")
		return unauthorized(c)
	}

	input, err := h.parseRequest(c)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("budget create invalid request")
		return response.Error(c, err)
	}

	item, err := h.budgets.Create(c.Request().Context(), userID, input)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("budget create failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", userID).Int64("budget_id", item.ID).Msg("budget create completed")
	return response.Created(c, newBudgetResponse(*item))
}

func (h *BudgetHandler) Update(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("budget update started")
	userID, ok := authmiddleware.UserIDFromContext(c)
	if !ok {
		logger.Warn().Msg("budget update unauthorized")
		return unauthorized(c)
	}

	id, err := parseIDParam(c, "id")
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("budget update invalid id")
		return response.Error(c, err)
	}

	input, err := h.parseRequest(c)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Int64("budget_id", id).Msg("budget update invalid request")
		return response.Error(c, err)
	}

	item, err := h.budgets.Update(c.Request().Context(), userID, id, input)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Int64("budget_id", id).Msg("budget update failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", userID).Int64("budget_id", item.ID).Msg("budget update completed")
	return response.OK(c, newBudgetResponse(*item))
}

func (h *BudgetHandler) Delete(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("budget delete started")
	userID, ok := authmiddleware.UserIDFromContext(c)
	if !ok {
		logger.Warn().Msg("budget delete unauthorized")
		return unauthorized(c)
	}

	id, err := parseIDParam(c, "id")
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("budget delete invalid id")
		return response.Error(c, err)
	}

	if err := h.budgets.Delete(c.Request().Context(), userID, id); err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Int64("budget_id", id).Msg("budget delete failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", userID).Int64("budget_id", id).Msg("budget delete completed")
	return c.NoContent(http.StatusNoContent)
}

func (h *BudgetHandler) parseRequest(c echo.Context) (service.BudgetInput, error) {
	var request budgetRequest
	if err := c.Bind(&request); err != nil {
		return service.BudgetInput{}, badRequestBody()
	}

	amount, err := parseAmount(c.Request().Context(), request.Amount, "amount")
	if err != nil {
		return service.BudgetInput{}, err
	}

	return service.BudgetInput{
		CategoryID:  request.CategoryID,
		Year:        request.Year,
		Month:       request.Month,
		AmountCents: amount,
	}, nil
}
