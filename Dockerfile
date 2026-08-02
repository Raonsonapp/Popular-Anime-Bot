# Root Dockerfile for deploying the core API as a single Render (or any
# generic Docker-PaaS) web service. Render defaults to looking for this
# file at the repo root, so it needs to exist here even though the other
# services keep their own Dockerfiles under services/<name>/.
#
# The bot, scheduler and mtproto-listener services are NOT started by this
# image - they're separate long-running processes best run elsewhere (a
# VPS via docker-compose, or as additional Render services later). See
# docs/RENDER.md.
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY services/api/go.mod services/api/go.sum ./
RUN go mod download
COPY services/api/. .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/api /usr/local/bin/api
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
