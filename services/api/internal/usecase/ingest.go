package usecase

import (
	"context"
	"fmt"

	"popular-anime-bot/api/internal/domain"
)

// IngestService is consumed by the MTProto listener to turn raw channel
// posts into catalog rows. It is intentionally idempotent: re-processing
// the same source message must not create duplicates.
type IngestService struct {
	Anime         domain.AnimeRepository
	Episode       domain.EpisodeRepository
	Season        domain.SeasonRepository
	Genre         domain.GenreRepository
	Studio        domain.StudioRepository
	SourceChannel domain.SourceChannelRepository
	ImportLog     domain.ImportLogRepository
}

func NewIngestService(
	a domain.AnimeRepository,
	e domain.EpisodeRepository,
	sn domain.SeasonRepository,
	g domain.GenreRepository,
	s domain.StudioRepository,
	sc domain.SourceChannelRepository,
	il domain.ImportLogRepository,
) *IngestService {
	return &IngestService{Anime: a, Episode: e, Season: sn, Genre: g, Studio: s, SourceChannel: sc, ImportLog: il}
}

type UpsertAnimeInput struct {
	SourceChannelID        int64
	TitleOriginal          string
	TitlePersian           string
	TitleEnglish           string
	TitleJapanese          string
	SynopsisPersian        string
	SynopsisOriginal       string
	Kind                   domain.AnimeKind
	Status                 domain.AnimeStatus
	Year                   int
	StudioName             string
	GenreNamesEnglish      []string
	GenreNamesPersian      []string
	PosterStorageChatID    int64
	PosterStorageMessageID int64
	DurationMinutes        int
	AutoPublish            bool
}

// UpsertAnime finds an existing anime with a matching original title -
// regardless of which source channel it came from, so the same anime
// scraped from several channels merges into one catalog entry instead of
// duplicating - or creates a new one.
func (s *IngestService) UpsertAnime(ctx context.Context, in UpsertAnimeInput) (*domain.Anime, error) {
	existing, err := s.Anime.FindByTitle(ctx, in.TitleOriginal)
	if err != nil {
		return nil, fmt.Errorf("lookup existing anime: %w", err)
	}

	if existing == nil {
		if in.Kind == "" {
			in.Kind = domain.KindTV
		}
		if in.Status == "" {
			in.Status = domain.StatusOngoing
		}
	}

	var studioID *int64
	if in.StudioName != "" {
		id, err := s.Studio.FindOrCreateByName(ctx, in.StudioName)
		if err != nil {
			return nil, fmt.Errorf("resolve studio: %w", err)
		}
		studioID = &id
	}

	a := &domain.Anime{
		Title:           in.TitlePersian,
		TitleOriginal:   &in.TitleOriginal,
		Kind:            in.Kind,
		Status:          in.Status,
		StudioID:        studioID,
		SourceChannelID: &in.SourceChannelID,
		IsPublished:     in.AutoPublish,
	}
	if in.TitleEnglish != "" {
		a.TitleEnglish = &in.TitleEnglish
	}
	if in.TitleJapanese != "" {
		a.TitleJapanese = &in.TitleJapanese
	}
	if in.SynopsisPersian != "" {
		a.SynopsisPersian = &in.SynopsisPersian
	}
	if in.SynopsisOriginal != "" {
		a.SynopsisOriginal = &in.SynopsisOriginal
	}
	if in.Year != 0 {
		a.Year = &in.Year
	}
	if in.DurationMinutes != 0 {
		a.DurationMinutes = &in.DurationMinutes
	}
	if in.PosterStorageChatID != 0 {
		a.PosterStorageChatID = &in.PosterStorageChatID
	}
	if in.PosterStorageMessageID != 0 {
		a.PosterStorageMessageID = &in.PosterStorageMessageID
	}

	if existing != nil {
		// A bare episode-drop post only carries a title; merge onto the
		// existing row instead of overwriting so it doesn't blank out
		// richer metadata set by an earlier announcement post.
		a.ID = existing.ID
		a.IsPublished = existing.IsPublished || in.AutoPublish
		if in.TitlePersian == "" {
			a.Title = existing.Title
		}
		if a.TitleEnglish == nil {
			a.TitleEnglish = existing.TitleEnglish
		}
		if a.TitleJapanese == nil {
			a.TitleJapanese = existing.TitleJapanese
		}
		if a.SynopsisPersian == nil {
			a.SynopsisPersian = existing.SynopsisPersian
		}
		if a.SynopsisOriginal == nil {
			a.SynopsisOriginal = existing.SynopsisOriginal
		}
		if a.Year == nil {
			a.Year = existing.Year
		}
		if a.DurationMinutes == nil {
			a.DurationMinutes = existing.DurationMinutes
		}
		if a.StudioID == nil {
			a.StudioID = existing.StudioID
		}
		if a.PosterStorageChatID == nil {
			a.PosterStorageChatID = existing.PosterStorageChatID
		}
		if a.PosterStorageMessageID == nil {
			a.PosterStorageMessageID = existing.PosterStorageMessageID
		}
		if in.Status == "" {
			a.Status = existing.Status
		}
		if in.Kind == "" {
			a.Kind = existing.Kind
		}
		if err := s.Anime.Update(ctx, a); err != nil {
			return nil, fmt.Errorf("update anime: %w", err)
		}
	} else {
		id, err := s.Anime.Create(ctx, a)
		if err != nil {
			return nil, fmt.Errorf("create anime: %w", err)
		}
		a.ID = id
	}

	if len(in.GenreNamesEnglish) > 0 {
		genreIDs := make([]int64, 0, len(in.GenreNamesEnglish))
		for i, nameEn := range in.GenreNamesEnglish {
			namePersian := nameEn
			if i < len(in.GenreNamesPersian) && in.GenreNamesPersian[i] != "" {
				namePersian = in.GenreNamesPersian[i]
			}
			gid, err := s.Genre.FindOrCreateByName(ctx, nameEn, namePersian)
			if err != nil {
				return nil, fmt.Errorf("resolve genre %q: %w", nameEn, err)
			}
			genreIDs = append(genreIDs, gid)
		}
		if err := s.Anime.SetGenres(ctx, a.ID, genreIDs); err != nil {
			return nil, fmt.Errorf("set genres: %w", err)
		}
	}

	return s.Anime.GetByID(ctx, a.ID)
}

