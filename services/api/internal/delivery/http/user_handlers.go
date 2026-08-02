package http

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type touchUserRequest struct {
	TelegramUserID int64  `json:"telegram_user_id"`
	Username       string `json:"username"`
	FirstName      string `json:"first_name"`
	Language       string `json:"language"`
}

func (h *Handler) TouchUser(w http.ResponseWriter, r *http.Request) {
	var req touchUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	u, err := h.Users.Touch(r.Context(), req.TelegramUserID, req.Username, req.FirstName, req.Language)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to upsert user")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

type toggleFavoriteRequest struct {
	UserID  int64 `json:"user_id"`
	AnimeID int64 `json:"anime_id"`
}

func (h *Handler) ToggleFavorite(w http.ResponseWriter, r *http.Request) {
	var req toggleFavoriteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	added, err := h.Users.ToggleFavorite(r.Context(), req.UserID, req.AnimeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to toggle favorite")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"added": added})
}

func (h *Handler) ListFavorites(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", 20)
	items, total, err := h.Users.ListFavorites(r.Context(), userID, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list favorites")
		return
	}
	writeJSON(w, http.StatusOK, listResponse{Items: items, Total: total, Page: page})
}

type recordProgressRequest struct {
	UserID          int64 `json:"user_id"`
	EpisodeID       int64 `json:"episode_id"`
	ProgressSeconds int   `json:"progress_seconds"`
	Completed       bool  `json:"completed"`
}

func (h *Handler) RecordProgress(w http.ResponseWriter, r *http.Request) {
	var req recordProgressRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := h.Users.RecordProgress(r.Context(), req.UserID, req.EpisodeID, req.ProgressSeconds, req.Completed); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record progress")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) ListHistory(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", 20)
	items, total, err := h.Users.ListHistory(r.Context(), userID, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list history")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": items, "total": total, "page": page})
}

func (h *Handler) ContinueWatching(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	items, err := h.Users.ContinueWatching(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list continue watching")
		return
	}
	writeJSON(w, http.StatusOK, items)
}
