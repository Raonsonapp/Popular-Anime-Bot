package http

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"popular-anime-bot/api/internal/domain"
)

func (h *Handler) ListSourceChannels(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active_only") == "true"
	channels, err := h.SourceChannel.List(r.Context(), activeOnly)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list source channels")
		return
	}
	writeJSON(w, http.StatusOK, channels)
}

type createSourceChannelRequest struct {
	TelegramChannelID int64  `json:"telegram_channel_id"`
	Username          string `json:"username"`
	Title             string `json:"title"`
	SourceLanguage    string `json:"source_language"`
}

func (h *Handler) CreateSourceChannel(w http.ResponseWriter, r *http.Request) {
	var req createSourceChannelRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.TelegramChannelID == 0 || req.Title == "" {
		writeError(w, http.StatusBadRequest, "telegram_channel_id and title are required")
		return
	}
	c := &domain.SourceChannel{
		TelegramChannelID: req.TelegramChannelID,
		Title:             req.Title,
		SourceLanguage:    req.SourceLanguage,
		IsActive:          true,
	}
	if req.Username != "" {
		c.Username = &req.Username
	}
	if c.SourceLanguage == "" {
		c.SourceLanguage = "ru"
	}
	id, err := h.SourceChannel.Create(r.Context(), c)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create source channel")
		return
	}
	c.ID = id
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) DeleteSourceChannel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.SourceChannel.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete source channel")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

type updateSyncCursorRequest struct {
	LastSyncedMessageID int64 `json:"last_synced_message_id"`
}

func (h *Handler) UpdateSourceChannelCursor(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req updateSyncCursorRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := h.SourceChannel.UpdateLastSyncedMessageID(r.Context(), id, req.LastSyncedMessageID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update cursor")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) ListPendingPosts(w http.ResponseWriter, r *http.Request) {
	targetChannelID := queryInt64(r, "target_channel_id", 0)
	if targetChannelID == 0 {
		writeError(w, http.StatusBadRequest, "target_channel_id is required")
		return
	}
	limit := queryInt(r, "limit", 5)
	animes, err := h.Publish.PendingAnime(r.Context(), targetChannelID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list pending posts")
		return
	}
	writeJSON(w, http.StatusOK, animes)
}

type recordPostRequest struct {
	AnimeID           int64 `json:"anime_id"`
	TargetChannelID   int64 `json:"target_channel_id"`
	TelegramMessageID int64 `json:"telegram_message_id"`
}

func (h *Handler) RecordPost(w http.ResponseWriter, r *http.Request) {
	var req recordPostRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := h.Publish.RecordPost(r.Context(), req.AnimeID, req.TargetChannelID, req.TelegramMessageID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record post")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) RefreshPopularity(w http.ResponseWriter, r *http.Request) {
	if err := h.Publish.RefreshPopularity(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to refresh popularity")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) GetListenerSession(w http.ResponseWriter, r *http.Request) {
	sessionString, err := h.ListenerSession.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load listener session")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"session_string": sessionString})
}

type saveListenerSessionRequest struct {
	SessionString string `json:"session_string"`
}

func (h *Handler) SaveListenerSession(w http.ResponseWriter, r *http.Request) {
	var req saveListenerSessionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.SessionString == "" {
		writeError(w, http.StatusBadRequest, "session_string is required")
		return
	}
	if err := h.ListenerSession.Save(r.Context(), req.SessionString); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save listener session")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}
