package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"popular-anime-bot/api/internal/domain"
)

type SeasonRepo struct{ db *sqlx.DB }

func NewSeasonRepo(db *sqlx.DB) *SeasonRepo { return &SeasonRepo{db: db} }

func (r *SeasonRepo) FindOrCreate(ctx context.Context, animeID int64, seasonNumber int) (int64, error) {
	var id int64
	err := r.db.GetContext(ctx, &id, `
		INSERT INTO seasons (anime_id, season_number)
		VALUES ($1, $2)
		ON CONFLICT (anime_id, season_number) DO UPDATE SET season_number = EXCLUDED.season_number
		RETURNING id`, animeID, seasonNumber)
	if err != nil {
		return 0, fmt.Errorf("find or create season: %w", err)
	}
	return id, nil
}

func (r *SeasonRepo) ListByAnime(ctx context.Context, animeID int64) ([]domain.Season, error) {
	var seasons []domain.Season
	query := `
		SELECT s.id, s.anime_id, s.season_number, s.title, s.year,
			(SELECT COUNT(*) FROM episodes e WHERE e.season_id = s.id AND e.is_deleted = false) AS episodes_count
		FROM seasons s
		WHERE s.anime_id = $1
		ORDER BY s.season_number ASC`
	if err := r.db.SelectContext(ctx, &seasons, query, animeID); err != nil {
		return nil, fmt.Errorf("list seasons: %w", err)
	}
	return seasons, nil
}
