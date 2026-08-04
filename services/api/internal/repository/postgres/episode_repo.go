package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"popular-anime-bot/api/internal/domain"
)

type EpisodeRepo struct {
	db *sqlx.DB
}

func NewEpisodeRepo(db *sqlx.DB) *EpisodeRepo {
	return &EpisodeRepo{db: db}
}

const episodeColumns = `
	id, anime_id, season_id, episode_number, title, quality, language_id,
	storage_chat_id, storage_message_id, source_channel_id, source_message_id,
	duration_seconds, size_bytes, is_deleted, created_at, updated_at`

func (r *EpisodeRepo) Create(ctx context.Context, e *domain.Episode) (int64, error) {
	query := `
		INSERT INTO episodes (
			anime_id, season_id, episode_number, title, quality, language_id,
			storage_chat_id, storage_message_id, source_channel_id, source_message_id,
			duration_seconds, size_bytes
		) VALUES (
			:anime_id, :season_id, :episode_number, :title, :quality, :language_id,
			:storage_chat_id, :storage_message_id, :source_channel_id, :source_message_id,
			:duration_seconds, :size_bytes
		)
		ON CONFLICT (anime_id, (COALESCE(season_id, 0)), episode_number, quality, (COALESCE(language_id, 0)))
		DO UPDATE SET
			storage_chat_id = EXCLUDED.storage_chat_id,
			storage_message_id = EXCLUDED.storage_message_id,
			season_id = EXCLUDED.season_id,
			updated_at = now()
		RETURNING id`

	rows, err := r.db.NamedQueryContext(ctx, query, e)
	if err != nil {
		return 0, fmt.Errorf("create episode: %w", err)
	}
	defer rows.Close()

	var id int64
	if rows.Next() {
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
	}

	if _, err := r.db.ExecContext(ctx,
		`UPDATE animes SET episodes_count = (
			SELECT COUNT(*) FROM episodes WHERE anime_id = $1 AND is_deleted = false
		) WHERE id = $1`, e.AnimeID); err != nil {
		return id, fmt.Errorf("refresh episode count: %w", err)
	}

	return id, nil
}

func (r *EpisodeRepo) Update(ctx context.Context, e *domain.Episode) error {
	query := `
		UPDATE episodes SET
			title = :title,
			quality = :quality,
			language_id = :language_id,
			storage_chat_id = :storage_chat_id,
			storage_message_id = :storage_message_id,
			duration_seconds = :duration_seconds,
			size_bytes = :size_bytes,
			updated_at = now()
		WHERE id = :id`
	_, err := r.db.NamedExecContext(ctx, query, e)
	return err
}

func (r *EpisodeRepo) GetByID(ctx context.Context, id int64) (*domain.Episode, error) {
	var e domain.Episode
	query := fmt.Sprintf(`SELECT %s FROM episodes WHERE id = $1 AND is_deleted = false`, episodeColumns)
	if err := r.db.GetContext(ctx, &e, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get episode: %w", err)
	}
	return &e, nil
}

func (r *EpisodeRepo) FindBySource(ctx context.Context, sourceChannelID, sourceMessageID int64) (*domain.Episode, error) {
	var e domain.Episode
	query := fmt.Sprintf(`
		SELECT %s FROM episodes
		WHERE source_channel_id = $1 AND source_message_id = $2`, episodeColumns)
	if err := r.db.GetContext(ctx, &e, query, sourceChannelID, sourceMessageID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find episode by source: %w", err)
	}
	return &e, nil
}

func (r *EpisodeRepo) ListByAnime(ctx context.Context, animeID, seasonID int64, page, pageSize int) ([]domain.Episode, int, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}

	var total int
	if err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM episodes WHERE anime_id = $1 AND is_deleted = false AND ($2 = 0 OR season_id = $2)`,
		animeID, seasonID); err != nil {
		return nil, 0, fmt.Errorf("count episodes: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT %s FROM episodes
		WHERE anime_id = $1 AND is_deleted = false AND ($2 = 0 OR season_id = $2)
		ORDER BY episode_number ASC
		LIMIT $3 OFFSET $4`, episodeColumns)

	var episodes []domain.Episode
	if err := r.db.SelectContext(ctx, &episodes, query, animeID, seasonID, pageSize, (page-1)*pageSize); err != nil {
		return nil, 0, fmt.Errorf("list episodes: %w", err)
	}
	return episodes, total, nil
}

func (r *EpisodeRepo) SoftDeleteBySource(ctx context.Context, sourceChannelID, sourceMessageID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE episodes SET is_deleted = true, updated_at = now()
		 WHERE source_channel_id = $1 AND source_message_id = $2`,
		sourceChannelID, sourceMessageID)
	return err
}
