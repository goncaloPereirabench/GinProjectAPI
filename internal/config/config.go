package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Environment string
	HTTP        HTTPConfig
	Database    DatabaseConfig
	Auth        AuthConfig
	RateLimit   RateLimitConfig
}

type HTTPConfig struct {
	Port            string
	ShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type AuthConfig struct {
	JWTSecret      string
	JWTIssuer      string
	AccessTokenTTL time.Duration
}

type RateLimitConfig struct {
	RequestsPerMinute int
	Burst             int
}

func Load() (Config, error) {
	cfg := Config{
		Environment: env("APP_ENV", "development"),
		HTTP: HTTPConfig{
			Port:            env("PORT", "8080"),
			ShutdownTimeout: durationEnv("SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		Database: DatabaseConfig{
			DSN:             env("DATABASE_DSN", ""),
			MaxOpenConns:    intEnv("DB_MAX_OPEN_CONNS", 10),
			MaxIdleConns:    intEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: durationEnv("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		},
		Auth: AuthConfig{
			JWTSecret:      env("JWT_SECRET", "dev-only-change-me"),
			JWTIssuer:      env("JWT_ISSUER", "gin-grocery-api"),
			AccessTokenTTL: durationEnv("JWT_ACCESS_TTL", 15*time.Minute),
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: intEnv("RATE_LIMIT_REQUESTS_PER_MINUTE", 60),
			Burst:             intEnv("RATE_LIMIT_BURST", 20),
		},
	}

	if cfg.Environment == "production" {
		if cfg.Auth.JWTSecret == "" || cfg.Auth.JWTSecret == "dev-only-change-me" {
			return Config{}, errors.New("JWT_SECRET must be set to a strong value in production")
		}
		if cfg.Database.DSN == "" {
			return Config{}, errors.New("DATABASE_DSN must be set in production")
		}
	}

	return cfg, nil
}

func env(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	return value
}

func intEnv(key string, fallback int) int {
	raw := env(key, "")
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	raw := env(key, "")
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}
