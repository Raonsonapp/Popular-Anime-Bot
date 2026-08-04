package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	BotToken         string
	APIBaseURL       string
	InternalAPIKey   string
	TargetChannelIDs []int64
	BotUsername      string
	PostCronSchedule string // e.g. "0 9 * * *" -> daily at 09:00
	PopularityCron   string // e.g. "0 * * * *" -> hourly
	PostsPerRun      int
}

func Load() Config {
	postsPerRun, err := strconv.Atoi(os.Getenv("POSTS_PER_RUN"))
	if err != nil || postsPerRun <= 0 {
		postsPerRun = 3
	}

	return Config{
		BotToken:         os.Getenv("BOT_TOKEN"),
		APIBaseURL:       getEnv("API_BASE_URL", "http://api:8080"),
		InternalAPIKey:   os.Getenv("INTERNAL_API_KEY"),
		TargetChannelIDs: parseChannelIDs(os.Getenv("TARGET_CHANNEL_ID")),
		BotUsername:      os.Getenv("BOT_USERNAME"),
		// Twice a day (8am and 8pm), in the Asia/Dushanbe timezone the
		// cron scheduler is configured with (see cmd/scheduler/main.go).
		PostCronSchedule: getEnv("POST_CRON_SCHEDULE", "0 8,20 * * *"),
		PopularityCron:   getEnv("POPULARITY_CRON_SCHEDULE", "0 * * * *"),
		PostsPerRun:      postsPerRun,
	}
}

// parseChannelIDs accepts one id or several comma-separated ids (posting
// the same anime to more than one showcase channel), skipping anything
// that doesn't parse instead of silently defaulting the whole config to 0.
func parseChannelIDs(raw string) []int64 {
	var ids []int64
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
