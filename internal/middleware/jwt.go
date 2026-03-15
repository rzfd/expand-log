package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/rzfd/expand/internal/pkg/apperror"
	"github.com/rzfd/expand/internal/pkg/auth"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/response"
)

const userIDContextKey = "user_id"

var middlewareTracer = otel.Tracer("middleware.jwt")

func JWT(tokenManager *auth.TokenManager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx, span := middlewareTracer.Start(c.Request().Context(), "middleware.jwt.validate")
			defer span.End()
			c.SetRequest(c.Request().WithContext(ctx))

			logger := logging.FromContext(c.Request().Context())
			logger.Info().Msg("jwt middleware started")
			header := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				span.SetStatus(codes.Error, "missing bearer token")
				logger.Warn().Msg("missing bearer token")
				return response.Error(c, apperror.New(http.StatusUnauthorized, "unauthorized", "missing bearer token"))
			}

			token := strings.TrimPrefix(header, "Bearer ")
			claims, err := tokenManager.Parse(token)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "invalid access token")
				logger.Warn().Err(err).Msg("invalid access token")
				return response.Error(c, apperror.New(http.StatusUnauthorized, "unauthorized", "invalid access token"))
			}
			span.SetAttributes(attribute.Int64("app.user.id", claims.UserID))

			ctxWithUser := logging.WithField(c.Request().Context(), userIDContextKey, claims.UserID)
			c.SetRequest(c.Request().WithContext(ctxWithUser))
			c.Set(userIDContextKey, claims.UserID)
			logging.FromContext(ctxWithUser).Info().Int64("user_id", claims.UserID).Msg("jwt middleware completed")
			return next(c)
		}
	}
}

func UserIDFromContext(c echo.Context) (int64, bool) {
	value := c.Get(userIDContextKey)
	userID, ok := value.(int64)
	return userID, ok
}
