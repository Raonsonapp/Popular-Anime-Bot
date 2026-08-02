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

## 5. One-time listener login

Open this service's **Shell** tab in Render and run:

```bash
cd listener
python login.py
```

Enter your phone number, the code Telegram texts you, and your 2FA
password if set. This writes the session file so the listener can
reconnect without further interaction from then on.

⚠️ **Disk persistence warning**: Render's free tier has no persistent
disk, so this session file can be wiped on the next deploy/restart,
requiring you to log in again the same way. If that becomes annoying,
either add a paid persistent disk, or run the listener on a VPS instead
(`docs/DEPLOYMENT.md`) where the session survives normally.

## 6. Register source channels

Still in the Shell, in the `listener` directory: edit the `CHANNELS` list
at the top of `register_channels.py` with the `@usernames` you want the
bot to pull anime from, then run:

```bash
python register_channels.py
```

It resolves each username to its numeric id, joins it with your account
(needed so Telegram actually pushes new-message events to the listener),
and registers it with the API - skipping anything already registered, so
it's safe to re-run after adding more usernames.

(Alternative: to manually register a channel by numeric id instead -
e.g. the single-channel self-curated setup from `docs/DEPLOYMENT.md` where
the source *is* your storage channel - `curl -X POST .../api/v1/source-channels`
directly; see that doc for the exact command.)

## 7. Point the scheduler at it (if you run one)

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
