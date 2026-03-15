package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/rzfd/expand/internal/pkg/apperror"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/response"
)

func HTTPErrorHandler(err error, c echo.Context) {
	span := trace.SpanFromContext(c.Request().Context())
	logger := logging.FromContext(c.Request().Context())
	if c.Response().Committed {
		logger.Warn().Err(err).Msg("http error response already committed")
		return
	}

	var echoErr *echo.HTTPError
	if errors.As(err, &echoErr) {
		message, _ := echoErr.Message.(string)
		if message == "" {
			message = http.StatusText(echoErr.Code)
		}
		logger.Warn().Err(err).Int("status", echoErr.Code).Msg("http error handled from echo")
		span.SetAttributes(
			attribute.Int("app.error.http_status", echoErr.Code),
			attribute.String("app.error.code", "http_error"),
			attribute.String("app.error.message", message),
		)
		span.RecordError(err)
		_ = response.Error(c, apperror.New(echoErr.Code, "http_error", message))
		return
	}

	span.RecordError(err)
	logger.Error().Err(err).Msg("http error handled")
	_ = response.Error(c, err)
}
