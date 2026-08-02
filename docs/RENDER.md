# Deploying to Render

This covers the setup for `https://popular-anime-bot.onrender.com`. Render's
free tier only keeps HTTP-serving **Web Services** alive for free
(Background Workers require a paid plan), so this deployment runs the
**API and the Telegram bot together in one process**, on one free Web
Service, listening on one port:

- The bot runs in **webhook mode**: Telegram pushes updates to us over
  HTTPS instead of us long-polling, so it fits the same HTTP server the
  API already runs (mounted at `/webhook`).
- `mtproto-listener` can also run as its own free Render Web Service - it
  binds a tiny `/healthz` endpoint to `$PORT` just to satisfy Render, while
  the actual work (watching Telegram) happens in the background. See
  section 6 below.
- `scheduler` still needs to run somewhere else (a VPS via
  `docker compose`, see `docs/DEPLOYMENT.md`) - it's cron-driven, not
  request-driven, so there's no free way to run it on Render.

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

Wherever you run `scheduler` (see `docs/DEPLOYMENT.md`), set:

```
API_BASE_URL=https://popular-anime-bot.onrender.com
INTERNAL_API_KEY=<the same value you set on the Render service>
```

## 6. Deploy the MTProto listener as a third Render Web Service

The simplest setup if you don't have a channel you already have scraping
permission for: create your own **private channel**, post anime videos
into it yourself (caption format: title on the first line, then optional
`Episode N`, quality tag, `Genre: ...` line - see `parser.py`), add your
bot as **admin** there, and register that same channel as both the
source and the storage channel. The listener detects this and skips the
forward-to-storage step entirely - it just indexes what you post, in
place.

Create a **new** Web Service on Render, same repo/branch:

- **Root Directory**: `services/mtproto-listener`
- **Dockerfile Path**: `Dockerfile`

Environment variables:

| Key                  | Value                                                                 |
| -------------------- | ---------------------------------------------------------------------- |
| `TELEGRAM_API_ID`    | From https://my.telegram.org.                                         |
| `TELEGRAM_API_HASH`  | From https://my.telegram.org.                                         |
| `SESSION_NAME`       | `/data/userbot.session` (see the disk warning below).                 |
| `STORAGE_CHANNEL_ID` | Your channel's numeric id, looks like `-100xxxxxxxxxx` (forward any message from it to @userinfobot to find it). |
| `API_BASE_URL`       | `https://popular-anime-bot.onrender.com`                              |
| `INTERNAL_API_KEY`   | The same value set on the main service.                               |

Don't set `PORT` manually - Render injects it, and the listener starts a
health endpoint on it automatically.

**One-time login**: Render's free Web Services include a **Shell** tab.
After the first deploy, open it and run:

```bash
python login.py
```

Enter your phone number, the code Telegram texts you, and your 2FA
password if set. This writes the session file so the main process can
reconnect without further interaction.

**Register the channel** as a source (do this once, from any machine):

```bash
curl -X POST https://popular-anime-bot.onrender.com/api/v1/source-channels \
  -H "X-Internal-Key: <INTERNAL_API_KEY>" \
  -H "Content-Type: application/json" \
  -d '{
        "telegram_channel_id": -100xxxxxxxxxx,
        "title": "My anime channel",
        "source_language": "fa"
      }'
```

Use `"source_language": "fa"` (or `"tg"`) if you're posting content
already in Persian/Tajik yourself - that skips the RU/EN→FA machine
translation step entirely, which is both faster and more accurate for
content you wrote yourself.

⚠️ **Disk persistence warning**: Render's free tier has no persistent
disk. The session file written by `login.py` can be wiped on the next
deploy or restart, requiring you to log in again via the Shell. If this
becomes annoying, either add a paid persistent disk to this service, or
run the listener on a VPS instead (see `docs/DEPLOYMENT.md`) where the
session survives normally.

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
