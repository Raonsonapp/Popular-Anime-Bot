package config

import (
	"os"
)

type Config struct {
	Port           string
	DatabaseURL    string
	InternalAPIKey string // shared secret required from bot/listener/scheduler services
}

func Load() Config {
	return Config{
		Port:           getEnv("API_PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/animebot?sslmode=disable"),
		InternalAPIKey: getEnv("INTERNAL_API_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
