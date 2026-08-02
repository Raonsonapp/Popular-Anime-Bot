# Architecture

## Overview

Two Telegram-facing surfaces, four backend services:

```
                    ┌─────────────────────┐
 Source channels ──▶│  mtproto-listener    │  (Python / Telethon userbot)
 (RU/EN anime       │  watches channels,   │
  channels you have │  parses posts,       │
  permission to     │  forwards media to   │
  monitor)          │  the storage channel │
                    └──────────┬───────────┘
                               │ HTTP (X-Internal-Key)
                               ▼
                    ┌─────────────────────┐
                    │        api          │  (Go / Clean Architecture)
                    │  Postgres-backed     │◀──────────────┐
                    │  catalog + users     │                │
                    └──────────┬───────────┘                │
                     ▲         │ HTTP                        │ HTTP (X-Internal-Key)
          public REST│         ▼                             │
                    ┌─────────────────────┐        ┌─────────────────────┐
                    │        bot           │        │      scheduler       │
                    │  Telegram Bot API,   │        │  cron: daily channel  │
                    │  long polling        │        │  posts + popularity   │
                    └──────────┬───────────┘        └──────────┬───────────┘
                               │ sendMessage/copyMessage        │ sendPhoto+buttons
                               ▼                                ▼
                         End users (DM)                 Showcase channel (public)
```

A private **storage channel** (the bot is admin there) sits in the middle:
the userbot forwards matched posts (episodes, posters) into it, and the bot
delivers content to end users via Bot API's `copyMessage`, which never needs
a raw Telethon file reference and never carries a "Forwarded from" tag.

## Why a storage channel instead of raw file_ids

Telethon (MTProto) and the Bot API are different protocol surfaces with
different, non-interchangeable file identifiers. A userbot session cannot
hand a bot a `file_id` that works with `sendVideo`. The standard, reliable
pattern (used by essentially every Telegram "media library" bot) is:

1. Userbot forwards the source message into a private channel the bot is a
   member/admin of.
2. Because the bot is a member of that chat, `copyMessage(target, storage_channel_id, message_id)`
   works without ever touching a file_id directly.

This also means re-uploads never happen - everything stays server-side on
Telegram's infrastructure.

## Services

### `services/api` (Go, Clean Architecture)

- `internal/domain` - entities + repository interfaces (ports). No framework
  imports here.
- `internal/repository/postgres` - `sqlx` + `pgx` implementations of those
  ports.
- `internal/usecase` - business logic: `CatalogService` (read-facing),
  `IngestService` (idempotent upserts from the listener), `UserService`
  (favorites/history), `PublishService` (scheduler feed).
- `internal/delivery/http` - `chi` router + handlers. Routes are split into
  public (bot-facing: search, anime detail, favorites, history) and
  internal (listener/scheduler/admin: mutations, source-channel management),
  gated by a shared `X-Internal-Key` header so a leaked bot token alone
  can't mutate the catalog.

### `services/bot` (Go)

Long-polls the Bot API (`go-telegram-bot-api`), calls the core API over
HTTP, and never touches Postgres directly - this keeps the bot stateless
and horizontally scalable. Delivers episodes via `copyMessage` from the
storage channel.

### `services/mtproto-listener` (Python, Telethon)

Runs as a single userbot session. Polls the API for the active
`source_channels` list every `CHANNEL_REFRESH_SECONDS` (so admins can add/
remove channels without restarting the container), listens for
`NewMessage` / `MessageEdited` / `MessageDeleted`, and does best-effort
regex parsing (see `parser.py`) plus optional machine translation
(`translator.py`, via `deep-translator`) before pushing structured data to
the API.

### `services/scheduler` (Go)

Two cron jobs (`robfig/cron`): one posts newly-approved anime to the public
showcase channel with a deep link back into the bot
(`t.me/<bot>?start=anime_<id>`), the other periodically recomputes the
popularity score used for the "Trending" feed.

## Database

See `migrations/0001_init.sql`. Highlights:

- `animes.search_vector` (tsvector, GIN-indexed) + `pg_trgm` on titles for
  fast full-text + fuzzy search across Persian/English/Japanese titles.
- `episodes` is deduplicated per `(source_channel_id, source_message_id)`
  so re-processing a message (e.g. after an edit) never creates duplicates.
- `import_logs` gives a full audit trail of every message the listener
  touched, for debugging bad parses.
- Soft deletes (`is_deleted`) everywhere content can disappear, so a
  deleted source post doesn't cascade-delete user favorites/history.

## Security notes

- The MTProto `api_id`/`api_hash` and the userbot session file are the most
  sensitive secrets in this system - anyone with the session file can act
  as your Telegram account. Never commit them; `.gitignore` already
  excludes `.env` and `*.session`.
- Internal routes require `X-Internal-Key`; rotate it if it ever leaks.
- Only import from channels you actually have permission to redistribute
  content from - this is a policy/legal concern the code can't enforce for
  you.
