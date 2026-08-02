package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"popular-anime-bot/api/internal/domain"
)

type AnimeRepo struct {
	db *sqlx.DB
}

func NewAnimeRepo(db *sqlx.DB) *AnimeRepo {
	return &AnimeRepo{db: db}
}

const animeColumns = `
	id, title_persian, title_english, title_japanese, title_original,
	synopsis_persian, synopsis_original, kind, status, year, studio_id,
	poster_storage_chat_id, poster_storage_message_id, poster_url, banner_url, trailer_url,
	episodes_count, duration_minutes, rating_score, rating_count, popularity_score, view_count,
	source_channel_id, is_published, is_deleted, created_at, updated_at`

func (r *AnimeRepo) Create(ctx context.Context, a *domain.Anime) (int64, error) {
	query := `
		INSERT INTO animes (
			title_persian, title_english, title_japanese, title_original,
			synopsis_persian, synopsis_original, kind, status, year, studio_id,
			poster_url, banner_url, trailer_url, episodes_count, duration_minutes,
			source_channel_id, is_published
		) VALUES (
			:title_persian, :title_english, :title_japanese, :title_original,
			:synopsis_persian, :synopsis_original, :kind, :status, :year, :studio_id,
			:poster_url, :banner_url, :trailer_url, :episodes_count, :duration_minutes,
			:source_channel_id, :is_published
		) RETURNING id`

	rows, err := r.db.NamedQueryContext(ctx, query, a)
	if err != nil {
		return 0, fmt.Errorf("create anime: %w", err)
	}
	defer rows.Close()

	var id int64
	if rows.Next() {
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
	}
	return id, nil
}

func (r *AnimeRepo) Update(ctx context.Context, a *domain.Anime) error {
	query := `
		UPDATE animes SET
			title_persian = :title_persian,
			title_english = :title_english,
			title_japanese = :title_japanese,
			title_original = :title_original,
			synopsis_persian = :synopsis_persian,
			synopsis_original = :synopsis_original,
			kind = :kind,
			status = :status,
			year = :year,
			studio_id = :studio_id,
			poster_storage_chat_id = :poster_storage_chat_id,
			poster_storage_message_id = :poster_storage_message_id,
			poster_url = :poster_url,
			banner_url = :banner_url,
			trailer_url = :trailer_url,
			episodes_count = :episodes_count,
			duration_minutes = :duration_minutes,
			is_published = :is_published
		WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, query, a)
	if err != nil {
		return fmt.Errorf("update anime: %w", err)
	}
	return nil
}

func (r *AnimeRepo) GetByID(ctx context.Context, id int64) (*domain.Anime, error) {
	var a domain.Anime
	query := fmt.Sprintf(`
		SELECT %s, COALESCE(s.name, '') AS studio_name
		FROM animes a
		LEFT JOIN studios s ON s.id = a.studio_id
		WHERE a.id = $1 AND a.is_deleted = false`,
		prefixColumns("a", animeColumns))

	if err := r.db.GetContext(ctx, &a, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get anime: %w", err)
	}

	genres, err := r.genresForAnime(ctx, id)
	if err != nil {
		return nil, err
	}
	a.Genres = genres

	return &a, nil
}

func (r *AnimeRepo) FindBySourceTitle(ctx context.Context, sourceChannelID int64, originalTitle string) (*domain.Anime, error) {
	var a domain.Anime
	query := fmt.Sprintf(`
		SELECT %s FROM animes
		WHERE source_channel_id = $1 AND lower(title_original) = lower($2) AND is_deleted = false
		LIMIT 1`, animeColumns)

	if err := r.db.GetContext(ctx, &a, query, sourceChannelID, originalTitle); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find anime by source title: %w", err)
	}
	return &a, nil
}

func (r *AnimeRepo) genresForAnime(ctx context.Context, animeID int64) ([]domain.Genre, error) {
	var genres []domain.Genre
	query := `
		SELECT g.id, g.name_persian, g.name_english
		FROM genres g
		JOIN anime_genres ag ON ag.genre_id = g.id
		WHERE ag.anime_id = $1
		ORDER BY g.name_persian`
	if err := r.db.SelectContext(ctx, &genres, query, animeID); err != nil {
		return nil, fmt.Errorf("genres for anime: %w", err)
	}
	return genres, nil
}

func (r *AnimeRepo) SetGenres(ctx context.Context, animeID int64, genreIDs []int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM anime_genres WHERE anime_id = $1`, animeID); err != nil {
		return err
	}
	for _, gid := range genreIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO anime_genres (anime_id, genre_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			animeID, gid); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *AnimeRepo) IncrementView(ctx context.Context, animeID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE animes SET view_count = view_count + 1 WHERE id = $1`, animeID)
	return err
}

// RecalculatePopularity applies a simple recency-weighted popularity score.
// Meant to be called periodically (e.g. hourly) by the scheduler.
func (r *AnimeRepo) RecalculatePopularity(ctx context.Context) error {
	query := `
		UPDATE animes a SET popularity_score = sub.score
		FROM (
			SELECT
				an.id,
				(
					COALESCE(recent_views.cnt, 0) * 3.0 +
					an.view_count * 0.1 +
					an.rating_score * an.rating_count * 0.5
				) AS score
			FROM animes an
			LEFT JOIN (
				SELECT anime_id, COUNT(*) AS cnt
				FROM views
				WHERE viewed_at > now() - interval '7 days'
				GROUP BY anime_id
			) recent_views ON recent_views.anime_id = an.id
		) sub
		WHERE a.id = sub.id`
	_, err := r.db.ExecContext(ctx, query)
	return err
}

func (r *AnimeRepo) SoftDelete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE animes SET is_deleted = true WHERE id = $1`, id)
	return err
}

