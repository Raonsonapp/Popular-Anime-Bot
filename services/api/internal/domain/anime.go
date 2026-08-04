package domain

import "time"

type AnimeKind string

const (
	KindTV      AnimeKind = "tv"
	KindMovie   AnimeKind = "movie"
	KindOVA     AnimeKind = "ova"
	KindSpecial AnimeKind = "special"
)

type AnimeStatus string

const (
	StatusOngoing   AnimeStatus = "ongoing"
	StatusCompleted AnimeStatus = "completed"
	StatusAnnounced AnimeStatus = "announced"
)

type Anime struct {
	ID            int64   `db:"id" json:"id"`
	Title         string  `db:"title_persian" json:"title_persian"`
	TitleEnglish  *string `db:"title_english" json:"title_english,omitempty"`
	TitleJapanese *string `db:"title_japanese" json:"title_japanese,omitempty"`
	TitleOriginal *string `db:"title_original" json:"title_original,omitempty"`

	SynopsisPersian  *string `db:"synopsis_persian" json:"synopsis_persian,omitempty"`
	SynopsisOriginal *string `db:"synopsis_original" json:"synopsis_original,omitempty"`

	Kind   AnimeKind   `db:"kind" json:"kind"`
	Status AnimeStatus `db:"status" json:"status"`

	Year     *int   `db:"year" json:"year,omitempty"`
	StudioID *int64 `db:"studio_id" json:"studio_id,omitempty"`

	PosterStorageChatID    *int64  `db:"poster_storage_chat_id" json:"poster_storage_chat_id,omitempty"`
	PosterStorageMessageID *int64  `db:"poster_storage_message_id" json:"poster_storage_message_id,omitempty"`
	PosterURL              *string `db:"poster_url" json:"poster_url,omitempty"`
	BannerURL              *string `db:"banner_url" json:"banner_url,omitempty"`
	TrailerURL             *string `db:"trailer_url" json:"trailer_url,omitempty"`

	EpisodesCount   int  `db:"episodes_count" json:"episodes_count"`
	DurationMinutes *int `db:"duration_minutes" json:"duration_minutes,omitempty"`

	RatingScore     float64 `db:"rating_score" json:"rating_score"`
	RatingCount     int     `db:"rating_count" json:"rating_count"`
	PopularityScore float64 `db:"popularity_score" json:"popularity_score"`
	ViewCount       int64   `db:"view_count" json:"view_count"`

	SourceChannelID *int64 `db:"source_channel_id" json:"source_channel_id,omitempty"`
	IsPublished     bool   `db:"is_published" json:"is_published"`
	IsDeleted       bool   `db:"is_deleted" json:"is_deleted"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`

	// populated by joins, not stored on this table
	StudioName string  `db:"studio_name" json:"studio_name,omitempty"`
	Genres     []Genre `db:"-" json:"genres,omitempty"`
}

type Genre struct {
	ID          int64  `db:"id" json:"id"`
	NamePersian string `db:"name_persian" json:"name_persian"`
	NameEnglish string `db:"name_english" json:"name_english"`
}

type Studio struct {
	ID   int64  `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
}

type Season struct {
	ID            int64   `db:"id" json:"id"`
	AnimeID       int64   `db:"anime_id" json:"anime_id"`
	SeasonNumber  int     `db:"season_number" json:"season_number"`
	Title         *string `db:"title" json:"title,omitempty"`
	Year          *int    `db:"year" json:"year,omitempty"`
	EpisodesCount int     `db:"episodes_count" json:"episodes_count"`
}

// AnimeFilter narrows the anime listing / search query.
type AnimeFilter struct {
	Search   string
	GenreID  int64
	StudioID int64
	Status   AnimeStatus
	Year     int
	SortBy   string // "newest", "trending", "top_rated", "random"
	Page     int
	PageSize int
}
