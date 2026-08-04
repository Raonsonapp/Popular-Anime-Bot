package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	botapiclient "popular-anime-bot/api/internal/bot/apiclient"
	bothandlers "popular-anime-bot/api/internal/bot/handlers"
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
	seasonRepo := postgres.NewSeasonRepo(db)
	genreRepo := postgres.NewGenreRepo(db)
	studioRepo := postgres.NewStudioRepo(db)
	sourceChannelRepo := postgres.NewSourceChannelRepo(db)
	userRepo := postgres.NewUserRepo(db)
	favoriteRepo := postgres.NewFavoriteRepo(db)
	historyRepo := postgres.NewHistoryRepo(db)
	channelPostRepo := postgres.NewChannelPostRepo(db)
	importLogRepo := postgres.NewImportLogRepo(db)
	listenerSessionRepo := postgres.NewListenerSessionRepo(db)

	catalog := usecase.NewCatalogService(animeRepo, episodeRepo, seasonRepo, genreRepo, studioRepo)
	ingest := usecase.NewIngestService(animeRepo, episodeRepo, seasonRepo, genreRepo, studioRepo, sourceChannelRepo, importLogRepo)
	users := usecase.NewUserService(userRepo, favoriteRepo, historyRepo)
	publish := usecase.NewPublishService(animeRepo, channelPostRepo)

	handler := httpDelivery.NewHandler(catalog, ingest, users, publish, sourceChannelRepo, listenerSessionRepo, logger)
	router := httpDelivery.NewRouter(handler, cfg.InternalAPIKey)

	// Optionally embed the Telegram bot in this same process/port - lets a
	// single free Render Web Service run both. See docs/RENDER.md.
	var runBotPolling func()
	if cfg.BotToken != "" {
		tgAPI, err := tgbotapi.NewBotAPI(cfg.BotToken)
		if err != nil {
			logger.Error("create bot api", "error", err)
			os.Exit(1)
		}
		logger.Info("bot authorized", "username", tgAPI.Self.UserName)

		botClient := botapiclient.New("http://localhost:"+cfg.Port, cfg.InternalAPIKey)
		bot := bothandlers.New(tgAPI, botClient, logger)

		if cfg.WebhookURL != "" {
			if err := bot.SetWebhook(cfg.WebhookURL, cfg.WebhookSecret); err != nil {
				logger.Error("set webhook", "error", err)
				os.Exit(1)
			}
			router.Post(bothandlers.WebhookPath, bot.WebhookHandler(ctx, cfg.WebhookSecret))
			logger.Info("bot webhook mounted", "path", bothandlers.WebhookPath, "webhook_url", cfg.WebhookURL+bothandlers.WebhookPath)
		} else {
			runBotPolling = func() { bot.Run(ctx) }
		}
	}

	// The MTProto listener's web-based login helper (used when Render's
	// Shell isn't available, e.g. the free tier) runs on a local-only port
	// inside the same container - proxy it through so it's reachable at
	// this service's public URL. Harmless no-op (502s) once login is done
	// and the helper has exited. See docs/RENDER.md.
	loginTarget, _ := url.Parse("http://localhost:8091")
	router.Handle("/telegram-login/*", httputil.NewSingleHostReverseProxy(loginTarget))

	// Same trick for the scheduler's on-demand publish trigger - lets a
	// showcase-channel post be forced out immediately instead of waiting
	// for the next cron tick. See services/scheduler/cmd/scheduler/main.go.
	schedulerTarget, _ := url.Parse("http://localhost:8092")
	router.Handle("/scheduler/*", httputil.NewSingleHostReverseProxy(schedulerTarget))

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

	if runBotPolling != nil {
		go runBotPolling()
	}

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
