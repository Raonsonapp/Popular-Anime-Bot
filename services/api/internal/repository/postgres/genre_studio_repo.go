package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"popular-anime-bot/api/internal/domain"
)

type GenreRepo struct{ db *sqlx.DB }

func NewGenreRepo(db *sqlx.DB) *GenreRepo { return &GenreRepo{db: db} }

func (r *GenreRepo) List(ctx context.Context) ([]domain.Genre, error) {
	var genres []domain.Genre
	err := r.db.SelectContext(ctx, &genres,
		`SELECT id, name_persian, name_english FROM genres ORDER BY name_persian`)
	return genres, err
}

func (r *GenreRepo) FindOrCreateByName(ctx context.Context, nameEnglish, namePersian string) (int64, error) {
	var id int64
	err := r.db.GetContext(ctx, &id, `SELECT id FROM genres WHERE lower(name_english) = lower($1)`, nameEnglish)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("lookup genre: %w", err)
	}

	err = r.db.GetContext(ctx, &id,
		`INSERT INTO genres (name_english, name_persian) VALUES ($1, $2) RETURNING id`,
		nameEnglish, namePersian)
	if err != nil {
		return 0, fmt.Errorf("create genre: %w", err)
	}
	return id, nil
}

type StudioRepo struct{ db *sqlx.DB }

func NewStudioRepo(db *sqlx.DB) *StudioRepo { return &StudioRepo{db: db} }

func (r *StudioRepo) List(ctx context.Context) ([]domain.Studio, error) {
	var studios []domain.Studio
	err := r.db.SelectContext(ctx, &studios, `SELECT id, name FROM studios ORDER BY name`)
	return studios, err
}

func (r *StudioRepo) FindOrCreateByName(ctx context.Context, name string) (int64, error) {
	var id int64
	err := r.db.GetContext(ctx, &id, `SELECT id FROM studios WHERE lower(name) = lower($1)`, name)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("lookup studio: %w", err)
	}

	err = r.db.GetContext(ctx, &id, `INSERT INTO studios (name) VALUES ($1) RETURNING id`, name)
	if err != nil {
		return 0, fmt.Errorf("create studio: %w", err)
	}
	return id, nil
}
