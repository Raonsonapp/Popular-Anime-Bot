# Deployment Guide

## 1. Prerequisites

- A Linux VPS (or your local machine for testing) with Docker + Docker
  Compose installed.
- A Telegram **bot** created via [@BotFather](https://t.me/BotFather) →
  `BOT_TOKEN`.
- Your **MTProto app credentials** from https://my.telegram.org
  (`api_id` / `api_hash`). This is tied to your personal Telegram account,
  not the bot.
- A private **storage channel**: create a new private Telegram channel,
  add your bot as admin (needs "Post messages" at minimum), and note its
  numeric id (forward any message from it to `@userinfobot`, or read it
  from the listener logs after first connecting).
- A public **showcase channel**: the one your audience already follows.
  Add the bot as admin there too (needs "Post messages").
- Explicit permission from the owners of any source channel you plan to
  monitor and redistribute content from. This project only automates
  something you already have the right to do manually - it doesn't grant
  you any rights over content you don't already have permission for.

## 2. Configure

```bash
git clone <this repo>
cd Popular-Anime-Bot
cp .env.example .env
```

Edit `.env`:

- `POSTGRES_PASSWORD` - pick a real password.
- `INTERNAL_API_KEY` - generate with `openssl rand -hex 32`.
- `BOT_TOKEN`, `BOT_USERNAME` - from BotFather.
- `TELEGRAM_API_ID`, `TELEGRAM_API_HASH` - from my.telegram.org.
- `STORAGE_CHANNEL_ID` - your private storage channel (looks like
  `-100xxxxxxxxxx`).
- `TARGET_CHANNEL_ID` - your public showcase channel.

Never commit `.env` - it's already in `.gitignore`.

## 3. Bring up the database and API first

```bash
docker compose up -d postgres redis api
```

The Postgres image auto-applies `migrations/0001_init.sql` on first boot
(mounted into `/docker-entrypoint-initdb.d`). Check the API is healthy:

```bash
curl http://localhost:8080/healthz
```

## 4. Log in the MTProto userbot (one-time, interactive)

```bash
docker compose run --rm mtproto-listener python login.py
```

Enter your phone number, the login code Telegram texts you, and your 2FA
password if you have one. This saves the session to Postgres (via the
API) so the long-running service can reconnect headlessly from then on -
no Docker volume needed, and it survives container recreation.

## 5. Register your source channels

The listener only watches channels registered via the API (so you can add/
remove them at runtime without redeploying). For each source channel you
have permission to monitor:

```bash
curl -X POST http://localhost:8080/api/v1/source-channels \
  -H "X-Internal-Key: $INTERNAL_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
        "telegram_channel_id": -1001234567890,
        "title": "Example RU Anime Channel",
        "source_language": "ru"
      }'
```

Your userbot account must already be a **member** of each source channel
(join it manually first) - Telethon can only see messages in chats the
account has joined.

## 6. Bring everything up

```bash
docker compose up -d
```

This starts `mtproto-listener`, `bot`, and `scheduler` alongside the
already-running `postgres`/`redis`/`api`.

Tail logs while you test:

```bash
docker compose logs -f mtproto-listener bot scheduler
```

## 7. Verify end-to-end

1. Post (or wait for) a new episode in a registered source channel.
2. Check `docker compose logs mtproto-listener` for "imported episode ...".
3. Open your bot in Telegram, send `/start`, search for the anime, open it,
   tap ▶️ تماشا (Watch), and confirm the episode delivers.
4. Wait for (or manually trigger, see below) the scheduler's daily post to
   confirm it lands in your showcase channel with working buttons.

To trigger the scheduler's cron job immediately for testing without
waiting for its schedule, temporarily set `POST_CRON_SCHEDULE=* * * * *`
(every minute) in `.env`, `docker compose up -d scheduler`, confirm it
works, then set it back.

## 8. Operations

- **Add an admin** (for future admin-panel work): insert directly for now
  -
  ```sql
  INSERT INTO admins (telegram_user_id, role) VALUES (<your_telegram_id>, 'owner');
  ```
- **Backups**: `pg_dump` the `postgres_data` volume on a schedule; it's the
  only stateful data store besides the Telethon session.
- **Updating**: `git pull && docker compose build && docker compose up -d`.
  Migrations are currently a single file - see `docs/ROADMAP.md` Phase 1
  for moving to a real migration tool before the schema needs to evolve
  further.
- **Scaling the bot**: it's stateless except for the small in-memory
  search-token cache (see `docs/ARCHITECTURE.md`); Phase 1 moves that to
  Redis so you can safely run multiple bot replicas behind webhooks.
