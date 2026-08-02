package config

import (
	"os"
	"strconv"
)

type Config struct {
	BotToken         string
	APIBaseURL       string
	InternalAPIKey   string
	TargetChannelID  int64
	BotUsername      string
	PostCronSchedule string // e.g. "0 9 * * *" -> daily at 09:00
	PopularityCron   string // e.g. "0 * * * *" -> hourly
	PostsPerRun      int
}

func Load() Config {
	targetChannelID, _ := strconv.ParseInt(os.Getenv("TARGET_CHANNEL_ID"), 10, 64)
	postsPerRun, err := strconv.Atoi(os.Getenv("POSTS_PER_RUN"))
	if err != nil || postsPerRun <= 0 {
		postsPerRun = 3
	}

	return Config{
		BotToken:         os.Getenv("BOT_TOKEN"),
		APIBaseURL:       getEnv("API_BASE_URL", "http://api:8080"),
		InternalAPIKey:   os.Getenv("INTERNAL_API_KEY"),
		TargetChannelID:  targetChannelID,
		BotUsername:      os.Getenv("BOT_USERNAME"),
		PostCronSchedule: getEnv("POST_CRON_SCHEDULE", "0 9 * * *"),
		PopularityCron:   getEnv("POPULARITY_CRON_SCHEDULE", "0 * * * *"),
		PostsPerRun:      postsPerRun,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
