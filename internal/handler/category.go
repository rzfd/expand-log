package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	authmiddleware "github.com/rzfd/expand/internal/middleware"
	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/apperror"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/response"
	"github.com/rzfd/expand/internal/service"
)

type CategoryHandler struct {
	categories *service.CategoryService
}

type categoryRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func NewCategoryHandler(categories *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{categories: categories}
}

func (h *CategoryHandler) List(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("category list started")
	userID, ok := authmiddleware.UserIDFromContext(c)
	if !ok {
		logger.Warn().Msg("category list unauthorized")
		return response.Error(c, apperror.New(http.StatusUnauthorized, "unauthorized", "missing user context"))
	}

	items, err := h.categories.List(c.Request().Context(), userID)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("category list failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", userID).Int("count", len(items)).Msg("category list completed")
	return response.OK(c, newCategoryResponses(items))
}

func (h *CategoryHandler) Create(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("category create started")
	userID, ok := authmiddleware.UserIDFromContext(c)
	if !ok {
		logger.Warn().Msg("category create unauthorized")
		return response.Error(c, apperror.New(http.StatusUnauthorized, "unauthorized", "missing user context"))
	}

	var request categoryRequest
	if err := c.Bind(&request); err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("category create bind failed")
		return response.Error(c, apperror.New(http.StatusBadRequest, "validation_error", "invalid request body"))
	}

	item, err := h.categories.Create(c.Request().Context(), userID, request.Name, model.TransactionType(request.Type))
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("category create failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", userID).Int64("category_id", item.ID).Msg("category create completed")
	return response.Created(c, newCategoryResponse(*item))
}

func (h *CategoryHandler) Update(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("category update started")
	userID, ok := authmiddleware.UserIDFromContext(c)
	if !ok {
		logger.Warn().Msg("category update unauthorized")
		return response.Error(c, apperror.New(http.StatusUnauthorized, "unauthorized", "missing user context"))
	}

	id, err := parseIDParam(c, "id")
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("category update invalid id")
		return response.Error(c, err)
	}

	var request categoryRequest
	if err := c.Bind(&request); err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Int64("category_id", id).Msg("category update bind failed")
		return response.Error(c, apperror.New(http.StatusBadRequest, "validation_error", "invalid request body"))
	}

	item, err := h.categories.Update(c.Request().Context(), userID, id, request.Name, model.TransactionType(request.Type))
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Int64("category_id", id).Msg("category update failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", userID).Int64("category_id", item.ID).Msg("category update completed")
	return response.OK(c, newCategoryResponse(*item))
}

func (h *CategoryHandler) Delete(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("category delete started")
	userID, ok := authmiddleware.UserIDFromContext(c)
	if !ok {
		logger.Warn().Msg("category delete unauthorized")
		return response.Error(c, apperror.New(http.StatusUnauthorized, "unauthorized", "missing user context"))
	}

	id, err := parseIDParam(c, "id")
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("category delete invalid id")
		return response.Error(c, err)
	}

	if err := h.categories.Delete(c.Request().Context(), userID, id); err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Int64("category_id", id).Msg("category delete failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", userID).Int64("category_id", id).Msg("category delete completed")
	return c.NoContent(http.StatusNoContent)
}
