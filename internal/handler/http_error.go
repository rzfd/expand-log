package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/rzfd/expand/internal/pkg/apperror"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/response"
)

func HTTPErrorHandler(err error, c echo.Context) {
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
		_ = response.Error(c, apperror.New(echoErr.Code, "http_error", message))
		return
	}

	logger.Error().Err(err).Msg("http error handled")
	_ = response.Error(c, err)
}
