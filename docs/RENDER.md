# Deploying to Render

This covers the setup discussed for `https://popular-anime-bot.onrender.com`.
Render's free tier only keeps HTTP-serving **Web Services** alive for free
(Background Workers require a paid plan), so:

- `api` runs as-is - it's already an HTTP server.
- `bot` runs in **webhook mode** instead of long-polling, so it's an HTTP
  server too and fits the free Web Service model.
- `scheduler` and `mtproto-listener` still need to run somewhere else (a
  VPS via `docker compose`, see `docs/DEPLOYMENT.md`) - they're not
  request-driven, so there's no free way to run them on Render.

## 1. Service settings on Render

If the Web Service was created with default settings pointing at this repo
with runtime **Docker**, it already builds the right thing: the root-level
`Dockerfile` builds `services/api` (the other services keep their own
Dockerfiles under `services/<name>/`, untouched by this one). Nothing to
change there unless you'd previously set a custom Dockerfile path/root
directory - in that case, point both back to the repo root.

## 2. Environment variables

In the Render dashboard, under the service's **Environment** tab, set:

| Key                | Value                                                                 |
| ------------------ | ---------------------------------------------------------------------- |
| `DATABASE_URL`     | Your Postgres connection string (from wherever you provisioned it).   |
| `INTERNAL_API_KEY` | A random secret - generate with `openssl rand -hex 32`.               |

Don't set `PORT` - Render injects it automatically and the API now reads
it (falls back to `API_PORT`/`8080` for local/docker-compose use).

## 3. Apply the database schema (one time)

`migrations/0001_init.sql` isn't applied automatically here (that
auto-apply trick only works for the official `postgres` Docker image via
`docker-entrypoint-initdb.d`, which doesn't apply to an externally hosted
Postgres). Run it once yourself, from any machine that can reach your
database:

```bash
psql "$DATABASE_URL" -f migrations/0001_init.sql
```

(If your Postgres provider has a built-in SQL console/shell, e.g. Render
Postgres's "Connect" button, pasting the file's contents there works too.)

## 4. Verify

```bash
curl https://popular-anime-bot.onrender.com/healthz
# {"status":"ok"}
```

If it 500s, check the Render service logs first - almost always either
`DATABASE_URL` is wrong/unreachable, or the migration hasn't been applied
yet.

## 5. Point the other services at it

Wherever you run `bot`, `scheduler` and `mtproto-listener` (see
`docs/DEPLOYMENT.md`), set:

```
API_BASE_URL=https://popular-anime-bot.onrender.com
INTERNAL_API_KEY=<the same value you set on the Render service>
```

## 6. Deploy the bot as a second Render Web Service

Create a **new** Web Service on Render, pointing at the same repo/branch, but:

- **Root Directory**: `services/bot`
- **Dockerfile Path**: `Dockerfile` (relative to the root directory above -
  `services/bot` already has its own, unrelated to the root one `api` uses)

Environment variables for this service:

| Key               | Value                                                                 |
| ----------------- | ---------------------------------------------------------------------- |
| `BOT_TOKEN`       | Your bot's token from @BotFather.                                     |
| `API_BASE_URL`    | `https://popular-anime-bot.onrender.com` (your `api` service's URL).  |
| `INTERNAL_API_KEY`| Not required for the bot (it only calls public routes) - fine to leave unset. |
| `WEBHOOK_URL`     | This bot service's own public Render URL, e.g. `https://popular-anime-bot-bot.onrender.com` (no trailing path - the app appends `/webhook` itself). |
| `WEBHOOK_SECRET`  | A random secret - generate with `openssl rand -hex 32`. Validated against Telegram's `X-Telegram-Bot-Api-Secret-Token` header so nobody else can POST fake updates to your webhook. |

Don't set `PORT` - Render injects it and the bot reads it automatically.

Once deployed, the bot calls Telegram's `setWebhook` itself on startup - no
manual step needed. Send `/start` to your bot in Telegram; check that
service's Render logs if nothing comes back.

## Notes / limitations

- Render's free Web Service instance type sleeps after 15 minutes of
  inactivity and wakes on the next request (~a few seconds to ~50s of
  cold-start delay, per Render's own warning). For the bot, that means the
  first message after a period of silence may take a while to get a reply -
  normal for the free tier, upgrade the instance type if it's a problem.
- This setup does **not** run `scheduler` or `mtproto-listener` on Render -
  they're not request-driven, so webhook mode doesn't apply to them.
  Running them as Render Background Workers is possible but requires a
  paid plan per worker, plus a persistent disk for the MTProto session file
  - see `docs/DEPLOYMENT.md` for the simpler single-VPS path for those two.
