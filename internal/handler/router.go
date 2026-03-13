package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	authmiddleware "github.com/rzfd/expand/internal/middleware"
	"github.com/rzfd/expand/internal/pkg/apperror"
	"github.com/rzfd/expand/internal/pkg/auth"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/response"
)

type RouterDependencies struct {
	TokenManager *auth.TokenManager
	Auth         *AuthHandler
	Categories   *CategoryHandler
	Transactions *TransactionHandler
	Reports      *ReportHandler
	Budgets      *BudgetHandler
	Recurring    *RecurringHandler
}

func NewEcho() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = HTTPErrorHandler
	e.Use(echomiddleware.RequestID())
	e.Use(RequestLogger())
	e.Use(echomiddleware.RecoverWithConfig(echomiddleware.RecoverConfig{
		DisablePrintStack: true,
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			logging.FromContext(c.Request().Context()).
				Error().
				Err(err).
				Bytes("stack", stack).
				Msg("panic recovered")
			return err
		},
	}))
	return e
}

func RegisterRoutes(e *echo.Echo, deps RouterDependencies) {
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	e.GET("/healthz", func(c echo.Context) error {
		return response.OK(c, map[string]string{"status": "ok"})
	})

	api := e.Group("/api/v1")
	api.POST("/auth/register", deps.Auth.Register)
	api.POST("/auth/login", deps.Auth.Login)

	protected := api.Group("")
	protected.Use(authmiddleware.JWT(deps.TokenManager))

	protected.GET("/categories", deps.Categories.List)
	protected.POST("/categories", deps.Categories.Create)
	protected.PUT("/categories/:id", deps.Categories.Update)
	protected.DELETE("/categories/:id", deps.Categories.Delete)

	protected.GET("/transactions", deps.Transactions.List)
	protected.GET("/transactions/:id", deps.Transactions.GetByID)
	protected.POST("/transactions", deps.Transactions.Create)
	protected.PUT("/transactions/:id", deps.Transactions.Update)
	protected.DELETE("/transactions/:id", deps.Transactions.Delete)

	protected.GET("/reports/monthly", deps.Reports.Monthly)
	protected.GET("/dashboard/summary", deps.Reports.Dashboard)

	protected.GET("/budgets", deps.Budgets.List)
	protected.POST("/budgets", deps.Budgets.Create)
	protected.PUT("/budgets/:id", deps.Budgets.Update)
	protected.DELETE("/budgets/:id", deps.Budgets.Delete)

	protected.GET("/recurring-transactions", deps.Recurring.List)
	protected.POST("/recurring-transactions", deps.Recurring.Create)
	protected.PUT("/recurring-transactions/:id", deps.Recurring.Update)
	protected.DELETE("/recurring-transactions/:id", deps.Recurring.Delete)

	e.GET("/", func(c echo.Context) error {
		return response.OK(c, map[string]string{
			"service": "expense-tracker-api",
			"status":  "ok",
		})
	})
}

func unauthorized(c echo.Context) error {
	return response.Error(c, badUnauthorized())
}

func badUnauthorized() error {
	return apperror.New(http.StatusUnauthorized, "unauthorized", "missing user context")
}

func badRequestBody() error {
	return apperror.New(http.StatusBadRequest, "validation_error", "invalid request body")
}

func badValidation(message string) error {
	return apperror.New(http.StatusBadRequest, "validation_error", message)
}
