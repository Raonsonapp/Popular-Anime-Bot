package domain

import "time"

type Episode struct {
	ID            int64   `db:"id" json:"id"`
	AnimeID       int64   `db:"anime_id" json:"anime_id"`
	SeasonID      *int64  `db:"season_id" json:"season_id,omitempty"`
	EpisodeNumber int     `db:"episode_number" json:"episode_number"`
	Title         *string `db:"title" json:"title,omitempty"`
	Quality       string  `db:"quality" json:"quality"`
	LanguageID    *int64  `db:"language_id" json:"language_id,omitempty"`

	// Where the bot can pull the actual video from via copyMessage.
	StorageChatID    int64 `db:"storage_chat_id" json:"storage_chat_id"`
	StorageMessageID int64 `db:"storage_message_id" json:"storage_message_id"`

	SourceChannelID *int64 `db:"source_channel_id" json:"source_channel_id,omitempty"`
	SourceMessageID *int64 `db:"source_message_id" json:"source_message_id,omitempty"`

	DurationSeconds *int   `db:"duration_seconds" json:"duration_seconds,omitempty"`
	SizeBytes       *int64 `db:"size_bytes" json:"size_bytes,omitempty"`

	IsDeleted bool      `db:"is_deleted" json:"is_deleted"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type SourceChannel struct {
	ID                  int64     `db:"id" json:"id"`
	TelegramChannelID   int64     `db:"telegram_channel_id" json:"telegram_channel_id"`
	Username            *string   `db:"username" json:"username,omitempty"`
	Title               string    `db:"title" json:"title"`
	SourceLanguage      string    `db:"source_language" json:"source_language"`
	IsActive            bool      `db:"is_active" json:"is_active"`
	LastSyncedMessageID int64     `db:"last_synced_message_id" json:"last_synced_message_id"`
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
}
