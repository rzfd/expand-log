package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rzfd/expand/internal/config"
	"github.com/rzfd/expand/internal/pkg/logging"
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
