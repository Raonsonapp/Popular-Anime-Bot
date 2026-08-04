package http

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"popular-anime-bot/api/internal/domain"
)

type listResponse struct {
	Items []domain.Anime `json:"items"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
}

func (h *Handler) ListAnime(w http.ResponseWriter, r *http.Request) {
	f := domain.AnimeFilter{
		Search:   r.URL.Query().Get("search"),
		Status:   domain.AnimeStatus(r.URL.Query().Get("status")),
		SortBy:   r.URL.Query().Get("sort"),
		GenreID:  queryInt64(r, "genre_id", 0),
		StudioID: queryInt64(r, "studio_id", 0),
		Year:     queryInt(r, "year", 0),
		Page:     queryInt(r, "page", 1),
		PageSize: queryInt(r, "page_size", 20),
	}

	items, total, err := h.Catalog.ListAnime(r.Context(), f)
	if err != nil {
		h.Logger.Error("list anime", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list anime")
		return
	}
	writeJSON(w, http.StatusOK, listResponse{Items: items, Total: total, Page: f.Page})
}

func (h *Handler) RandomAnime(w http.ResponseWriter, r *http.Request) {
	a, err := h.Catalog.RandomAnime(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch random anime")
		return
	}
	if a == nil {
		writeError(w, http.StatusNotFound, "no anime available")
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *Handler) GetAnime(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid anime id")
		return
	}
	a, err := h.Catalog.GetAnime(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch anime")
		return
	}
	if a == nil {
		writeError(w, http.StatusNotFound, "anime not found")
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *Handler) RecordAnimeView(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid anime id")
		return
	}
	if err := h.Catalog.RecordView(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record view")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) ListEpisodes(w http.ResponseWriter, r *http.Request) {
	animeID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid anime id")
		return
	}
	seasonID := queryInt64(r, "season_id", 0)
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", 20)

	episodes, total, err := h.Catalog.ListEpisodes(r.Context(), animeID, seasonID, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list episodes")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": episodes, "total": total, "page": page,
	})
}

func (h *Handler) ListSeasons(w http.ResponseWriter, r *http.Request) {
	animeID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid anime id")
		return
	}
	seasons, err := h.Catalog.ListSeasons(r.Context(), animeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list seasons")
		return
	}
	writeJSON(w, http.StatusOK, seasons)
}

func (h *Handler) GetEpisode(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid episode id")
		return
	}
	e, err := h.Catalog.GetEpisode(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch episode")
		return
	}
	if e == nil {
		writeError(w, http.StatusNotFound, "episode not found")
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *Handler) ListGenres(w http.ResponseWriter, r *http.Request) {
	genres, err := h.Catalog.ListGenres(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list genres")
		return
	}
	writeJSON(w, http.StatusOK, genres)
}

func (h *Handler) ListStudios(w http.ResponseWriter, r *http.Request) {
	studios, err := h.Catalog.ListStudios(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list studios")
		return
	}
	writeJSON(w, http.StatusOK, studios)
}
