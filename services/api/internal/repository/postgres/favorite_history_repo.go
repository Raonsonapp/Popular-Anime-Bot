package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"popular-anime-bot/api/internal/domain"
)

type FavoriteRepo struct{ db *sqlx.DB }

func NewFavoriteRepo(db *sqlx.DB) *FavoriteRepo { return &FavoriteRepo{db: db} }

func (r *FavoriteRepo) Add(ctx context.Context, userID, animeID int64) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO favorites (user_id, anime_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, animeID)
	return err
}

func (r *FavoriteRepo) Remove(ctx context.Context, userID, animeID int64) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM favorites WHERE user_id = $1 AND anime_id = $2`, userID, animeID)
	return err
}

func (r *FavoriteRepo) IsFavorite(ctx context.Context, userID, animeID int64) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM favorites WHERE user_id = $1 AND anime_id = $2)`, userID, animeID)
	return exists, err
}

func (r *FavoriteRepo) List(ctx context.Context, userID int64, page, pageSize int) ([]domain.Anime, int, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}

	var total int
	if err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM favorites WHERE user_id = $1`, userID); err != nil {
		return nil, 0, fmt.Errorf("count favorites: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT %s FROM animes a
		JOIN favorites f ON f.anime_id = a.id
		WHERE f.user_id = $1 AND a.is_deleted = false
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3`, prefixColumns("a", animeColumns))

	var animes []domain.Anime
	if err := r.db.SelectContext(ctx, &animes, query, userID, pageSize, (page-1)*pageSize); err != nil {
		return nil, 0, fmt.Errorf("list favorites: %w", err)
	}
	return animes, total, nil
}

type HistoryRepo struct{ db *sqlx.DB }

func NewHistoryRepo(db *sqlx.DB) *HistoryRepo { return &HistoryRepo{db: db} }

func (r *HistoryRepo) Upsert(ctx context.Context, h *domain.WatchHistoryEntry) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO watch_history (user_id, episode_id, progress_seconds, completed, watched_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (user_id, episode_id) DO UPDATE SET
			progress_seconds = EXCLUDED.progress_seconds,
			completed = EXCLUDED.completed,
			watched_at = now()`,
		h.UserID, h.EpisodeID, h.ProgressSeconds, h.Completed)
	return err
}

func (r *HistoryRepo) ListByUser(ctx context.Context, userID int64, page, pageSize int) ([]domain.WatchHistoryEntry, int, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}

	var total int
	if err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM watch_history WHERE user_id = $1`, userID); err != nil {
		return nil, 0, err
	}

	var entries []domain.WatchHistoryEntry
	err := r.db.SelectContext(ctx, &entries, `
		SELECT id, user_id, episode_id, progress_seconds, completed, watched_at
		FROM watch_history WHERE user_id = $1
		ORDER BY watched_at DESC
		LIMIT $2 OFFSET $3`, userID, pageSize, (page-1)*pageSize)
	return entries, total, err
}

func (r *HistoryRepo) ContinueWatching(ctx context.Context, userID int64, limit int) ([]domain.WatchHistoryEntry, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	var entries []domain.WatchHistoryEntry
	err := r.db.SelectContext(ctx, &entries, `
		SELECT id, user_id, episode_id, progress_seconds, completed, watched_at
		FROM watch_history
		WHERE user_id = $1 AND completed = false
		ORDER BY watched_at DESC
		LIMIT $2`, userID, limit)
	return entries, err
}
