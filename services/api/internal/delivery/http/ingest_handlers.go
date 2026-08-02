package http

import (
	"net/http"

	"popular-anime-bot/api/internal/domain"
	"popular-anime-bot/api/internal/usecase"
)

type upsertAnimeRequest struct {
	SourceChannelID        int64    `json:"source_channel_id"`
	TitleOriginal          string   `json:"title_original"`
	TitlePersian           string   `json:"title_persian"`
	TitleEnglish           string   `json:"title_english"`
	TitleJapanese          string   `json:"title_japanese"`
	SynopsisPersian        string   `json:"synopsis_persian"`
	SynopsisOriginal       string   `json:"synopsis_original"`
	Kind                   string   `json:"kind"`
	Status                 string   `json:"status"`
	Year                   int      `json:"year"`
	StudioName             string   `json:"studio_name"`
	GenreNamesEnglish      []string `json:"genre_names_english"`
	GenreNamesPersian      []string `json:"genre_names_persian"`
	PosterStorageChatID    int64    `json:"poster_storage_chat_id"`
	PosterStorageMessageID int64    `json:"poster_storage_message_id"`
	DurationMinutes        int      `json:"duration_minutes"`
	AutoPublish            bool     `json:"auto_publish"`
}

func (h *Handler) UpsertAnime(w http.ResponseWriter, r *http.Request) {
	var req upsertAnimeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.TitleOriginal == "" || req.SourceChannelID == 0 {
		writeError(w, http.StatusBadRequest, "source_channel_id and title_original are required")
		return
	}

	a, err := h.Ingest.UpsertAnime(r.Context(), usecase.UpsertAnimeInput{
		SourceChannelID:        req.SourceChannelID,
		TitleOriginal:          req.TitleOriginal,
		TitlePersian:           req.TitlePersian,
		TitleEnglish:           req.TitleEnglish,
		TitleJapanese:          req.TitleJapanese,
		SynopsisPersian:        req.SynopsisPersian,
		SynopsisOriginal:       req.SynopsisOriginal,
		Kind:                   domain.AnimeKind(req.Kind),
		Status:                 domain.AnimeStatus(req.Status),
		Year:                   req.Year,
		StudioName:             req.StudioName,
		GenreNamesEnglish:      req.GenreNamesEnglish,
		GenreNamesPersian:      req.GenreNamesPersian,
		PosterStorageChatID:    req.PosterStorageChatID,
		PosterStorageMessageID: req.PosterStorageMessageID,
		DurationMinutes:        req.DurationMinutes,
		AutoPublish:            req.AutoPublish,
	})
	if err != nil {
		h.Logger.Error("upsert anime", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to upsert anime")
		return
	}
	writeJSON(w, http.StatusOK, a)
}

type upsertEpisodeRequest struct {
	AnimeID          int64  `json:"anime_id"`
	EpisodeNumber    int    `json:"episode_number"`
	Title            string `json:"title"`
	Quality          string `json:"quality"`
	LanguageID       int64  `json:"language_id"`
	StorageChatID    int64  `json:"storage_chat_id"`
	StorageMessageID int64  `json:"storage_message_id"`
	SourceChannelID  int64  `json:"source_channel_id"`
	SourceMessageID  int64  `json:"source_message_id"`
	DurationSeconds  int    `json:"duration_seconds"`
	SizeBytes        int64  `json:"size_bytes"`
}

func (h *Handler) UpsertEpisode(w http.ResponseWriter, r *http.Request) {
	var req upsertEpisodeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.AnimeID == 0 || req.StorageChatID == 0 || req.StorageMessageID == 0 {
		writeError(w, http.StatusBadRequest, "anime_id, storage_chat_id and storage_message_id are required")
		return
	}

	e, err := h.Ingest.UpsertEpisode(r.Context(), usecase.UpsertEpisodeInput{
		AnimeID:          req.AnimeID,
		EpisodeNumber:    req.EpisodeNumber,
		Title:            req.Title,
		Quality:          req.Quality,
		LanguageID:       req.LanguageID,
		StorageChatID:    req.StorageChatID,
		StorageMessageID: req.StorageMessageID,
		SourceChannelID:  req.SourceChannelID,
		SourceMessageID:  req.SourceMessageID,
		DurationSeconds:  req.DurationSeconds,
		SizeBytes:        req.SizeBytes,
	})
	if err != nil {
		h.Logger.Error("upsert episode", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to upsert episode")
		return
	}
	writeJSON(w, http.StatusOK, e)
}

type deleteEpisodeRequest struct {
	SourceChannelID int64 `json:"source_channel_id"`
	SourceMessageID int64 `json:"source_message_id"`
}

func (h *Handler) DeleteEpisodeBySource(w http.ResponseWriter, r *http.Request) {
	var req deleteEpisodeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := h.Ingest.MarkEpisodeDeleted(r.Context(), req.SourceChannelID, req.SourceMessageID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete episode")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

type importLogRequest struct {
	SourceChannelID   *int64 `json:"source_channel_id"`
	TelegramMessageID int64  `json:"telegram_message_id"`
	Action            string `json:"action"`
	AnimeID           *int64 `json:"anime_id"`
	EpisodeID         *int64 `json:"episode_id"`
	Detail            string `json:"detail"`
}

func (h *Handler) CreateImportLog(w http.ResponseWriter, r *http.Request) {
	var req importLogRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	h.Ingest.LogImport(r.Context(), req.SourceChannelID, req.TelegramMessageID, req.Action, req.AnimeID, req.EpisodeID, req.Detail, nil)
	writeJSON(w, http.StatusNoContent, nil)
}
