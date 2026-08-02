# Roadmap: MVP → Production

## Phase 0 - This commit (MVP backbone)

- [x] Postgres schema covering anime/episodes/genres/studios/users/
      favorites/history/admins/source channels/channel posts.
- [x] Core API (Go, Clean Architecture) with public + internal route split.
- [x] Telegram bot: menu, search, anime detail, paginated episodes,
      favorites, history/continue-watching, deep links from channel posts.
- [x] MTProto listener: watches configurable source channels, heuristic
      parsing, forwards media to a storage channel, best-effort RU/EN→FA
      translation.
- [x] Scheduler: daily showcase-channel posts + hourly popularity refresh.
- [x] Docker Compose for local/single-VPS deployment.

## Phase 1 - Hardening the MVP

- [ ] Admin panel (even a Telegram-based one first): approve/reject
      imported anime before it's published, edit metadata, manage
      `source_channels` without hitting the API by hand.
- [ ] Replace the bot's in-memory search-token cache with Redis (already
      provisioned in docker-compose) so the bot can run multiple replicas.
- [ ] Rate limiting + abuse protection on public API routes.
- [ ] Structured error responses + request logging/tracing (OpenTelemetry).
- [ ] Integration tests for the ingest pipeline (golden-file tests for
      `parser.py`, repository tests against a real Postgres via
      testcontainers).
- [ ] Alembic-equivalent for Go: a proper migration tool (e.g. `goose` or
      `golang-migrate`) instead of a single init SQL file.

## Phase 2 - Product depth

- [ ] Proper admin web dashboard (stats, user management, broadcast,
      logs) - a small React/Next.js app talking to the internal API
      behind auth.
- [ ] Recommendation engine v1: collaborative filtering off
      `favorites`/`watch_history`/`ratings` (start with a simple
      item-item similarity job, not full ML).
- [ ] Smarter ingest: replace the regex parser with an LLM-assisted
      extractor for title/episode/genre detection, with the regex path as
      a fast/cheap fallback.
- [ ] Multi-language UI (fa/ru/en) driven by `users.language_pref`
      (schema already supports it; bot copy needs an i18n layer).
- [ ] Notifications: new-episode alerts for favorited anime
      (`notifications` table already modeled).
- [ ] Duplicate-anime detection across multiple source channels (same
      anime scraped from two channels shouldn't create two catalog rows).

## Phase 3 - Scale to ~1M users

- [ ] Move the bot to webhooks behind a load balancer instead of long
      polling, run N stateless replicas.
- [ ] Read replicas / connection pooling (pgbouncer) for Postgres.
- [ ] Move `AnimeFilter` search from `tsvector` to a dedicated search
      engine (Meilisearch/Typesense) once catalog size and query volume
      justify it.
- [ ] CDN/edge cache for anything served over HTTP (posters via URL,
      public API GETs).
- [ ] Multi-region MTProto listener failover (a single userbot session
      is a single point of failure - plan a hot-standby session).
- [ ] Formal SLOs + on-call runbooks; this is also the point where
      moving off a single VPS to managed Postgres/Kubernetes pays off.

## Explicitly out of scope for now

These were in the original spec but need real infrastructure/product
decisions before they're worth building, so they're parked here rather
than half-implemented:

- Full ML-based recommendation/popularity models (Phase 2 starts simple).
- A generic multi-tenant "import any channel" self-serve flow - this repo
  is architected for one operator managing their own source-channel list.
