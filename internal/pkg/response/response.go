package response

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/rzfd/expand/internal/pkg/apperror"
	"github.com/rzfd/expand/internal/pkg/logging"
)

type Envelope struct {
	Success bool       `json:"success"`
	Data    any        `json:"data,omitempty"`
	Meta    any        `json:"meta,omitempty"`
	Error   *ErrorBody `json:"error,omitempty"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func JSON(c echo.Context, status int, data any, meta any) error {
	span := trace.SpanFromContext(c.Request().Context())
	span.SetAttributes(attribute.Int("app.response.status", status))
	span.AddEvent("response_sent")
	logging.FromContext(c.Request().Context()).Info().Int("status", status).Msg("response json")
	return c.JSON(status, Envelope{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

func OK(c echo.Context, data any) error {
	logging.FromContext(c.Request().Context()).Info().Msg("response ok")
	return JSON(c, http.StatusOK, data, nil)
}

func Created(c echo.Context, data any) error {
	logging.FromContext(c.Request().Context()).Info().Msg("response created")
	return JSON(c, http.StatusCreated, data, nil)
}

func Error(c echo.Context, err error) error {
	span := trace.SpanFromContext(c.Request().Context())
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		if appErr.Code == "validation_error" || appErr.Status == http.StatusTooManyRequests {
			IncValidationFailure(appErr.Code)
		}
		logError(c, appErr.Status, appErr.Code, appErr.Message, appErr.Details, err)
		span.SetAttributes(
			attribute.Int("app.response.status", appErr.Status),
			attribute.String("app.error.code", appErr.Code),
			attribute.String("app.error.message", appErr.Message),
		)
		span.AddEvent("response_error")
		return c.JSON(appErr.Status, Envelope{
			Success: false,
			Error: &ErrorBody{
				Code:    appErr.Code,
				Message: appErr.Message,
				Details: appErr.Details,
			},
		})
	}

	logError(c, http.StatusInternalServerError, "internal_error", "internal server error", nil, err)
	span.SetAttributes(
		attribute.Int("app.response.status", http.StatusInternalServerError),
		attribute.String("app.error.code", "internal_error"),
		attribute.String("app.error.message", "internal server error"),
	)
	span.AddEvent("response_error")
	return c.JSON(http.StatusInternalServerError, Envelope{
		Success: false,
		Error: &ErrorBody{
			Code:    "internal_error",
			Message: "internal server error",
		},
	})
}

func logError(c echo.Context, status int, code, message string, details any, err error) {
	if c == nil || c.Request() == nil {
		return
	}

	logger := logging.FromContext(c.Request().Context())
	event := logger.Warn().Err(err)
	if status >= 500 || code == "validation_error" {
		event = logger.Error().Err(err)
	}

	if details != nil {
		event = event.Interface("error_details", details)
	}

	event.
		Int("status", status).
		Str("error_code", code).
		Str("error_message", message).
		Msg("request failed")
}
