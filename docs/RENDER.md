# Deploying to Render

This covers the setup for `https://popular-anime-bot.onrender.com`. Render's
free tier only keeps HTTP-serving **Web Services** alive for free
(Background Workers require a paid plan), so this deployment runs the
**API and the Telegram bot together in one process**, on one free Web
Service, listening on one port:

- The bot runs in **webhook mode**: Telegram pushes updates to us over
  HTTPS instead of us long-polling, so it fits the same HTTP server the
  API already runs (mounted at `/webhook`).
- `scheduler` and `mtproto-listener` still need to run somewhere else (a
  VPS via `docker compose`, see `docs/DEPLOYMENT.md`) - they're not
  request-driven, so there's no free way to run them on Render.

(The standalone `services/bot` also still works unchanged for VPS/
docker-compose setups that prefer long-polling and a separate process -
this Render path is an alternative for the free-tier case, not a
replacement.)

## 1. Service settings on Render

If the Web Service was created with default settings pointing at this repo
with runtime **Docker**, it already builds the right thing: the root-level
`Dockerfile` builds `services/api`, which now includes the bot's logic as
an internal package. Nothing to change unless you'd previously set a
custom Dockerfile path/root directory - in that case, point both back to
the repo root.

## 2. Environment variables

In the Render dashboard, under the service's **Environment** tab, set:

| Key                | Value                                                                 |
| ------------------ | ---------------------------------------------------------------------- |
| `DATABASE_URL`     | Your Postgres connection string.                                      |
| `INTERNAL_API_KEY` | A random secret - generate with `openssl rand -hex 32`.               |
| `BOT_TOKEN`        | Your bot's token from @BotFather.                                     |
| `WEBHOOK_URL`      | This service's own public Render URL, e.g. `https://popular-anime-bot.onrender.com` (no trailing path - the app appends `/webhook` itself). |
| `WEBHOOK_SECRET`   | A random secret - generate with `openssl rand -hex 32`. Validated against Telegram's `X-Telegram-Bot-Api-Secret-Token` header so nobody else can POST fake updates to your webhook. |

Don't set `PORT` - Render injects it automatically and the app reads it.

If `BOT_TOKEN` is left unset, the service just runs the API alone (no bot) -
useful if you ever want to split them back into separate services later.

## 3. Apply the database schema (one time)

`migrations/0001_init.sql` isn't applied automatically here (that
auto-apply trick only works for the official `postgres` Docker image via
`docker-entrypoint-initdb.d`, which doesn't apply to an externally hosted
Postgres). Run it once yourself, from any machine that can reach your
database:

```bash
psql "$DATABASE_URL" -f migrations/0001_init.sql
```

(If your Postgres provider has a built-in SQL console/shell - e.g. Neon's
or Render Postgres's - pasting the file's contents there works too. If a
single paste gets truncated by the editor, split it into a few smaller
chunks at statement boundaries and run them one at a time.)

## 4. Verify

```bash
curl https://popular-anime-bot.onrender.com/healthz
# {"status":"ok"}
```

If it 500s or crash-loops, check the Render service logs first - almost
always either `DATABASE_URL` is wrong/unreachable, or the migration hasn't
been applied yet.

Then send `/start` to your bot in Telegram. The service calls Telegram's
`setWebhook` itself on startup - no manual step needed. If nothing comes
back, check the logs for `bot authorized` / `bot webhook mounted` lines to
confirm it registered.

## 5. Point the other services at it

Wherever you run `scheduler` and `mtproto-listener` (see
`docs/DEPLOYMENT.md`), set:

```
API_BASE_URL=https://popular-anime-bot.onrender.com
INTERNAL_API_KEY=<the same value you set on the Render service>
```

## Notes / limitations

- Render's free Web Service instance type sleeps after 15 minutes of
  inactivity and wakes on the next request (~a few seconds to ~50s of
  cold-start delay, per Render's own warning). For the bot, that means the
  first message after a period of silence may take a while to get a reply
  while Telegram's webhook delivery wakes the container - normal for the
  free tier, upgrade the instance type if it's a problem.
- `scheduler` and `mtproto-listener` are not request-driven, so webhook
  mode doesn't apply to them - they can't run on Render's free tier.
  Running them as Render Background Workers is possible but requires a
  paid plan per worker, plus a persistent disk for the MTProto session file
  - see `docs/DEPLOYMENT.md` for the simpler single-VPS path for those two.
