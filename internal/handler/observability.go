package handler

import "github.com/labstack/echo/v4"

func shouldSkipObservability(path string) bool {
	return path == "/" || path == "/metrics"
}

func shouldSkipObservabilityForContext(c echo.Context) bool {
	path := c.Path()
	if path == "" {
		path = c.Request().URL.Path
	}
	return shouldSkipObservability(path)
}
