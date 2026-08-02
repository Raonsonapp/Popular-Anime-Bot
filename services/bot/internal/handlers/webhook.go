package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const webhookPath = "/webhook"

// RunWebhook is the alternative to Run() for PaaS free tiers (Render, etc.)
// that only keep HTTP-serving "web services" alive for free: Telegram
// pushes updates to us over HTTPS instead of us long-polling for them.
func (b *Bot) RunWebhook(ctx context.Context, port, webhookURL, secret string) error {
	// This library version's WebhookConfig doesn't expose secret_token, so
	// the setWebhook call is made directly to pass it through.
	params := tgbotapi.Params{"url": webhookURL + webhookPath}
	if secret != "" {
		params["secret_token"] = secret
	}
	if _, err := b.API.MakeRequest("setWebhook", params); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc(webhookPath, func(w http.ResponseWriter, r *http.Request) {
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
	})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	b.Logger.Info("webhook server listening", "port", port, "webhook_url", webhookURL+webhookPath)

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
