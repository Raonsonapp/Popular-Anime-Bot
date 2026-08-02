-- Popular Anime Bot - core schema
-- PostgreSQL 15+

CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS unaccent;

-- ========== Reference tables ==========

CREATE TABLE studios (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE genres (
    id              BIGSERIAL PRIMARY KEY,
    name_persian    TEXT NOT NULL,
    name_english    TEXT NOT NULL UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE languages (
    id      BIGSERIAL PRIMARY KEY,
    code    TEXT NOT NULL UNIQUE, -- fa, ru, en, ja
    name    TEXT NOT NULL
);

-- ========== Source ingestion ==========

-- Telegram channels the MTProto userbot monitors for content
CREATE TABLE source_channels (
    id                      BIGSERIAL PRIMARY KEY,
    telegram_channel_id     BIGINT NOT NULL UNIQUE,
    username                TEXT,
    title                   TEXT NOT NULL,
    source_language         TEXT NOT NULL DEFAULT 'ru', -- ru, en, fa
    is_active               BOOLEAN NOT NULL DEFAULT true,
    last_synced_message_id  BIGINT NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Every message the listener processes, for auditing / dedup / reprocessing
CREATE TABLE import_logs (
    id                  BIGSERIAL PRIMARY KEY,
    source_channel_id   BIGINT REFERENCES source_channels(id) ON DELETE SET NULL,
    telegram_message_id BIGINT NOT NULL,
    action              TEXT NOT NULL CHECK (action IN ('new', 'edited', 'deleted', 'skipped', 'error')),
    anime_id            BIGINT,
    episode_id          BIGINT,
    detail              TEXT,
    raw_payload         JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_import_logs_source_msg ON import_logs(source_channel_id, telegram_message_id);

-- ========== Anime catalog ==========

CREATE TABLE animes (
    id                  BIGSERIAL PRIMARY KEY,

    title_persian       TEXT NOT NULL,
    title_english       TEXT,
    title_japanese      TEXT,
    title_original      TEXT, -- as scraped from the source channel, pre-translation

    synopsis_persian    TEXT,
    synopsis_original   TEXT,

    kind                TEXT NOT NULL DEFAULT 'tv' CHECK (kind IN ('tv', 'movie', 'ova', 'special')),
    status              TEXT NOT NULL DEFAULT 'ongoing' CHECK (status IN ('ongoing', 'completed', 'announced')),

    year                INT,
    studio_id           BIGINT REFERENCES studios(id) ON DELETE SET NULL,

    poster_storage_chat_id    BIGINT,
    poster_storage_message_id BIGINT,
    poster_url                TEXT,
    banner_url                TEXT,
    trailer_url               TEXT,

    episodes_count      INT NOT NULL DEFAULT 0,
    duration_minutes    INT,

    rating_score        NUMERIC(3,1) NOT NULL DEFAULT 0.0,
    rating_count        INT NOT NULL DEFAULT 0,
    popularity_score    NUMERIC(10,2) NOT NULL DEFAULT 0.0,
    view_count          BIGINT NOT NULL DEFAULT 0,

    source_channel_id   BIGINT REFERENCES source_channels(id) ON DELETE SET NULL,
    is_published        BOOLEAN NOT NULL DEFAULT false, -- approved & shown to users
    is_deleted          BOOLEAN NOT NULL DEFAULT false,

    search_vector       tsvector,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_animes_search ON animes USING GIN (search_vector);
CREATE INDEX idx_animes_title_trgm ON animes USING GIN (title_persian gin_trgm_ops, title_english gin_trgm_ops);
CREATE INDEX idx_animes_status ON animes(status) WHERE is_deleted = false;
CREATE INDEX idx_animes_popularity ON animes(popularity_score DESC) WHERE is_deleted = false;

CREATE OR REPLACE FUNCTION animes_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('simple', unaccent(coalesce(NEW.title_persian, ''))), 'A') ||
        setweight(to_tsvector('simple', unaccent(coalesce(NEW.title_english, ''))), 'A') ||
        setweight(to_tsvector('simple', unaccent(coalesce(NEW.title_japanese, ''))), 'B') ||
        setweight(to_tsvector('simple', unaccent(coalesce(NEW.synopsis_persian, ''))), 'C');
    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_animes_search_vector
    BEFORE INSERT OR UPDATE ON animes
    FOR EACH ROW EXECUTE FUNCTION animes_search_vector_update();

CREATE TABLE anime_genres (
    anime_id    BIGINT NOT NULL REFERENCES animes(id) ON DELETE CASCADE,
    genre_id    BIGINT NOT NULL REFERENCES genres(id) ON DELETE CASCADE,
    PRIMARY KEY (anime_id, genre_id)
);

CREATE TABLE seasons (
    id              BIGSERIAL PRIMARY KEY,
    anime_id        BIGINT NOT NULL REFERENCES animes(id) ON DELETE CASCADE,
    season_number   INT NOT NULL,
    title           TEXT,
    year            INT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (anime_id, season_number)
);

CREATE TABLE characters (
    id                          BIGSERIAL PRIMARY KEY,
    name                        TEXT NOT NULL,
    description                 TEXT,
    image_storage_chat_id       BIGINT,
    image_storage_message_id    BIGINT,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE voice_actors (
    id                          BIGSERIAL PRIMARY KEY,
    name                        TEXT NOT NULL,
    image_storage_chat_id       BIGINT,
    image_storage_message_id    BIGINT,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE anime_characters (
    anime_id        BIGINT NOT NULL REFERENCES animes(id) ON DELETE CASCADE,
    character_id    BIGINT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    voice_actor_id  BIGINT REFERENCES voice_actors(id) ON DELETE SET NULL,
    role            TEXT NOT NULL DEFAULT 'supporting' CHECK (role IN ('main', 'supporting')),
    PRIMARY KEY (anime_id, character_id)
);

-- ========== Episodes ==========
-- Files are delivered to end users via the Bot API's copyMessage, referencing
-- a message that lives in a private "storage channel" the bot is admin of.
-- The MTProto userbot forwards matched posts from source channels into that
-- storage channel; the bot never needs to touch a Telethon file reference.

CREATE TABLE episodes (
    id                          BIGSERIAL PRIMARY KEY,
    anime_id                    BIGINT NOT NULL REFERENCES animes(id) ON DELETE CASCADE,
    season_id                   BIGINT REFERENCES seasons(id) ON DELETE SET NULL,

    episode_number              INT NOT NULL,
    title                       TEXT,
    quality                     TEXT NOT NULL DEFAULT '720p' CHECK (quality IN ('480p', '720p', '1080p', 'other')),
    language_id                 BIGINT REFERENCES languages(id) ON DELETE SET NULL,

    storage_chat_id             BIGINT NOT NULL,
    storage_message_id          BIGINT NOT NULL,

    source_channel_id           BIGINT REFERENCES source_channels(id) ON DELETE SET NULL,
    source_message_id           BIGINT,

    thumbnail_storage_chat_id   BIGINT,
    thumbnail_storage_message_id BIGINT,

    duration_seconds            INT,
    size_bytes                  BIGINT,

    is_deleted                  BOOLEAN NOT NULL DEFAULT false,

    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (anime_id, episode_number, quality, language_id)
);
CREATE INDEX idx_episodes_anime ON episodes(anime_id, episode_number) WHERE is_deleted = false;
CREATE UNIQUE INDEX idx_episodes_source_dedup ON episodes(source_channel_id, source_message_id)
    WHERE source_channel_id IS NOT NULL AND source_message_id IS NOT NULL;

-- ========== Users ==========

CREATE TABLE users (
    id                  BIGSERIAL PRIMARY KEY,
    telegram_user_id    BIGINT NOT NULL UNIQUE,
    username            TEXT,
    first_name          TEXT,
    language_pref       TEXT NOT NULL DEFAULT 'fa' CHECK (language_pref IN ('fa', 'ru', 'en')),
    is_banned           BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE admins (
    id                  BIGSERIAL PRIMARY KEY,
    telegram_user_id    BIGINT NOT NULL UNIQUE,
    role                TEXT NOT NULL DEFAULT 'moderator' CHECK (role IN ('owner', 'admin', 'moderator')),
    permissions         JSONB NOT NULL DEFAULT '[]',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE favorites (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    anime_id    BIGINT NOT NULL REFERENCES animes(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, anime_id)
);

CREATE TABLE watch_history (
    id                  BIGSERIAL PRIMARY KEY,
    user_id             BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    episode_id          BIGINT NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    progress_seconds    INT NOT NULL DEFAULT 0,
    completed           BOOLEAN NOT NULL DEFAULT false,
    watched_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, episode_id)
);
CREATE INDEX idx_watch_history_user ON watch_history(user_id, watched_at DESC);

CREATE TABLE downloads (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    episode_id      BIGINT NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    downloaded_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE views (
    id          BIGSERIAL PRIMARY KEY,
    anime_id    BIGINT NOT NULL REFERENCES animes(id) ON DELETE CASCADE,
    user_id     BIGINT REFERENCES users(id) ON DELETE SET NULL,
    viewed_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_views_anime_time ON views(anime_id, viewed_at DESC);

CREATE TABLE ratings (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    anime_id    BIGINT NOT NULL REFERENCES animes(id) ON DELETE CASCADE,
    score       SMALLINT NOT NULL CHECK (score BETWEEN 1 AND 10),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, anime_id)
);

CREATE TABLE recommendations (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    anime_id    BIGINT NOT NULL REFERENCES animes(id) ON DELETE CASCADE,
    score       NUMERIC(6,3) NOT NULL DEFAULT 0,
    reason      TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, anime_id)
);

CREATE TABLE notifications (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type        TEXT NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}',
    is_read     BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ========== Channel publishing ==========

CREATE TABLE channel_posts (
    id                      BIGSERIAL PRIMARY KEY,
    anime_id                BIGINT NOT NULL REFERENCES animes(id) ON DELETE CASCADE,
    target_channel_id       BIGINT NOT NULL,
    telegram_message_id     BIGINT,
    published_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (anime_id, target_channel_id)
);

-- seed reference data
INSERT INTO languages (code, name) VALUES
    ('fa', 'فارسی'), ('ru', 'Русский'), ('en', 'English'), ('ja', '日本語')
ON CONFLICT DO NOTHING;
