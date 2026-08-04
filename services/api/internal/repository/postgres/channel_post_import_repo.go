package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"popular-anime-bot/api/internal/domain"
)

type ChannelPostRepo struct{ db *sqlx.DB }

func NewChannelPostRepo(db *sqlx.DB) *ChannelPostRepo { return &ChannelPostRepo{db: db} }

func (r *ChannelPostRepo) Create(ctx context.Context, p *domain.ChannelPost) (int64, error) {
	var id int64
	err := r.db.GetContext(ctx, &id, `
		INSERT INTO channel_posts (anime_id, target_channel_id, telegram_message_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (anime_id, target_channel_id) DO UPDATE SET telegram_message_id = EXCLUDED.telegram_message_id
		RETURNING id`, p.AnimeID, p.TargetChannelID, p.TelegramMessageID)
	if err != nil {
		return 0, fmt.Errorf("create channel post: %w", err)
	}
	return id, nil
}

func (r *ChannelPostRepo) ExistsForAnime(ctx context.Context, animeID, targetChannelID int64) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM channel_posts WHERE anime_id = $1 AND target_channel_id = $2)`,
		animeID, targetChannelID)
	return exists, err
}

func (r *ChannelPostRepo) PendingAnimeForPosting(ctx context.Context, targetChannelID int64, limit int) ([]domain.Anime, error) {
	if limit <= 0 || limit > 50 {
		limit = 5
	}
	query := fmt.Sprintf(`
		SELECT %s, COALESCE(s.name, '') AS studio_name
		FROM animes a
		LEFT JOIN studios s ON s.id = a.studio_id
		WHERE a.is_published = true AND a.is_deleted = false AND a.episodes_count > 0
		AND NOT EXISTS (
			SELECT 1 FROM channel_posts cp WHERE cp.anime_id = a.id AND cp.target_channel_id = $1
		)
		ORDER BY a.created_at ASC
		LIMIT $2`, prefixColumns("a", animeColumns))

	var animes []domain.Anime
	err := r.db.SelectContext(ctx, &animes, query, targetChannelID, limit)
	return animes, err
}

type ImportLogRepo struct{ db *sqlx.DB }

func NewImportLogRepo(db *sqlx.DB) *ImportLogRepo { return &ImportLogRepo{db: db} }

func (r *ImportLogRepo) Create(ctx context.Context, sourceChannelID *int64, telegramMessageID int64, action string, animeID, episodeID *int64, detail string, rawPayload []byte) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO import_logs (source_channel_id, telegram_message_id, action, anime_id, episode_id, detail, raw_payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		sourceChannelID, telegramMessageID, action, animeID, episodeID, detail, rawPayload)
	return err
}
