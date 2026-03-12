package logging

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func WithContext(ctx context.Context, logger zerolog.Logger) context.Context {
	return logger.WithContext(ctx)
}

func FromContext(ctx context.Context) *zerolog.Logger {
	if ctx == nil {
		return &log.Logger
	}

	return zerolog.Ctx(ctx)
}

func WithField(ctx context.Context, key string, value any) context.Context {
	logger := FromContext(ctx).With().Interface(key, value).Logger()
	return logger.WithContext(ctx)
}
