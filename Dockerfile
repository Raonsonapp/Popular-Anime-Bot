# Root Dockerfile: everything runs in ONE container / ONE Render Web
# Service, since that's the only free option available. Render defaults to
# looking for this file at the repo root.
#
# What actually runs, per entrypoint.sh:
#   - the Go binary (core API, with the Telegram bot embedded when
#     BOT_TOKEN is set - see docs/RENDER.md)
#   - the Python MTProto listener, as a background process in the same
#     container
#   - the scheduler binary, also as a background process - it's
#     cron-driven rather than request-driven, but that just means it
#     doesn't need to bind $PORT, not that it needs its own container.
#
# The individual services/<name>/Dockerfile files are untouched and still
# used by docker-compose.yml for a proper multi-container VPS deployment.

FROM golang:1.25-alpine AS build
WORKDIR /src/api
COPY services/api/go.mod services/api/go.sum ./
RUN go mod download
COPY services/api/. .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api

WORKDIR /src/scheduler
COPY services/scheduler/go.mod services/scheduler/go.sum ./
RUN go mod download
COPY services/scheduler/. .
RUN CGO_ENABLED=0 go build -o /out/scheduler ./cmd/scheduler

# python:3.11-slim (glibc, not musl) so the listener's dependencies
# (aiohttp etc.) install from prebuilt wheels without needing a compiler.
FROM python:3.11-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
RUN mkdir -p /data

COPY --from=build /out/api /usr/local/bin/api
COPY --from=build /out/scheduler /usr/local/bin/scheduler

COPY services/mtproto-listener/requirements.txt ./listener/requirements.txt
RUN pip install --no-cache-dir -r ./listener/requirements.txt
COPY services/mtproto-listener/. ./listener/

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 8080
ENTRYPOINT ["/entrypoint.sh"]
