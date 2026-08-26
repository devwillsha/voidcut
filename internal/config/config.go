// Package config loads environment-based configuration shared by all binaries.
package config

import (
	"fmt"
	"os"
)

// Config holds settings common to every service and the agent.
type Config struct {
	Env           string // "development", "staging", "production"
	GatewayURL    string
	MicSampleRate int
	MicChunkSize  int
	MicThreshold  float64
	NATSURL       string
	PostgresDSN   string
	RedisAddr     string
}

// Load reads configuration from environment variables, falling back to
// local-development defaults so `go run` works out of the box.
func Load() Config {
	return Config{
		Env:           getEnv("VOIDCUT_ENV", "development"),
		GatewayURL:    getEnv("VOIDCUT_GATEWAY_URL", "http://localhost:8080"),
		MicSampleRate: getEnvInt("VOIDCUT_MIC_SAMPLE_RATE", 48000),
		MicChunkSize:  getEnvInt("VOIDCUT_MIC_CHUNK_SIZE", 480),
		MicThreshold:  getEnvFloat("VOIDCUT_MIC_THRESHOLD", 0.05),
		NATSURL:       getEnv("VOIDCUT_NATS_URL", "nats://localhost:4222"),
		PostgresDSN:   getEnv("VOIDCUT_POSTGRES_DSN", "postgres://voidcut:voidcut@localhost:5432/voidcut?sslmode=disable"),
		RedisAddr:     getEnv("VOIDCUT_REDIS_ADDR", "localhost:6379"),
	}
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	var parsed int
	if _, err := fmt.Sscan(value, &parsed); err != nil {
		return fallback
	}
	return parsed
}

func getEnvFloat(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	var parsed float64
	if _, err := fmt.Sscan(value, &parsed); err != nil {
		return fallback
	}
	return parsed
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
