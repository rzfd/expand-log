package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rzfd/expand/internal/pkg/logging"
)

type Config struct {
	App      AppConfig
	Logging  LoggingConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Worker   WorkerConfig
}

type AppConfig struct {
	Name string
	Port string
}

type LoggingConfig struct {
	Level  string
	Format string
}

type DatabaseConfig struct {
	Host              string
	Port              string
	User              string
	Password          string
	Name              string
	SSLMode           string
	MaxConns          int32
	MinConns          int32
	ConnectRetries    int
	ConnectRetryDelay time.Duration
}

type AuthConfig struct {
	JWTSecret string
	JWTTTL    time.Duration
}

type WorkerConfig struct {
	PollInterval time.Duration
	BatchSize    int
}

func Load() (Config, error) {
	logging.FromContext(nil).Info().Msg("config load started")
	cfg := Config{
		App: AppConfig{
			Name: getEnv("APP_NAME", "expense-tracker"),
			Port: getEnv("APP_PORT", "8080"),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "text"),
		},
		Database: DatabaseConfig{
			Host:              getEnv("DB_HOST", "postgres"),
			Port:              getEnv("DB_PORT", "5432"),
			User:              getEnv("DB_USER", "expense_user"),
			Password:          getEnv("DB_PASSWORD", "expense_password"),
			Name:              getEnv("DB_NAME", "expense_tracker"),
			SSLMode:           getEnv("DB_SSLMODE", "disable"),
			MaxConns:          int32(getEnvInt("DB_MAX_CONNS", 10)),
			MinConns:          int32(getEnvInt("DB_MIN_CONNS", 2)),
			ConnectRetries:    getEnvInt("DB_CONNECT_RETRIES", 20),
			ConnectRetryDelay: getEnvDuration("DB_CONNECT_RETRY_DELAY", 3*time.Second),
		},
		Auth: AuthConfig{
			JWTSecret: getEnv("JWT_SECRET", "change-me-in-production"),
			JWTTTL:    getEnvDuration("JWT_TTL", 24*time.Hour),
		},
		Worker: WorkerConfig{
			PollInterval: getEnvDuration("WORKER_POLL_INTERVAL", time.Minute),
			BatchSize:    getEnvInt("WORKER_BATCH_SIZE", 50),
		},
	}

	if strings.TrimSpace(cfg.Auth.JWTSecret) == "" {
		logging.FromContext(nil).Error().Msg("config load failed empty jwt secret")
		return Config{}, fmt.Errorf("JWT_SECRET must not be empty")
	}

	logging.FromContext(nil).Info().Msg("config load completed")
	return cfg, nil
}

func (c DatabaseConfig) DSN() string {
	logging.FromContext(nil).Info().Msg("config build database dsn")
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Name,
		c.SSLMode,
	)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		logging.FromContext(nil).Info().Str("key", key).Msg("config env value loaded")
		return value
	}
	logging.FromContext(nil).Info().Str("key", key).Msg("config env fallback used")
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok {
		logging.FromContext(nil).Info().Str("key", key).Msg("config env int fallback used")
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		logging.FromContext(nil).Warn().Err(err).Str("key", key).Str("value", value).Msg("config env int parse failed")
		return fallback
	}

	logging.FromContext(nil).Info().Str("key", key).Int("value", parsed).Msg("config env int loaded")
	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok {
		logging.FromContext(nil).Info().Str("key", key).Msg("config env duration fallback used")
		return fallback
	}

	if seconds, err := strconv.Atoi(value); err == nil {
		logging.FromContext(nil).Info().Str("key", key).Int("seconds", seconds).Msg("config env duration parsed seconds")
		return time.Duration(seconds) * time.Second
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		logging.FromContext(nil).Warn().Err(err).Str("key", key).Str("value", value).Msg("config env duration parse failed")
		return fallback
	}

	logging.FromContext(nil).Info().Str("key", key).Str("duration", parsed.String()).Msg("config env duration loaded")
	return parsed
}
