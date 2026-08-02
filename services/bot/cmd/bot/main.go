package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"popular-anime-bot/bot/internal/apiclient"
	"popular-anime-bot/bot/internal/config"
	"popular-anime-bot/bot/internal/handlers"
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
	logger.Info("authorized", "username", api.Self.UserName)

	client := apiclient.New(cfg.APIBaseURL, cfg.InternalAPIKey)
	bot := handlers.New(api, client, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.WebhookURL != "" {
		if err := bot.RunWebhook(ctx, cfg.Port, cfg.WebhookURL, cfg.WebhookSecret); err != nil {
			logger.Error("webhook server", "error", err)
			os.Exit(1)
		}
	} else {
		bot.Run(ctx)
	}
	logger.Info("bot stopped")
}
