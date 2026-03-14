package handler

import (
	"time"

	"github.com/labstack/echo/v4"

	"github.com/rzfd/expand/internal/pkg/logging"
)

func RequestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			request := c.Request()

			route := c.Path()
			if route == "" {
				route = request.URL.Path
			}
			requestID := request.Header.Get(echo.HeaderXRequestID)
			if requestID == "" {
				requestID = c.Response().Header().Get(echo.HeaderXRequestID)
			}

			requestLogger := logging.FromContext(request.Context()).With().
				Str("request_id", requestID).
				Str("method", request.Method).
				Str("route", route).
				Str("uri", request.RequestURI).
				Str("remote_ip", c.RealIP()).
				Logger()
			c.SetRequest(request.WithContext(logging.WithContext(request.Context(), requestLogger)))

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			request = c.Request()
			response := c.Response()
			route = c.Path()
			if route == "" {
				route = request.URL.Path
			}

			logger := logging.FromContext(request.Context())
			var event = logger.Info()
			switch {
			case response.Status >= 500:
				event = logger.Error()
			case response.Status >= 400:
				event = logger.Warn()
			}

			if err != nil {
				event = event.Err(err)
			}

			event.
				Str("route", route).
				Int("status", response.Status).
				Int64("latency_ms", time.Since(start).Milliseconds()).
				Int64("bytes_in", request.ContentLength).
				Int64("bytes_out", response.Size).
				Str("user_agent", request.UserAgent()).
				Msg("http request completed")

			return nil
		}
	}
}
