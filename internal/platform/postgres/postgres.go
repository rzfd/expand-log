package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/rzfd/expand/internal/config"
	"github.com/rzfd/expand/internal/pkg/logging"
)

var postgresTracer = otel.Tracer("platform.postgres")

func OpenWithRetry(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	ctx, span := postgresTracer.Start(ctx, "platform.postgres.open_with_retry")
	defer span.End()
	span.SetAttributes(
		attribute.Int("db.connect.max_attempts", cfg.ConnectRetries),
		attribute.String("db.system", "postgresql"),
	)

	var lastErr error
	logger := logging.FromContext(ctx)
	logger.Info().Int("max_attempts", cfg.ConnectRetries).Msg("postgres open with retry started")

	for attempt := 1; attempt <= cfg.ConnectRetries; attempt++ {
		span.AddEvent("connect_attempt", trace.WithAttributes(attribute.Int("attempt", attempt)))
		pool, err := open(ctx, cfg)
		if err == nil {
			if attempt > 1 {
				logger.Info().Int("attempt", attempt).Msg("postgres connection established after retry")
			}
			return pool, nil
		}

		lastErr = err
		span.RecordError(err)
		logger.Warn().
			Int("attempt", attempt).
			Int("max_attempts", cfg.ConnectRetries).
			Str("retry_delay", cfg.ConnectRetryDelay.String()).
			Err(err).
			Msg("postgres connection attempt failed")

		select {
		case <-ctx.Done():
			span.RecordError(ctx.Err())
			span.SetStatus(codes.Error, "postgres connection canceled")
			logger.Warn().Err(ctx.Err()).Msg("postgres open with retry canceled")
			return nil, ctx.Err()
		case <-time.After(cfg.ConnectRetryDelay):
		}
	}

	span.RecordError(lastErr)
	span.SetStatus(codes.Error, "postgres connection retries exhausted")
	logger.Error().Err(lastErr).Int("max_attempts", cfg.ConnectRetries).Msg("postgres open with retry exhausted")
	return nil, fmt.Errorf("connect postgres after %d attempts: %w", cfg.ConnectRetries, lastErr)
}

func open(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	ctx, span := postgresTracer.Start(ctx, "platform.postgres.open")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.host", cfg.Host),
		attribute.String("db.port", cfg.Port),
		attribute.String("db.name", cfg.Name),
	)

	logger := logging.FromContext(ctx)
	logger.Info().Msg("postgres open started")
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "parse postgres config failed")
		logger.Error().Err(err).Msg("postgres parse config failed")
		return nil, err
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create pool failed")
		logger.Error().Err(err).Msg("postgres new pool failed")
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		span.RecordError(err)
		span.SetStatus(codes.Error, "postgres ping failed")
		logger.Error().Err(err).Msg("postgres ping failed")
		return nil, err
	}

	logger.Info().Msg("postgres open completed")
	return pool, nil
}
