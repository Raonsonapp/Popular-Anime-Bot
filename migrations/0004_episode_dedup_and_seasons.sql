-- Fixes a real dedup bug: episodes.language_id is NULL for every episode
-- our ingest pipeline creates (multi-language episode tracking was never
-- wired up), and Postgres treats every NULL as distinct for uniqueness
-- purposes - so the original UNIQUE(anime_id, episode_number, quality,
-- language_id) constraint never actually caught duplicates, letting the
-- same episode accumulate multiple rows (visible in the bot as several
-- identical "Episode N" buttons). This also folds season_id into the
-- dedup key, now that multi-season anime are tracked per-season -
-- otherwise season 2's episode 1 would collide with season 1's episode 1.

-- Clean up pre-existing duplicate rows (keep only the most recently
-- updated one per group) - a unique index cannot be created while
-- duplicates still exist.
DELETE FROM episodes e
USING episodes e2
WHERE e.anime_id = e2.anime_id
  AND COALESCE(e.season_id, 0) = COALESCE(e2.season_id, 0)
  AND e.episode_number = e2.episode_number
  AND e.quality = e2.quality
  AND COALESCE(e.language_id, 0) = COALESCE(e2.language_id, 0)
  AND (e.updated_at, e.id) < (e2.updated_at, e2.id);

ALTER TABLE episodes DROP CONSTRAINT IF EXISTS episodes_anime_id_episode_number_quality_language_id_key;

CREATE UNIQUE INDEX idx_episodes_dedup ON episodes (
    anime_id, COALESCE(season_id, 0), episode_number, quality, COALESCE(language_id, 0)
);

-- Recompute after the cleanup above.
UPDATE animes a SET episodes_count = (
    SELECT COUNT(*) FROM episodes e WHERE e.anime_id = a.id AND e.is_deleted = false
);
