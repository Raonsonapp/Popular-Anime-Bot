package usecase

import (
	"context"
	"fmt"

	"popular-anime-bot/api/internal/domain"
)

type CatalogService struct {
	Anime   domain.AnimeRepository
	Episode domain.EpisodeRepository
	Genre   domain.GenreRepository
	Studio  domain.StudioRepository
}

func NewCatalogService(a domain.AnimeRepository, e domain.EpisodeRepository, g domain.GenreRepository, s domain.StudioRepository) *CatalogService {
	return &CatalogService{Anime: a, Episode: e, Genre: g, Studio: s}
}

func (c *CatalogService) GetAnime(ctx context.Context, id int64) (*domain.Anime, error) {
	a, err := c.Anime.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get anime: %w", err)
	}
	return a, nil
}

func (c *CatalogService) ListAnime(ctx context.Context, f domain.AnimeFilter) ([]domain.Anime, int, error) {
	return c.Anime.List(ctx, f)
}

func (c *CatalogService) RandomAnime(ctx context.Context) (*domain.Anime, error) {
	list, _, err := c.Anime.List(ctx, domain.AnimeFilter{SortBy: "random", Page: 1, PageSize: 1})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return &list[0], nil
}

func (c *CatalogService) ListEpisodes(ctx context.Context, animeID int64, page, pageSize int) ([]domain.Episode, int, error) {
	return c.Episode.ListByAnime(ctx, animeID, page, pageSize)
}

func (c *CatalogService) RecordView(ctx context.Context, animeID int64) error {
	return c.Anime.IncrementView(ctx, animeID)
}

func (c *CatalogService) ListGenres(ctx context.Context) ([]domain.Genre, error) {
	return c.Genre.List(ctx)
}

func (c *CatalogService) ListStudios(ctx context.Context) ([]domain.Studio, error) {
	return c.Studio.List(ctx)
}

func (c *CatalogService) GetEpisode(ctx context.Context, id int64) (*domain.Episode, error) {
	return c.Episode.GetByID(ctx, id)
}
