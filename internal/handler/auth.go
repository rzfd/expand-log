package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/rzfd/expand/internal/pkg/apperror"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/response"
	"github.com/rzfd/expand/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Register(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("auth register started")

	var request authRequest
	if err := c.Bind(&request); err != nil {
		logger.Warn().Err(err).Msg("auth register bind failed")
		return response.Error(c, apperror.New(http.StatusBadRequest, "validation_error", "invalid request body"))
	}

	user, err := h.auth.Register(c.Request().Context(), request.Email, request.Password)
	if err != nil {
		logger.Warn().Err(err).Msg("auth register failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", user.ID).Msg("auth register completed")
	return response.Created(c, newUserResponse(*user))
}

func (h *AuthHandler) Login(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("auth login started")

	var request authRequest
	if err := c.Bind(&request); err != nil {
		logger.Warn().Err(err).Msg("auth login bind failed")
		return response.Error(c, apperror.New(http.StatusBadRequest, "validation_error", "invalid request body"))
	}

	result, err := h.auth.Login(c.Request().Context(), request.Email, request.Password)
	if err != nil {
		logger.Warn().Err(err).Msg("auth login failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", result.User.ID).Msg("auth login completed")
	return response.OK(c, newAuthResponse(userAuthResult{
		User:      result.User,
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
	}))
}
