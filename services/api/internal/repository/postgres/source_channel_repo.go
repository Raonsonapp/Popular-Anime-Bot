package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"popular-anime-bot/api/internal/domain"
)

type SourceChannelRepo struct{ db *sqlx.DB }

func NewSourceChannelRepo(db *sqlx.DB) *SourceChannelRepo { return &SourceChannelRepo{db: db} }

const sourceChannelColumns = `id, telegram_channel_id, username, title, source_language, is_active, last_synced_message_id, created_at`

func (r *SourceChannelRepo) Create(ctx context.Context, c *domain.SourceChannel) (int64, error) {
	var id int64
	err := r.db.GetContext(ctx, &id, `
		INSERT INTO source_channels (telegram_channel_id, username, title, source_language, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		c.TelegramChannelID, c.Username, c.Title, c.SourceLanguage, c.IsActive)
	if err != nil {
		return 0, fmt.Errorf("create source channel: %w", err)
	}
	return id, nil
}

func (r *SourceChannelRepo) Update(ctx context.Context, c *domain.SourceChannel) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE source_channels SET
			username = $2, title = $3, source_language = $4, is_active = $5, updated_at = now()
		WHERE id = $1`,
		c.ID, c.Username, c.Title, c.SourceLanguage, c.IsActive)
	return err
}

func (r *SourceChannelRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM source_channels WHERE id = $1`, id)
	return err
}

func (r *SourceChannelRepo) List(ctx context.Context, activeOnly bool) ([]domain.SourceChannel, error) {
	query := fmt.Sprintf(`SELECT %s FROM source_channels`, sourceChannelColumns)
	if activeOnly {
		query += ` WHERE is_active = true`
	}
	query += ` ORDER BY created_at ASC`

	var channels []domain.SourceChannel
	err := r.db.SelectContext(ctx, &channels, query)
	return channels, err
}

func (r *SourceChannelRepo) GetByTelegramID(ctx context.Context, telegramChannelID int64) (*domain.SourceChannel, error) {
	var c domain.SourceChannel
	query := fmt.Sprintf(`SELECT %s FROM source_channels WHERE telegram_channel_id = $1`, sourceChannelColumns)
	if err := r.db.GetContext(ctx, &c, query, telegramChannelID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get source channel: %w", err)
	}
	return &c, nil
}

func (r *SourceChannelRepo) UpdateLastSyncedMessageID(ctx context.Context, id, messageID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE source_channels SET last_synced_message_id = $2, updated_at = now() WHERE id = $1`,
		id, messageID)
	return err
}
