package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/rzfd/expand/internal/pkg/apperror"
	"github.com/rzfd/expand/internal/pkg/auth"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/response"
)

const userIDContextKey = "user_id"

func JWT(tokenManager *auth.TokenManager) echo.MiddlewareFunc {
	logging.FromContext(nil).Info().Msg("jwt middleware initialized")
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			logger := logging.FromContext(c.Request().Context())
			logger.Info().Msg("jwt middleware started")
			header := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				logger.Warn().Msg("missing bearer token")
				return response.Error(c, apperror.New(http.StatusUnauthorized, "unauthorized", "missing bearer token"))
			}

			token := strings.TrimPrefix(header, "Bearer ")
			claims, err := tokenManager.Parse(token)
			if err != nil {
				logger.Warn().Err(err).Msg("invalid access token")
				return response.Error(c, apperror.New(http.StatusUnauthorized, "unauthorized", "invalid access token"))
			}

			c.SetRequest(c.Request().WithContext(logging.WithField(c.Request().Context(), userIDContextKey, claims.UserID)))
			c.Set(userIDContextKey, claims.UserID)
			logger.Info().Int64("user_id", claims.UserID).Msg("jwt middleware completed")
			return next(c)
		}
	}
}

func UserIDFromContext(c echo.Context) (int64, bool) {
	logging.FromContext(c.Request().Context()).Info().Msg("jwt user id from context")
	value := c.Get(userIDContextKey)
	userID, ok := value.(int64)
	return userID, ok
}
