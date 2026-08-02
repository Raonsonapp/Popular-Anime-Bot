package config

import "os"

type Config struct {
	BotToken       string
	APIBaseURL     string
	InternalAPIKey string

	// Webhook mode (used on PaaS free tiers, e.g. Render, that only run
	// HTTP-serving "web services" for free). When WebhookURL is empty the
	// bot falls back to long-polling, which is what docker-compose/VPS
	// deployments use.
	Port          string
	WebhookURL    string
	WebhookSecret string
}

func Load() Config {
	return Config{
		BotToken:       os.Getenv("BOT_TOKEN"),
		APIBaseURL:     getEnv("API_BASE_URL", "http://api:8080"),
		InternalAPIKey: os.Getenv("INTERNAL_API_KEY"),
		Port:           getEnv("PORT", "8081"),
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
