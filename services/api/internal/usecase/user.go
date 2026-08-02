package usecase

import (
	"context"
	"fmt"

	"popular-anime-bot/api/internal/domain"
)

type UserService struct {
	User     domain.UserRepository
	Favorite domain.FavoriteRepository
	History  domain.HistoryRepository
}

func NewUserService(u domain.UserRepository, f domain.FavoriteRepository, h domain.HistoryRepository) *UserService {
	return &UserService{User: u, Favorite: f, History: h}
}

func (s *UserService) Touch(ctx context.Context, telegramUserID int64, username, firstName, lang string) (*domain.User, error) {
	u := &domain.User{TelegramUserID: telegramUserID, LanguagePref: lang}
	if username != "" {
		u.Username = &username
	}
	if firstName != "" {
		u.FirstName = &firstName
	}
	if u.LanguagePref == "" {
		u.LanguagePref = "tg"
	}
	out, err := s.User.Upsert(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("touch user: %w", err)
	}
	return out, nil
}

func (s *UserService) SetLanguage(ctx context.Context, telegramUserID int64, lang string) error {
	return s.User.SetLanguage(ctx, telegramUserID, lang)
}

func (s *UserService) ToggleFavorite(ctx context.Context, userID, animeID int64) (bool, error) {
	isFav, err := s.Favorite.IsFavorite(ctx, userID, animeID)
	if err != nil {
		return false, err
	}
	if isFav {
		return false, s.Favorite.Remove(ctx, userID, animeID)
	}
	return true, s.Favorite.Add(ctx, userID, animeID)
}

func (s *UserService) ListFavorites(ctx context.Context, userID int64, page, pageSize int) ([]domain.Anime, int, error) {
	return s.Favorite.List(ctx, userID, page, pageSize)
}

func (s *UserService) RecordProgress(ctx context.Context, userID, episodeID int64, progressSeconds int, completed bool) error {
	return s.History.Upsert(ctx, &domain.WatchHistoryEntry{
		UserID:          userID,
		EpisodeID:       episodeID,
		ProgressSeconds: progressSeconds,
		Completed:       completed,
	})
}

func (s *UserService) ListHistory(ctx context.Context, userID int64, page, pageSize int) ([]domain.WatchHistoryEntry, int, error) {
	return s.History.ListByUser(ctx, userID, page, pageSize)
}

func (s *UserService) ContinueWatching(ctx context.Context, userID int64) ([]domain.WatchHistoryEntry, error) {
	return s.History.ContinueWatching(ctx, userID, 10)
}
