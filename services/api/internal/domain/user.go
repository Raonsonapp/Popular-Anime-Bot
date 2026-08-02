package domain

import "time"

type User struct {
	ID             int64     `db:"id" json:"id"`
	TelegramUserID int64     `db:"telegram_user_id" json:"telegram_user_id"`
	Username       *string   `db:"username" json:"username,omitempty"`
	FirstName      *string   `db:"first_name" json:"first_name,omitempty"`
	LanguagePref   string    `db:"language_pref" json:"language_pref"`
	IsBanned       bool      `db:"is_banned" json:"is_banned"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	LastSeenAt     time.Time `db:"last_seen_at" json:"last_seen_at"`
}

type Admin struct {
	ID             int64  `db:"id" json:"id"`
	TelegramUserID int64  `db:"telegram_user_id" json:"telegram_user_id"`
	Role           string `db:"role" json:"role"`
}

type Favorite struct {
	ID        int64     `db:"id" json:"id"`
	UserID    int64     `db:"user_id" json:"user_id"`
	AnimeID   int64     `db:"anime_id" json:"anime_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type WatchHistoryEntry struct {
	ID              int64     `db:"id" json:"id"`
	UserID          int64     `db:"user_id" json:"user_id"`
	EpisodeID       int64     `db:"episode_id" json:"episode_id"`
	ProgressSeconds int       `db:"progress_seconds" json:"progress_seconds"`
	Completed       bool      `db:"completed" json:"completed"`
	WatchedAt       time.Time `db:"watched_at" json:"watched_at"`
}

type ChannelPost struct {
	ID                int64     `db:"id" json:"id"`
	AnimeID           int64     `db:"anime_id" json:"anime_id"`
	TargetChannelID   int64     `db:"target_channel_id" json:"target_channel_id"`
	TelegramMessageID *int64    `db:"telegram_message_id" json:"telegram_message_id,omitempty"`
	PublishedAt       time.Time `db:"published_at" json:"published_at"`
}