func (r *AnimeRepo) List(ctx context.Context, f domain.AnimeFilter) ([]domain.Anime, int, error) {
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 20
	}
	if f.Page <= 0 {
		f.Page = 1
	}

	var (
		conditions []string
		args       []interface{}
		argN       = 1
	)
	conditions = append(conditions, "a.is_deleted = false", "a.is_published = true")

	if f.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"a.search_vector @@ plainto_tsquery('simple', unaccent($%d))", argN))
		args = append(args, f.Search)
		argN++
	}
	if f.Status != "" {
		conditions = append(conditions, fmt.Sprintf("a.status = $%d", argN))
		args = append(args, f.Status)
		argN++
	}
	if f.Year != 0 {
		conditions = append(conditions, fmt.Sprintf("a.year = $%d", argN))
		args = append(args, f.Year)
		argN++
	}
	if f.StudioID != 0 {
		conditions = append(conditions, fmt.Sprintf("a.studio_id = $%d", argN))
		args = append(args, f.StudioID)
		argN++
	}

	joinGenre := ""
	if f.GenreID != 0 {
		joinGenre = "JOIN anime_genres fg ON fg.anime_id = a.id"
		conditions = append(conditions, fmt.Sprintf("fg.genre_id = $%d", argN))
		args = append(args, f.GenreID)
		argN++
	}

	orderBy := "a.created_at DESC"
	switch f.SortBy {
	case "trending", "popularity":
		orderBy = "a.popularity_score DESC"
	case "top_rated":
		orderBy = "a.rating_score DESC, a.rating_count DESC"
	case "random":
		orderBy = "random()"
	case "newest":
		orderBy = "a.created_at DESC"
	}

	where := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf(`SELECT COUNT(DISTINCT a.id) FROM animes a %s WHERE %s`, joinGenre, where)
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count animes: %w", err)
	}

	limitArg := argN
	offsetArg := argN + 1
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)

	listQuery := fmt.Sprintf(`
		SELECT DISTINCT %s, COALESCE(s.name, '') AS studio_name
		FROM animes a
		LEFT JOIN studios s ON s.id = a.studio_id
		%s
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`,
		prefixColumns("a", animeColumns), joinGenre, where, orderBy, limitArg, offsetArg)

	var results []domain.Anime
	if err := r.db.SelectContext(ctx, &results, listQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("list animes: %w", err)
	}

	return results, total, nil
}

func prefixColumns(alias, cols string) string {
	parts := strings.Split(cols, ",")
	for i, p := range parts {
		parts[i] = alias + "." + strings.TrimSpace(p)
	}
	return strings.Join(parts, ", ")
}
