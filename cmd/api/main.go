package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rzfd/expand/internal/config"
	"github.com/rzfd/expand/internal/handler"
	"github.com/rzfd/expand/internal/pkg/auth"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/tracing"
	"github.com/rzfd/expand/internal/platform/postgres"
	"github.com/rzfd/expand/internal/repository"
	"github.com/rzfd/expand/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		logger := logging.Configure(logging.Config{
			Service:   "expense-tracker",
			Component: "api",
			Level:     "info",
			Format:    "json",
		})
		logger.Error().Err(err).Msg("load config")
		os.Exit(1)
	}

	logger := logging.Configure(logging.Config{
		Service:   cfg.App.Name,
		Component: "api",
		Level:     cfg.Logging.Level,
		Format:    cfg.Logging.Format,
	})

	ctx, stop := signal.NotifyContext(logging.WithContext(context.Background(), logger), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	traceShutdown, err := tracing.Setup(ctx, tracing.Config{
		Enabled:        cfg.Tracing.Enabled,
		Endpoint:       cfg.Tracing.Endpoint,
		Insecure:       cfg.Tracing.Insecure,
		SampleRatio:    cfg.Tracing.SampleRatio,
		ServiceName:    cfg.App.Name,
		ServiceVersion: cfg.Tracing.ServiceVersion,
		Environment:    cfg.Tracing.Environment,
	})
	if err != nil {
		logger.Error().Err(err).Msg("setup tracing")
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := traceShutdown(shutdownCtx); err != nil {
			logger.Warn().Err(err).Msg("shutdown tracing")
		}
	}()
	if !cfg.Tracing.Enabled {
		logger.Info().Msg("tracing disabled")
	}

	db, err := postgres.OpenWithRetry(ctx, cfg.Database)
	if err != nil {
		logger.Error().Err(err).Msg("connect database")
		os.Exit(1)
	}
	defer db.Close()

	logger.Info().
		Str("host", cfg.Database.Host).
		Str("port", cfg.Database.Port).
		Str("name", cfg.Database.Name).
		Msg("database connected")

	tokenManager := auth.NewTokenManager(cfg.Auth.JWTSecret, cfg.Auth.JWTTTL)

	userRepo := repository.NewUserRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	reportRepo := repository.NewReportRepository(db)
	budgetRepo := repository.NewBudgetRepository(db)
	recurringRepo := repository.NewRecurringTransactionRepository(db)

	authService := service.NewAuthService(userRepo, tokenManager)
	categoryService := service.NewCategoryService(categoryRepo)
	transactionService := service.NewTransactionService(transactionRepo, categoryRepo)
	reportService := service.NewReportService(reportRepo)
	budgetService := service.NewBudgetService(budgetRepo, categoryRepo)
	recurringService := service.NewRecurringService(recurringRepo, categoryRepo)

	echo := handler.NewEcho(cfg.App.Name)
	handler.RegisterRoutes(echo, handler.RouterDependencies{
		TokenManager: tokenManager,
		Auth:         handler.NewAuthHandler(authService),
		Categories:   handler.NewCategoryHandler(categoryService),
		Transactions: handler.NewTransactionHandler(transactionService),
		Reports:      handler.NewReportHandler(reportService),
		Budgets:      handler.NewBudgetHandler(budgetService),
		Recurring:    handler.NewRecurringHandler(recurringService),
	})

	serverErr := make(chan error, 1)
	go func() {
		address := ":" + cfg.App.Port
		logger.Info().Str("address", address).Msg("starting api server")
		serverErr <- echo.Start(address)
	}()

	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("api server exited with error")
			os.Exit(1)
		}
	case <-ctx.Done():
		logger.Info().Msg("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := echo.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("shutdown api server")
		os.Exit(1)
	}

	logger.Info().Msg("api server stopped")
}
