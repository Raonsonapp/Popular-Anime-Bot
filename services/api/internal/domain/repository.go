package domain

import "context"

// AnimeRepository persists and queries the anime catalog.
type AnimeRepository interface {
	Create(ctx context.Context, a *Anime) (int64, error)
	Update(ctx context.Context, a *Anime) error
	GetByID(ctx context.Context, id int64) (*Anime, error)
	FindBySourceTitle(ctx context.Context, sourceChannelID int64, originalTitle string) (*Anime, error)
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
	ListByAnime(ctx context.Context, animeID int64, page, pageSize int) ([]Episode, int, error)
	SoftDeleteBySource(ctx context.Context, sourceChannelID, sourceMessageID int64) error
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
