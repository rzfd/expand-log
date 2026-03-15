package handler

import (
	"fmt"
	"strings"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var handlerTracer = otel.Tracer("handler.http")

func HandlerTracing() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Path()
			if path == "" {
				path = c.Request().URL.Path
			}
			if shouldSkipObservability(path) {
				return next(c)
			}

			operation := formatHandlerOperation(c.Request().Method, path)
			ctx, span := handlerTracer.Start(c.Request().Context(), operation)
			span.SetAttributes(
				attribute.String("app.http.method", c.Request().Method),
				attribute.String("app.http.route", path),
			)
			defer span.End()

			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

func formatHandlerOperation(method, path string) string {
	route := strings.Trim(path, "/")
	if route == "" {
		route = "root"
	}
	route = strings.ReplaceAll(route, "/", ".")
	route = strings.ReplaceAll(route, ":", "")
	return fmt.Sprintf("handler.%s.%s", strings.ToLower(method), route)
}
