package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"popular-anime-bot/api/internal/domain"
)

type UserRepo struct{ db *sqlx.DB }

func NewUserRepo(db *sqlx.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) Upsert(ctx context.Context, u *domain.User) (*domain.User, error) {
	var out domain.User
	query := `
		INSERT INTO users (telegram_user_id, username, first_name, language_pref)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (telegram_user_id) DO UPDATE SET
			username = EXCLUDED.username,
			first_name = EXCLUDED.first_name,
			last_seen_at = now()
		RETURNING id, telegram_user_id, username, first_name, language_pref, is_banned, created_at, last_seen_at`

	if err := r.db.GetContext(ctx, &out, query, u.TelegramUserID, u.Username, u.FirstName, u.LanguagePref); err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}
	return &out, nil
}

func (r *UserRepo) GetByTelegramID(ctx context.Context, telegramUserID int64) (*domain.User, error) {
	var u domain.User
	err := r.db.GetContext(ctx, &u,
		`SELECT id, telegram_user_id, username, first_name, language_pref, is_banned, created_at, last_seen_at
		 FROM users WHERE telegram_user_id = $1`, telegramUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) SetLanguage(ctx context.Context, telegramUserID int64, lang string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET language_pref = $2 WHERE telegram_user_id = $1`, telegramUserID, lang)
	return err
}

func (r *UserRepo) IsAdmin(ctx context.Context, telegramUserID int64) (bool, string, error) {
	var role string
	err := r.db.GetContext(ctx, &role, `SELECT role FROM admins WHERE telegram_user_id = $1`, telegramUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, "", nil
		}
		return false, "", fmt.Errorf("check admin: %w", err)
	}
	return true, role, nil
}
