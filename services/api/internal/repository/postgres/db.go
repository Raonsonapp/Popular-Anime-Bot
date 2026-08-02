package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Connect retries the initial ping with backoff, since serverless Postgres
// providers (e.g. Neon's free tier) suspend when idle and take a few
// seconds to wake on the first connection after a deploy/restart.
func Connect(ctx context.Context, dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	backoff := []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second, 30 * time.Second}
	var pingErr error
	for attempt := 0; attempt <= len(backoff); attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		pingErr = db.PingContext(pingCtx)
		cancel()
		if pingErr == nil {
			return db, nil
		}
		if attempt < len(backoff) {
			time.Sleep(backoff[attempt])
		}
	}

	return nil, fmt.Errorf("ping postgres: %w", pingErr)
}
