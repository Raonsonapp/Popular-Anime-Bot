package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"popular-anime-bot/api/internal/domain"
	"popular-anime-bot/api/internal/usecase"
)

type Handler struct {
	Catalog       *usecase.CatalogService
	Ingest        *usecase.IngestService
	Users         *usecase.UserService
	Publish       *usecase.PublishService
	SourceChannel domain.SourceChannelRepository
	Logger        *slog.Logger
}

func NewHandler(catalog *usecase.CatalogService, ingest *usecase.IngestService, users *usecase.UserService, publish *usecase.PublishService, sourceChannel domain.SourceChannelRepository, logger *slog.Logger) *Handler {
	return &Handler{Catalog: catalog, Ingest: ingest, Users: users, Publish: publish, SourceChannel: sourceChannel, Logger: logger}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func queryInt(r *http.Request, key string, fallback int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func queryInt64(r *http.Request, key string, fallback int64) int64 {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}
