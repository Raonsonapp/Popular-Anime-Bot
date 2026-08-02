package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// ListenerSessionRepo persists the MTProto userbot's Telethon StringSession
// in a single-row table, so it survives PaaS redeploys that wipe local disk.
type ListenerSessionRepo struct{ db *sqlx.DB }

func NewListenerSessionRepo(db *sqlx.DB) *ListenerSessionRepo { return &ListenerSessionRepo{db: db} }

func (r *ListenerSessionRepo) Get(ctx context.Context) (string, error) {
	var sessionString string
	err := r.db.GetContext(ctx, &sessionString, `SELECT session_string FROM listener_session WHERE id = 1`)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("get listener session: %w", err)
	}
	return sessionString, nil
}

func (r *ListenerSessionRepo) Save(ctx context.Context, sessionString string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO listener_session (id, session_string, updated_at)
		VALUES (1, $1, now())
		ON CONFLICT (id) DO UPDATE SET session_string = EXCLUDED.session_string, updated_at = now()`,
		sessionString)
	if err != nil {
		return fmt.Errorf("save listener session: %w", err)
	}
	return nil
}
