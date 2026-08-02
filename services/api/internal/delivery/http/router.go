package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter wires public (bot-facing) and internal (listener/scheduler/admin-facing)
// routes. Internal routes require the shared X-Internal-Key header so a stolen
// bot token alone can't be used to mutate the catalog.
func NewRouter(h *Handler, internalAPIKey string) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Public catalog + user endpoints, used by the bot.
		r.Get("/anime", h.ListAnime)
		r.Get("/anime/random", h.RandomAnime)
		r.Get("/anime/{id}", h.GetAnime)
		r.Post("/anime/{id}/view", h.RecordAnimeView)
		r.Get("/anime/{id}/episodes", h.ListEpisodes)
		r.Get("/episodes/{id}", h.GetEpisode)
		r.Get("/genres", h.ListGenres)
		r.Get("/studios", h.ListStudios)

		r.Post("/users", h.TouchUser)
		r.Post("/favorites/toggle", h.ToggleFavorite)
		r.Get("/users/{userID}/favorites", h.ListFavorites)
		r.Post("/history", h.RecordProgress)
		r.Get("/users/{userID}/history", h.ListHistory)
		r.Get("/users/{userID}/continue-watching", h.ContinueWatching)

		// Internal-only: MTProto listener, scheduler, admin panel.
		r.Group(func(r chi.Router) {
			r.Use(internalAuth(internalAPIKey))

			r.Post("/anime", h.UpsertAnime)
			r.Post("/episodes", h.UpsertEpisode)
			r.Delete("/episodes", h.DeleteEpisodeBySource)
			r.Post("/import-logs", h.CreateImportLog)

			r.Get("/source-channels", h.ListSourceChannels)
			r.Post("/source-channels", h.CreateSourceChannel)
			r.Delete("/source-channels/{id}", h.DeleteSourceChannel)
			r.Post("/source-channels/{id}/cursor", h.UpdateSourceChannelCursor)

			r.Get("/publish/pending", h.ListPendingPosts)
			r.Post("/publish/record", h.RecordPost)
			r.Post("/publish/refresh-popularity", h.RefreshPopularity)
		})
	})

	return r
}

func internalAuth(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" || r.Header.Get("X-Internal-Key") != key {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
