package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rzfd/expand/internal/config"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/tracing"
	"github.com/rzfd/expand/internal/platform/postgres"
	"github.com/rzfd/expand/internal/repository"
	"github.com/rzfd/expand/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		logger := logging.Configure(logging.Config{
			Service:   "expense-tracker",
			Component: "worker",
			Level:     "info",
			Format:    "json",
		})
		logger.Error().Err(err).Msg("load config")
		os.Exit(1)
	}

	logger := logging.Configure(logging.Config{
		Service:   cfg.App.Name,
		Component: "worker",
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
		ServiceName:    cfg.App.Name + "-worker",
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

	recurringRepo := repository.NewRecurringTransactionRepository(db)
	processor := worker.NewProcessor(recurringRepo, cfg.Worker.BatchSize)
	runner := worker.NewRunner(processor, cfg.Worker.PollInterval)

	logger.Info().
		Str("interval", cfg.Worker.PollInterval.String()).
		Int("batch_size", cfg.Worker.BatchSize).
		Msg("starting worker")
	runner.Start(ctx)

	logger.Info().Msg("worker stopped")
	os.Exit(0)
}
