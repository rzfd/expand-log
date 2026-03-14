package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	authmiddleware "github.com/rzfd/expand/internal/middleware"
	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/pagination"
	"github.com/rzfd/expand/internal/pkg/response"
	"github.com/rzfd/expand/internal/service"
)

type TransactionHandler struct {
	transactions *service.TransactionService
}

type transactionRequest struct {
	CategoryID      int64  `json:"category_id"`
	Type            string `json:"type"`
	Amount          string `json:"amount"`
	Note            string `json:"note"`
	TransactionDate string `json:"transaction_date"`
}

func NewTransactionHandler(transactions *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{transactions: transactions}
}

func (h *TransactionHandler) List(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("transaction list started")
	userID, ok := authmiddleware.UserIDFromContext(c)
	if !ok {
		logger.Warn().Msg("transaction list unauthorized")
		return unauthorized(c)
	}

	params, err := parsePagination(c)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("transaction list invalid pagination")
		return response.Error(c, err)
	}
	startDate, err := parseOptionalDate(c.Request().Context(), c.QueryParam("start_date"), "start_date")
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("transaction list invalid start_date")
		return response.Error(c, err)
	}
	endDate, err := parseOptionalDate(c.Request().Context(), c.QueryParam("end_date"), "end_date")
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("transaction list invalid end_date")
		return response.Error(c, err)
	}
	transactionType, err := parseTransactionType(c.Request().Context(), c.QueryParam("type"))
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("transaction list invalid type")
		return response.Error(c, err)
	}
	categoryID, err := parseOptionalInt64(c.Request().Context(), c.QueryParam("category_id"), "category_id")
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("transaction list invalid category_id")
		return response.Error(c, err)
	}
	minAmount, err := parseOptionalAmount(c.Request().Context(), c.QueryParam("min_amount"), "min_amount")
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("transaction list invalid min_amount")
		return response.Error(c, err)
	}
	maxAmount, err := parseOptionalAmount(c.Request().Context(), c.QueryParam("max_amount"), "max_amount")
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("transaction list invalid max_amount")
		return response.Error(c, err)
	}

	filter := model.TransactionFilter{
		StartDate:      startDate,
		EndDate:        endDate,
		Type:           transactionType,
		CategoryID:     categoryID,
		MinAmountCents: minAmount,
		MaxAmountCents: maxAmount,
		Page:           params.Page,
		PageSize:       params.PageSize,
		Offset:         params.Offset,
	}

	if startDate != nil && endDate != nil && endDate.Before(*startDate) {
		logger.Warn().Int64("user_id", userID).Msg("transaction list invalid range")
		return response.Error(c, badValidation("end_date must be on or after start_date"))
	}
	if startDate != nil && endDate != nil {
		maxRange := startDate.AddDate(1, 0, 0)
		if endDate.After(maxRange) {
			logger.Warn().Int64("user_id", userID).Msg("transaction list date range too large")
			return response.Error(c, badValidation("date range cannot exceed 1 year"))
		}
	}
	if minAmount != nil && maxAmount != nil && *maxAmount < *minAmount {
		logger.Warn().Int64("user_id", userID).Msg("transaction list invalid amount range")
		return response.Error(c, badValidation("max_amount must be greater than or equal to min_amount"))
	}

	items, total, err := h.transactions.List(c.Request().Context(), userID, filter)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("transaction list failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", userID).Int("count", len(items)).Int("total", total).Msg("transaction list completed")
	return response.JSON(c, http.StatusOK, newTransactionResponses(items), pagination.BuildMeta(params, total))
}

func (h *TransactionHandler) GetByID(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("transaction get by id started")
	userID, ok := authmiddleware.UserIDFromContext(c)
	if !ok {
		logger.Warn().Msg("transaction get by id unauthorized")
		return unauthorized(c)
	}

	id, err := parseIDParam(c, "id")
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("transaction get by id invalid id")
		return response.Error(c, err)
	}

	item, err := h.transactions.GetByID(c.Request().Context(), userID, id)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Int64("transaction_id", id).Msg("transaction get by id failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", userID).Int64("transaction_id", item.ID).Msg("transaction get by id completed")
	return response.OK(c, newTransactionResponse(*item))
}

func (h *TransactionHandler) Create(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("transaction create started")
	userID, ok := authmiddleware.UserIDFromContext(c)
	if !ok {
		logger.Warn().Msg("transaction create unauthorized")
		return unauthorized(c)
	}

	input, err := h.parseRequest(c)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("transaction create invalid request")
		return response.Error(c, err)
	}

	item, err := h.transactions.Create(c.Request().Context(), userID, input)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("transaction create failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", userID).Int64("transaction_id", item.ID).Msg("transaction create completed")
	return response.Created(c, newTransactionResponse(*item))
}

func (h *TransactionHandler) Update(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("transaction update started")
	userID, ok := authmiddleware.UserIDFromContext(c)
	if !ok {
		logger.Warn().Msg("transaction update unauthorized")
		return unauthorized(c)
	}

	id, err := parseIDParam(c, "id")
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("transaction update invalid id")
		return response.Error(c, err)
	}

	input, err := h.parseRequest(c)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Int64("transaction_id", id).Msg("transaction update invalid request")
		return response.Error(c, err)
	}

	item, err := h.transactions.Update(c.Request().Context(), userID, id, input)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Int64("transaction_id", id).Msg("transaction update failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", userID).Int64("transaction_id", item.ID).Msg("transaction update completed")
	return response.OK(c, newTransactionResponse(*item))
}

func (h *TransactionHandler) Delete(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("transaction delete started")
	userID, ok := authmiddleware.UserIDFromContext(c)
	if !ok {
		logger.Warn().Msg("transaction delete unauthorized")
		return unauthorized(c)
	}

	id, err := parseIDParam(c, "id")
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("transaction delete invalid id")
		return response.Error(c, err)
	}

	if err := h.transactions.Delete(c.Request().Context(), userID, id); err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Int64("transaction_id", id).Msg("transaction delete failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", userID).Int64("transaction_id", id).Msg("transaction delete completed")
	return c.NoContent(http.StatusNoContent)
}

func (h *TransactionHandler) parseRequest(c echo.Context) (service.TransactionInput, error) {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("transaction parse request started")
	var request transactionRequest
	if err := c.Bind(&request); err != nil {
		logger.Warn().Err(err).Msg("transaction parse request bind failed")
		return service.TransactionInput{}, badRequestBody()
	}

	amount, err := parseAmount(c.Request().Context(), request.Amount, "amount")
	if err != nil {
		logger.Warn().Err(err).Msg("transaction parse request invalid amount")
		return service.TransactionInput{}, err
	}
	transactionDate, err := parseRequiredDate(c.Request().Context(), request.TransactionDate, "transaction_date")
	if err != nil {
		logger.Warn().Err(err).Msg("transaction parse request invalid transaction_date")
		return service.TransactionInput{}, err
	}

	result := service.TransactionInput{
		CategoryID:      request.CategoryID,
		Type:            model.TransactionType(request.Type),
		AmountCents:     amount,
		Note:            request.Note,
		TransactionDate: transactionDate,
	}
	logger.Info().Msg("transaction parse request completed")
	return result, nil
}
