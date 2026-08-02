package config

import "os"

type Config struct {
	BotToken       string
	APIBaseURL     string
	InternalAPIKey string
}

func Load() Config {
	return Config{
		BotToken:       os.Getenv("BOT_TOKEN"),
		APIBaseURL:     getEnv("API_BASE_URL", "http://api:8080"),
		InternalAPIKey: os.Getenv("INTERNAL_API_KEY"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
