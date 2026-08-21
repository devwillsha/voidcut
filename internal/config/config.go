// Package config loads environment-based configuration shared by all binaries.
package config

import (
	"os"
)

// Config holds settings common to every service and the agent.
type Config struct {
	Env         string // "development", "staging", "production"
	NATSURL     string
	PostgresDSN string
	RedisAddr   string
}

// Load reads configuration from environment variables, falling back to
// local-development defaults so `go run` works out of the box.
func Load() Config {
	return Config{
		Env:         getEnv("VOIDCUT_ENV", "development"),
		NATSURL:     getEnv("VOIDCUT_NATS_URL", "nats://localhost:4222"),
		PostgresDSN: getEnv("VOIDCUT_POSTGRES_DSN", "postgres://voidcut:voidcut@localhost:5432/voidcut?sslmode=disable"),
		RedisAddr:   getEnv("VOIDCUT_REDIS_ADDR", "localhost:6379"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