type UpsertEpisodeInput struct {
	AnimeID          int64
	SeasonNumber     int // 0 means "no season info" - episode_number is used as-is, ungrouped
	EpisodeNumber    int
	Title            string
	Quality          string
	LanguageID       int64
	StorageChatID    int64
	StorageMessageID int64
	SourceChannelID  int64
	SourceMessageID  int64
	DurationSeconds  int
	SizeBytes        int64
}

func (s *IngestService) UpsertEpisode(ctx context.Context, in UpsertEpisodeInput) (*domain.Episode, error) {
	e := &domain.Episode{
		AnimeID:          in.AnimeID,
		EpisodeNumber:    in.EpisodeNumber,
		Quality:          in.Quality,
		StorageChatID:    in.StorageChatID,
		StorageMessageID: in.StorageMessageID,
		SourceChannelID:  &in.SourceChannelID,
		SourceMessageID:  &in.SourceMessageID,
	}
	if in.SeasonNumber > 0 {
		seasonID, err := s.Season.FindOrCreate(ctx, in.AnimeID, in.SeasonNumber)
		if err != nil {
			return nil, fmt.Errorf("resolve season: %w", err)
		}
		e.SeasonID = &seasonID
	}
	if in.Title != "" {
		e.Title = &in.Title
	}
	if in.LanguageID != 0 {
		e.LanguageID = &in.LanguageID
	}
	if in.DurationSeconds != 0 {
		e.DurationSeconds = &in.DurationSeconds
	}
	if in.SizeBytes != 0 {
		e.SizeBytes = &in.SizeBytes
	}
	if e.Quality == "" {
		e.Quality = "720p"
	}

	id, err := s.Episode.Create(ctx, e)
	if err != nil {
		return nil, fmt.Errorf("upsert episode: %w", err)
	}
	e.ID = id
	return e, nil
}

func (s *IngestService) MarkEpisodeDeleted(ctx context.Context, sourceChannelID, sourceMessageID int64) error {
	return s.Episode.SoftDeleteBySource(ctx, sourceChannelID, sourceMessageID)
}

func (s *IngestService) DeleteAnime(ctx context.Context, animeID int64) error {
	return s.Anime.SoftDelete(ctx, animeID)
}

func (s *IngestService) LogImport(ctx context.Context, sourceChannelID *int64, telegramMessageID int64, action string, animeID, episodeID *int64, detail string, rawPayload []byte) {
	_ = s.ImportLog.Create(ctx, sourceChannelID, telegramMessageID, action, animeID, episodeID, detail, rawPayload)
}
