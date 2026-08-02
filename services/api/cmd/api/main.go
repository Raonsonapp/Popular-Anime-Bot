package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"popular-anime-bot/api/internal/config"
	httpDelivery "popular-anime-bot/api/internal/delivery/http"
	"popular-anime-bot/api/internal/repository/postgres"
	"popular-anime-bot/api/internal/usecase"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect to postgres", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	animeRepo := postgres.NewAnimeRepo(db)
	episodeRepo := postgres.NewEpisodeRepo(db)
	genreRepo := postgres.NewGenreRepo(db)
	studioRepo := postgres.NewStudioRepo(db)
	sourceChannelRepo := postgres.NewSourceChannelRepo(db)
	userRepo := postgres.NewUserRepo(db)
	favoriteRepo := postgres.NewFavoriteRepo(db)
	historyRepo := postgres.NewHistoryRepo(db)
	channelPostRepo := postgres.NewChannelPostRepo(db)
	importLogRepo := postgres.NewImportLogRepo(db)

	catalog := usecase.NewCatalogService(animeRepo, episodeRepo, genreRepo, studioRepo)
	ingest := usecase.NewIngestService(animeRepo, episodeRepo, genreRepo, studioRepo, sourceChannelRepo, importLogRepo)
	users := usecase.NewUserService(userRepo, favoriteRepo, historyRepo)
	publish := usecase.NewPublishService(animeRepo, channelPostRepo)

	handler := httpDelivery.NewHandler(catalog, ingest, users, publish, sourceChannelRepo, logger)
	router := httpDelivery.NewRouter(handler, cfg.InternalAPIKey)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("api listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
