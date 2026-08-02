package usecase

import (
	"context"

	"popular-anime-bot/api/internal/domain"
)

// PublishService is consumed by the scheduler to drive the daily
// showcase-channel post generator.
type PublishService struct {
	Anime       domain.AnimeRepository
	ChannelPost domain.ChannelPostRepository
}

func NewPublishService(a domain.AnimeRepository, cp domain.ChannelPostRepository) *PublishService {
	return &PublishService{Anime: a, ChannelPost: cp}
}

func (s *PublishService) PendingAnime(ctx context.Context, targetChannelID int64, limit int) ([]domain.Anime, error) {
	return s.ChannelPost.PendingAnimeForPosting(ctx, targetChannelID, limit)
}

func (s *PublishService) RecordPost(ctx context.Context, animeID, targetChannelID int64, telegramMessageID int64) error {
	_, err := s.ChannelPost.Create(ctx, &domain.ChannelPost{
		AnimeID:           animeID,
		TargetChannelID:   targetChannelID,
		TelegramMessageID: &telegramMessageID,
	})
	return err
}

func (s *PublishService) RefreshPopularity(ctx context.Context) error {
	return s.Anime.RecalculatePopularity(ctx)
}
