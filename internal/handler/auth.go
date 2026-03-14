package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/rzfd/expand/internal/pkg/apperror"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/response"
	"github.com/rzfd/expand/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

var authLimiter = newSlidingWindowLimiter(5*time.Minute, 5)

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

	if err := enforceAuthRateLimit(c, "register", request.Email); err != nil {
		logger.Warn().Err(err).Msg("auth register rate limited")
		return response.Error(c, err)
	}

	user, err := h.auth.Register(c.Request().Context(), request.Email, request.Password)
	if err != nil {
		logger.Warn().Err(err).Msg("auth register failed")
		return response.Error(c, err)
	}

	resetAuthRateLimit(c, "register", request.Email)

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

	if err := enforceAuthRateLimit(c, "login", request.Email); err != nil {
		logger.Warn().Err(err).Msg("auth login rate limited")
		return response.Error(c, err)
	}

	result, err := h.auth.Login(c.Request().Context(), request.Email, request.Password)
	if err != nil {
		logger.Warn().Err(err).Msg("auth login failed")
		return response.Error(c, err)
	}

	resetAuthRateLimit(c, "login", request.Email)

	logger.Info().Int64("user_id", result.User.ID).Msg("auth login completed")
	return response.OK(c, newAuthResponse(userAuthResult{
		User:      result.User,
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
	}))
}

func enforceAuthRateLimit(c echo.Context, action, email string) error {
	now := time.Now().UTC()
	emailKey := strings.ToLower(strings.TrimSpace(email))
	ipKey := strings.TrimSpace(c.RealIP())

	allowIP, retryIP := authLimiter.allow(action+":ip:"+ipKey, now)
	allowEmail, retryEmail := authLimiter.allow(action+":email:"+emailKey, now)
	if allowIP && allowEmail {
		return nil
	}

	retryAfter := retryIP
	if retryEmail > retryAfter {
		retryAfter = retryEmail
	}

	return apperror.WithDetails(
		apperror.New(http.StatusTooManyRequests, "rate_limited", "too many authentication attempts, please try again later"),
		map[string]any{"retry_after_seconds": int(retryAfter.Seconds())},
	)
}

func resetAuthRateLimit(c echo.Context, action, email string) {
	emailKey := strings.ToLower(strings.TrimSpace(email))
	ipKey := strings.TrimSpace(c.RealIP())
	authLimiter.reset(action + ":ip:" + ipKey)
	authLimiter.reset(action + ":email:" + emailKey)
}
