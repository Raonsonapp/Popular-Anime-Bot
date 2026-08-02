# Deploying the core API to Render

This covers the setup discussed for `https://popular-anime-bot.onrender.com`:
Render hosts **only the `api` service** (it's the one HTTP-serving piece,
so it's the only one that fits Render's Web Service model cleanly). The
`bot`, `scheduler` and `mtproto-listener` services still need to run
somewhere else (a VPS via `docker compose`, see `docs/DEPLOYMENT.md`) and
just point at this API's public URL.

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

## Notes / limitations

- Render's free Web Service instance type sleeps after 15 minutes of
  inactivity and wakes on the next request (~a few seconds of cold-start
  delay). Fine for the API since the bot/scheduler call it on demand; if
  that cold start becomes annoying, upgrade the instance type.
- This setup does **not** run `bot`, `scheduler`, or `mtproto-listener` on
  Render. Running those as Render Background Workers is possible but
  requires a paid plan per worker, plus a persistent disk for the MTProto
  session file - see `docs/DEPLOYMENT.md` for the simpler single-VPS path.
