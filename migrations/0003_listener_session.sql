-- Persists the MTProto userbot's session (Telethon StringSession) in the
-- database instead of a local file, so it survives redeploys/restarts on
-- PaaS providers with ephemeral disk (e.g. Render's free tier).

CREATE TABLE listener_session (
    id              SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    session_string  TEXT NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
