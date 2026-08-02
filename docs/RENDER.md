# Deploying to Render

This covers the setup for `https://popular-anime-bot.onrender.com`. Render's
free tier only keeps one thing alive for free per service: an HTTP-serving
**Web Service**. So this deployment runs **everything that can share one
container in that single service**: the API, the Telegram bot, and the
MTProto listener all run together, as separate processes inside one
container (`entrypoint.sh` starts them). Only `scheduler` is left out - see
the note at the bottom.

- The bot runs in **webhook mode**: Telegram pushes updates to us over
  HTTPS instead of us long-polling, so it fits the same HTTP server the
  API already runs (mounted at `/webhook`).
- The MTProto listener runs as a background process in the same
  container, talking to the API over `localhost` (no public URL needed).
- `scheduler` still needs to run somewhere else (a VPS via
  `docker compose`, see `docs/DEPLOYMENT.md`) - it's cron-driven, not
  request-driven, so there's no way to fit it into this same container.

(The standalone `services/bot` and `services/mtproto-listener` also still
work unchanged for a proper multi-container VPS deployment via
`docker-compose.yml` - this single-service Render path is a free-tier
alternative, not a replacement.)

## 1. Service settings on Render

If the Web Service was created with default settings pointing at this repo
with runtime **Docker**, it already builds the right thing: the root-level
`Dockerfile` builds the Go binary (API + bot) and bundles the Python
listener alongside it, then `entrypoint.sh` starts both. Nothing to change
unless you'd previously set a custom Dockerfile path/root directory - in
that case, point both back to the repo root.

## 2. Environment variables

In the Render dashboard, under the service's **Environment** tab, set:

| Key                  | Value                                                                 |
| -------------------- | ---------------------------------------------------------------------- |
| `DATABASE_URL`       | Your Postgres connection string.                                      |
| `INTERNAL_API_KEY`   | A random secret - generate with `openssl rand -hex 32`.               |
| `BOT_TOKEN`          | Your bot's token from @BotFather.                                     |
| `WEBHOOK_URL`        | This service's own public Render URL, e.g. `https://popular-anime-bot.onrender.com` (no trailing path - the app appends `/webhook` itself). |
| `WEBHOOK_SECRET`     | A random secret - generate with `openssl rand -hex 32`.               |
| `TELEGRAM_API_ID`    | From https://my.telegram.org - enables the listener. Leave unset to run without it. |
| `TELEGRAM_API_HASH`  | From https://my.telegram.org.                                         |
| `STORAGE_CHANNEL_ID` | Numeric id of your private "storage" channel, looks like `-100xxxxxxxxxx` (forward any message from it to @userinfobot to get this). Your bot must be **admin** there. |

Don't set `PORT` or `API_BASE_URL` - Render injects `PORT` automatically,
and `entrypoint.sh` points the listener at the right local address itself.

Leaving `TELEGRAM_API_ID`/`TELEGRAM_API_HASH` unset just skips starting the
listener (the API and bot still run normally) - useful if you want to add
it later.

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

## 4. Verify the API and bot

```bash
curl https://popular-anime-bot.onrender.com/healthz
# {"status":"ok"}
```

Then send `/start` to your bot in Telegram - it should reply immediately.
If not, check the Render logs for `bot authorized` / `bot webhook mounted`.

## 5. One-time listener login (no Shell needed)

Render's **Shell** tab requires a paid plan, so logging in from a
terminal isn't an option on the free tier. Instead, the service exposes a
tiny web-based login flow at `/telegram-login/...`, completed entirely by
opening links in your phone's browser - no app or terminal needed.

Add the `CHANNELS` list you want the bot to pull anime from **before**
logging in: edit `services/mtproto-listener/register_channels.py` in the
repo (list of `("username", "language")` tuples), commit, and push - the
login flow registers them automatically right after you sign in.

Then, in a browser:

1. Open `https://popular-anime-bot.onrender.com/telegram-login/send-code?key=<INTERNAL_API_KEY>&phone=992XXXXXXXXX`
   (replace `<INTERNAL_API_KEY>` with your real value, and `phone=` with
   your number, digits only, country code first, no `+` or spaces).
2. Telegram sends you a login code in the app. Open:
   `https://popular-anime-bot.onrender.com/telegram-login/sign-in?key=<INTERNAL_API_KEY>&code=12345`
   (replace `code=` with the real code).
3. If your account has a 2FA password, step 2's page will tell you to
   instead open a `sign-in?...&password=YOUR_PASSWORD` link - do that.
4. On success, the page lists which source channels got registered. The
   real listener starts automatically within ~10 seconds - no restart
   needed.

⚠️ **Disk persistence warning**: Render's free tier has no persistent
disk, so the session file can be wiped on a future deploy/restart,
requiring you to repeat this login flow. If that becomes annoying, either
add a paid persistent disk, or run the listener on a VPS instead
(`docs/DEPLOYMENT.md`) where the session survives normally.

## 6. Adding more source channels later

Edit the `CHANNELS` list in `register_channels.py`, commit, and push -
this redeploys the service (session file is preserved across a redeploy
that doesn't wipe disk, only across a fresh container). Since the login
flow already ran once, `main.py` is running normally; to pick up new
channels without a full re-login, use the manual single-channel
registration instead (works any time, no login flow needed):

```bash
curl -X POST https://popular-anime-bot.onrender.com/api/v1/source-channels \
  -H "X-Internal-Key: <INTERNAL_API_KEY>" \
  -H "Content-Type: application/json" \
  -d '{
        "telegram_channel_id": -100xxxxxxxxxx,
        "title": "Channel title",
        "source_language": "fa"
      }'
```

(You still need the channel's numeric id and your account to already be
a member/have joined it - the automatic username-resolving + join only
happens during the `/telegram-login/sign-in` flow above.)

## 7. Importing a channel's existing history (backfill)

By default, only messages posted **after** a channel is registered get
imported - the login flow above doesn't touch anything already in the
channel. To also pull in what's already there, open (once the listener is
running normally, a few seconds after login):

```
https://popular-anime-bot.onrender.com/telegram-login/backfill?key=<INTERNAL_API_KEY>&limit=300
```

`limit` is the max number of messages scanned per registered channel
(default 300 if omitted) - raise it, or set `limit=all` to walk a
channel's entire history with no cap (can take a long time for
channels with thousands of posts, but runs in the background regardless).
The page responds immediately either way, and progress/results show up in
the Render **Logs** tab (look for `backfilling ...` / `backfill
finished: ...` lines). Safe to re-run - already-imported episodes are
skipped via the same dedup logic used for live messages.

## 8. Point the scheduler at it (if you run one)

Wherever you run `scheduler` (see `docs/DEPLOYMENT.md`), set:

```
API_BASE_URL=https://popular-anime-bot.onrender.com
INTERNAL_API_KEY=<the same value you set on the Render service>
```

## Notes / limitations

- Render's free Web Service instance type sleeps after 15 minutes of
  inactivity and wakes on the next request (~a few seconds to ~50s of
  cold-start delay, per Render's own warning). While asleep, the listener
  isn't watching Telegram either - new posts in your source channels will
  only get picked up after something wakes the service back up (e.g. a
  user messaging the bot). Upgrade the instance type if that gap matters.
- `scheduler` isn't request-driven, so it can't share this container - it
  still needs a VPS (`docs/DEPLOYMENT.md`) or a paid Render Background
  Worker.
