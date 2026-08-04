package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // embeds the IANA database, so LoadLocation works even on minimal base images without it installed

	"github.com/robfig/cron/v3"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	apiclient "popular-anime-bot/scheduler/internal/client"
	"popular-anime-bot/scheduler/internal/config"
	"popular-anime-bot/scheduler/internal/publisher"
)

// TriggerPort is where a tiny local-only HTTP server listens for an
// on-demand publish run - reachable publicly via the Go API's reverse
// proxy at /scheduler/trigger (see cmd/api/main.go), for whenever waiting
// for the next cron tick isn't good enough (e.g. catching up after a bug
// that skipped the scheduled runs).
const TriggerPort = "8092"

func startTriggerServer(ctx context.Context, cfg config.Config, api *tgbotapi.BotAPI, cl *apiclient.Client, logger *slog.Logger) {
	mux := http.NewServeMux()
	mux.HandleFunc("/scheduler/trigger", func(w http.ResponseWriter, r *http.Request) {
		if cfg.InternalAPIKey == "" || r.URL.Query().Get("key") != cfg.InternalAPIKey {
			http.Error(w, "invalid or missing ?key=", http.StatusForbidden)
			return
		}
		logger.Info("manual publish trigger requested")
		go publisher.Run(ctx, cfg, api, cl, logger)
		w.Write([]byte("Publish run triggered - check the Render logs for progress."))
	})

	srv := &http.Server{Addr: "127.0.0.1:" + TriggerPort, Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("trigger server error", "error", err)
		}
	}()
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	if cfg.BotToken == "" {
		logger.Error("BOT_TOKEN is required")
		os.Exit(1)
	}

	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		logger.Error("create bot api", "error", err)
		os.Exit(1)
	}

	if cfg.BotUsername == "" {
		cfg.BotUsername = api.Self.UserName
	}

	cl := apiclient.New(cfg.APIBaseURL, cfg.InternalAPIKey)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The default schedule (8am/8pm) means Dushanbe local time regardless
	// of the server's own timezone (typically UTC on Render/most VPSes).
	loc, err := time.LoadLocation("Asia/Dushanbe")
	if err != nil {
		logger.Warn("could not load Asia/Dushanbe timezone, falling back to UTC", "error", err)
		loc = time.UTC
	}
	startTriggerServer(ctx, cfg, api, cl, logger)

	c := cron.New(cron.WithLocation(loc))

	if _, err := c.AddFunc(cfg.PostCronSchedule, func() {
		logger.Info("running scheduled publish job")
		publisher.Run(ctx, cfg, api, cl, logger)
	}); err != nil {
		logger.Error("schedule publish job", "error", err)
		os.Exit(1)
	}

	if _, err := c.AddFunc(cfg.PopularityCron, func() {
		logger.Info("refreshing popularity scores")
		if err := cl.RefreshPopularity(ctx); err != nil {
			logger.Error("refresh popularity", "error", err)
		}
	}); err != nil {
		logger.Error("schedule popularity job", "error", err)
		os.Exit(1)
	}

	c.Start()
	logger.Info("scheduler started",
		"post_cron", cfg.PostCronSchedule,
		"popularity_cron", cfg.PopularityCron,
		"target_channel_ids", cfg.TargetChannelIDs,
	)

	<-ctx.Done()
	logger.Info("shutting down scheduler")
	stopCtx := c.Stop()
	<-stopCtx.Done()
}
