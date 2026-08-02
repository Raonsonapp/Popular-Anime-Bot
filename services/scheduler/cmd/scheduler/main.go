package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/robfig/cron/v3"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	apiclient "popular-anime-bot/scheduler/internal/client"
	"popular-anime-bot/scheduler/internal/config"
	"popular-anime-bot/scheduler/internal/publisher"
)

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

	c := cron.New()

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
		"target_channel_id", cfg.TargetChannelID,
	)

	<-ctx.Done()
	logger.Info("shutting down scheduler")
	stopCtx := c.Stop()
	<-stopCtx.Done()
}
