package domain

import "context"

// AnimeRepository persists and queries the anime catalog.
type AnimeRepository interface {
	Create(ctx context.Context, a *Anime) (int64, error)
	Update(ctx context.Context, a *Anime) error
	GetByID(ctx context.Context, id int64) (*Anime, error)
	// FindByTitle matches by original title alone (not scoped to a source
	// channel) so the same anime scraped from multiple channels merges
	// into one catalog entry instead of one per channel.
	FindByTitle(ctx context.Context, originalTitle string) (*Anime, error)
	List(ctx context.Context, f AnimeFilter) ([]Anime, int, error)
	SetGenres(ctx context.Context, animeID int64, genreIDs []int64) error
	IncrementView(ctx context.Context, animeID int64) error
	RecalculatePopularity(ctx context.Context) error
	SoftDelete(ctx context.Context, id int64) error
}

type EpisodeRepository interface {
	Create(ctx context.Context, e *Episode) (int64, error)
	Update(ctx context.Context, e *Episode) error
	GetByID(ctx context.Context, id int64) (*Episode, error)
	FindBySource(ctx context.Context, sourceChannelID, sourceMessageID int64) (*Episode, error)
	// ListByAnime lists episodes for an anime, optionally narrowed to one
	// season (seasonID == 0 means "no filter" - every episode regardless
	// of season, the pre-seasons behavior).
	ListByAnime(ctx context.Context, animeID, seasonID int64, page, pageSize int) ([]Episode, int, error)
	SoftDeleteBySource(ctx context.Context, sourceChannelID, sourceMessageID int64) error
}

// SeasonRepository lets multi-season anime (scraped as e.g. "S02E01" in a
// filename) group their episodes under a season the bot can let the user
// pick, instead of flattening every season's episode 1, 2, 3... into one
// ambiguous, overlapping list.
type SeasonRepository interface {
	FindOrCreate(ctx context.Context, animeID int64, seasonNumber int) (int64, error)
	ListByAnime(ctx context.Context, animeID int64) ([]Season, error)
}

type GenreRepository interface {
	List(ctx context.Context) ([]Genre, error)
	FindOrCreateByName(ctx context.Context, nameEnglish, namePersian string) (int64, error)
}

type StudioRepository interface {
	List(ctx context.Context) ([]Studio, error)
	FindOrCreateByName(ctx context.Context, name string) (int64, error)
}

type SourceChannelRepository interface {
	Create(ctx context.Context, c *SourceChannel) (int64, error)
	Update(ctx context.Context, c *SourceChannel) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, activeOnly bool) ([]SourceChannel, error)
	GetByTelegramID(ctx context.Context, telegramChannelID int64) (*SourceChannel, error)
	UpdateLastSyncedMessageID(ctx context.Context, id, messageID int64) error
}

type UserRepository interface {
	Upsert(ctx context.Context, u *User) (*User, error)
	GetByTelegramID(ctx context.Context, telegramUserID int64) (*User, error)
	SetLanguage(ctx context.Context, telegramUserID int64, lang string) error
	IsAdmin(ctx context.Context, telegramUserID int64) (bool, string, error)
}

type FavoriteRepository interface {
	Add(ctx context.Context, userID, animeID int64) error
	Remove(ctx context.Context, userID, animeID int64) error
	List(ctx context.Context, userID int64, page, pageSize int) ([]Anime, int, error)
	IsFavorite(ctx context.Context, userID, animeID int64) (bool, error)
}

type HistoryRepository interface {
	Upsert(ctx context.Context, h *WatchHistoryEntry) error
	ListByUser(ctx context.Context, userID int64, page, pageSize int) ([]WatchHistoryEntry, int, error)
	ContinueWatching(ctx context.Context, userID int64, limit int) ([]WatchHistoryEntry, error)
}

type ChannelPostRepository interface {
	Create(ctx context.Context, p *ChannelPost) (int64, error)
	ExistsForAnime(ctx context.Context, animeID, targetChannelID int64) (bool, error)
	PendingAnimeForPosting(ctx context.Context, targetChannelID int64, limit int) ([]Anime, error)
}

type ImportLogRepository interface {
	Create(ctx context.Context, sourceChannelID *int64, telegramMessageID int64, action string, animeID, episodeID *int64, detail string, rawPayload []byte) error
}

// ListenerSessionRepository persists the MTProto userbot's Telethon
// StringSession, so it survives a PaaS's ephemeral disk (e.g. Render's
// free tier wiping local files on every redeploy) instead of requiring a
// fresh login each time.
type ListenerSessionRepository interface {
	Get(ctx context.Context) (string, error) // "" if none stored yet
	Save(ctx context.Context, sessionString string) error
}
