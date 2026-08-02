package apiclient

type Genre struct {
	ID          int64  `json:"id"`
	NamePersian string `json:"name_persian"`
	NameEnglish string `json:"name_english"`
}

type Anime struct {
	ID                     int64   `json:"id"`
	Title                  string  `json:"title_persian"`
	TitleEnglish           *string `json:"title_english"`
	TitleJapanese          *string `json:"title_japanese"`
	SynopsisPersian        *string `json:"synopsis_persian"`
	Kind                   string  `json:"kind"`
	Status                 string  `json:"status"`
	Year                   *int    `json:"year"`
	StudioName             string  `json:"studio_name"`
	PosterStorageChatID    *int64  `json:"poster_storage_chat_id"`
	PosterStorageMessageID *int64  `json:"poster_storage_message_id"`
	PosterURL              *string `json:"poster_url"`
	TrailerURL             *string `json:"trailer_url"`
	EpisodesCount          int     `json:"episodes_count"`
	RatingScore            float64 `json:"rating_score"`
	ViewCount              int64   `json:"view_count"`
	Genres                 []Genre `json:"genres"`
}

type Episode struct {
	ID               int64   `json:"id"`
	AnimeID          int64   `json:"anime_id"`
	EpisodeNumber    int     `json:"episode_number"`
	Title            *string `json:"title"`
	Quality          string  `json:"quality"`
	StorageChatID    int64   `json:"storage_chat_id"`
	StorageMessageID int64   `json:"storage_message_id"`
}

type ListResponse struct {
	Items []Anime `json:"items"`
	Total int     `json:"total"`
	Page  int     `json:"page"`
}

type EpisodeListResponse struct {
	Items []Episode `json:"items"`
	Total int       `json:"total"`
	Page  int       `json:"page"`
}

type User struct {
	ID             int64  `json:"id"`
	TelegramUserID int64  `json:"telegram_user_id"`
	LanguagePref   string `json:"language_pref"`
}

type WatchHistoryEntry struct {
	ID              int64 `json:"id"`
	EpisodeID       int64 `json:"episode_id"`
	ProgressSeconds int   `json:"progress_seconds"`
	Completed       bool  `json:"completed"`
}
