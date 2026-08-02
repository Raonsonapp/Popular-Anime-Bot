# Popular Anime Bot

A Telegram-native anime platform: a public **showcase channel** for
discovery and a **bot** that serves as the streaming library, backed by an
MTProto userbot that automatically imports and translates content from
source channels you have permission to monitor.

See `docs/ARCHITECTURE.md` for how the pieces fit together, `docs/ROADMAP.md`
for what's built vs. planned, and `docs/DEPLOYMENT.md` for a step-by-step
setup guide.

## Services

| Service                       | Language | Role                                                             |
| ------------------------------ | -------- | ----------------------------------------------------------------- |
| `services/api`                 | Go       | Core catalog/user API, Postgres-backed, Clean Architecture       |
| `services/bot`                 | Go       | Telegram Bot API service (menu, search, watch, favorites)        |
| `services/mtproto-listener`    | Python   | Telethon userbot: watches source channels, imports + translates  |
| `services/scheduler`           | Go       | Daily showcase-channel posts + hourly popularity refresh (cron)  |

## Quick start

```bash
cp .env.example .env   # fill in your real credentials, see docs/DEPLOYMENT.md
docker compose up -d postgres redis api
docker compose run --rm mtproto-listener python login.py   # one-time interactive login
docker compose up -d
```

Full walkthrough, including how to register source channels and set up the
storage/showcase channels: **`docs/DEPLOYMENT.md`**.

## Local development (without Docker)

```bash
# API
cd services/api && go run ./cmd/api

# Bot
cd services/bot && go run ./cmd/bot

# Scheduler
cd services/scheduler && go run ./cmd/scheduler

# Listener
cd services/mtproto-listener && pip install -r requirements.txt && python main.py
```

Each service reads its config from environment variables (see
`.env.example` for the full list).

## A note on content

This project automates something you need to already have the right to
do: monitor and redistribute posts from specific Telegram channels. It
doesn't grant any rights over content - get explicit permission from the
source channels' owners before pointing the listener at them.
