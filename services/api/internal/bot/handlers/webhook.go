package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const WebhookPath = "/webhook"

// SetWebhook registers webhookURL+WebhookPath with Telegram. This library
// version's WebhookConfig doesn't expose secret_token, so the setWebhook
// call is made directly to pass it through.
func (b *Bot) SetWebhook(webhookURL, secret string) error {
	params := tgbotapi.Params{"url": webhookURL + WebhookPath}
	if secret != "" {
		params["secret_token"] = secret
	}
	_, err := b.API.MakeRequest("setWebhook", params)
	return err
}

// WebhookHandler returns the http.HandlerFunc that processes incoming
// Telegram updates. Exported separately from RunWebhook so it can be
// mounted onto an existing router/server (e.g. the core API's, to run
// both in a single process/port on PaaS free tiers) instead of always
// owning its own listener.
func (b *Bot) WebhookHandler(ctx context.Context, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if secret != "" && r.Header.Get("X-Telegram-Bot-Api-Secret-Token") != secret {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		var update tgbotapi.Update
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			b.Logger.Error("decode webhook update", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		go b.dispatch(ctx, update)
	}
}

// RunWebhook is the standalone alternative to Run() for PaaS free tiers
// (Render, etc.) that only keep HTTP-serving "web services" alive for
// free: Telegram pushes updates to us over HTTPS instead of us
// long-polling for them. Used when the bot runs as its own process with
// its own listener (see cmd/bot); when embedded in another server, use
// SetWebhook + WebhookHandler directly instead.
func (b *Bot) RunWebhook(ctx context.Context, port, webhookURL, secret string) error {
	if err := b.SetWebhook(webhookURL, secret); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc(WebhookPath, b.WebhookHandler(ctx, secret))

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	b.Logger.Info("webhook server listening", "port", port, "webhook_url", webhookURL+WebhookPath)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
