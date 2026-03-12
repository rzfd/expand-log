package logging

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Service   string
	Component string
	Level     string
	Format    string
}

func Configure(cfg Config) zerolog.Logger {
	zerolog.SetGlobalLevel(parseLevel(cfg.Level))
	zerolog.TimeFieldFormat = time.RFC3339

	logger := New(cfg)
	zerolog.DefaultContextLogger = &logger
	log.Logger = logger
	return logger
}

func New(cfg Config) zerolog.Logger {
	context := zerolog.New(outputWriter(cfg.Format)).With().Timestamp()
	if cfg.Service != "" {
		context = context.Str("service", cfg.Service)
	}
	if cfg.Component != "" {
		context = context.Str("component", cfg.Component)
	}

	return context.Logger()
}

func outputWriter(format string) io.Writer {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		return os.Stdout
	default:
		return zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}
}

func parseLevel(value string) zerolog.Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return zerolog.DebugLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}
