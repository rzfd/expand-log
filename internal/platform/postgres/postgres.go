package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rzfd/expand/internal/config"
	"github.com/rzfd/expand/internal/pkg/logging"
)

func OpenWithRetry(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	var lastErr error
	logger := logging.FromContext(ctx)
	logger.Info().Int("max_attempts", cfg.ConnectRetries).Msg("postgres open with retry started")

	for attempt := 1; attempt <= cfg.ConnectRetries; attempt++ {
		pool, err := open(ctx, cfg)
		if err == nil {
			if attempt > 1 {
				logger.Info().Int("attempt", attempt).Msg("postgres connection established after retry")
			}
			return pool, nil
		}

		lastErr = err
		logger.Warn().
			Int("attempt", attempt).
			Int("max_attempts", cfg.ConnectRetries).
			Str("retry_delay", cfg.ConnectRetryDelay.String()).
			Err(err).
			Msg("postgres connection attempt failed")

		select {
		case <-ctx.Done():
			logger.Warn().Err(ctx.Err()).Msg("postgres open with retry canceled")
			return nil, ctx.Err()
		case <-time.After(cfg.ConnectRetryDelay):
		}
	}

	logger.Error().Err(lastErr).Int("max_attempts", cfg.ConnectRetries).Msg("postgres open with retry exhausted")
	return nil, fmt.Errorf("connect postgres after %d attempts: %w", cfg.ConnectRetries, lastErr)
}

func open(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Msg("postgres open started")
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		logger.Error().Err(err).Msg("postgres parse config failed")
		return nil, err
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		logger.Error().Err(err).Msg("postgres new pool failed")
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		logger.Error().Err(err).Msg("postgres ping failed")
		return nil, err
	}

	logger.Info().Msg("postgres open completed")
	return pool, nil
}
