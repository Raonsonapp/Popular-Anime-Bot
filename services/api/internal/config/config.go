package config

import (
	"os"
)

type Config struct {
	Port           string
	DatabaseURL    string
	InternalAPIKey string // shared secret required from bot/listener/scheduler services

	// Optional: embeds the Telegram bot in this same process/port when set,
	// so a single free Render Web Service can run both. Leave BotToken
	// empty to run the API alone (standalone services/bot handles the bot
	// instead, e.g. for docker-compose/VPS deployments).
	BotToken      string
	WebhookURL    string
	WebhookSecret string
}

func Load() Config {
	return Config{
		// Render (and most PaaS providers) inject PORT and expect the app to
		// bind to it; API_PORT stays as the docker-compose/local convention.
		Port:           getEnv("PORT", getEnv("API_PORT", "8080")),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/animebot?sslmode=disable"),
		InternalAPIKey: getEnv("INTERNAL_API_KEY", ""),
		BotToken:       os.Getenv("BOT_TOKEN"),
		WebhookURL:     os.Getenv("WEBHOOK_URL"),
		WebhookSecret:  os.Getenv("WEBHOOK_SECRET"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
