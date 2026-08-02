package handlers

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"popular-anime-bot/bot/internal/apiclient"
	"popular-anime-bot/bot/internal/i18n"
)

type userInfo struct {
	id   int64
	lang i18n.Lang
}

type Bot struct {
	API    *tgbotapi.BotAPI
	Client *apiclient.Client
	Logger *slog.Logger

	userCache sync.Map // telegram user id -> userInfo

	// Search queries can't be embedded directly in callback_data (Telegram's
	// 64-byte limit), so pagination on search results goes through a short
	// in-memory token instead.
	searchQueries sync.Map // token -> query string
	searchCounter atomic.Int64
}

// storeSearchQuery keeps queries in-process for the life of the bot. Fine for
// an MVP single-instance deployment; a multi-instance/production deployment
// should move this to Redis with a short TTL instead.
func (b *Bot) storeSearchQuery(query string) int64 {
	token := b.searchCounter.Add(1)
	b.searchQueries.Store(token, query)
	return token
}

func (b *Bot) loadSearchQuery(token int64) (string, bool) {
	v, ok := b.searchQueries.Load(token)
	if !ok {
		return "", false
	}
	return v.(string), true
}

func New(api *tgbotapi.BotAPI, client *apiclient.Client, logger *slog.Logger) *Bot {
	return &Bot{API: api, Client: client, Logger: logger}
}

func (b *Bot) Run(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := b.API.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			b.API.StopReceivingUpdates()
			return
		case update := <-updates:
			go b.dispatch(ctx, update)
		}
	}
}

func (b *Bot) dispatch(ctx context.Context, update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			b.Logger.Error("panic recovered in update handler", "error", r)
		}
	}()

	switch {
	case update.Message != nil:
		b.handleMessage(ctx, update.Message)
	case update.CallbackQuery != nil:
		b.handleCallback(ctx, update.CallbackQuery)
	}
}

// ensureUser upserts the Telegram user in the catalog and returns the
// internal (DB) user id and their language preference, caching the mapping
// to avoid a round trip per update.
func (b *Bot) ensureUser(ctx context.Context, from *tgbotapi.User) (int64, i18n.Lang, error) {
	if from == nil {
		return 0, i18n.Default, nil
	}
	if cached, ok := b.userCache.Load(from.ID); ok {
		info := cached.(userInfo)
		return info.id, info.lang, nil
	}

	u, err := b.Client.TouchUser(ctx, from.ID, from.UserName, from.FirstName, string(i18n.Default))
	if err != nil {
		return 0, i18n.Default, err
	}
	lang := i18n.Normalize(u.LanguagePref)
	b.userCache.Store(from.ID, userInfo{id: u.ID, lang: lang})
	return u.ID, lang, nil
}

// setUserLang updates the cached language immediately after the user picks
// a new one, without waiting for another TouchUser round trip.
func (b *Bot) setUserLang(telegramUserID int64, lang i18n.Lang) {
	if cached, ok := b.userCache.Load(telegramUserID); ok {
		info := cached.(userInfo)
		info.lang = lang
		b.userCache.Store(telegramUserID, info)
	}
}

func (b *Bot) reply(chatID int64, text string, kb *tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	if kb != nil {
		msg.ReplyMarkup = kb
	}
	if _, err := b.API.Send(msg); err != nil {
		b.Logger.Error("send message", "error", err)
	}
}

func (b *Bot) editMessage(chatID int64, messageID int, text string, kb *tgbotapi.InlineKeyboardMarkup) {
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = tgbotapi.ModeHTML
	if kb != nil {
		edit.ReplyMarkup = kb
	}
	if _, err := b.API.Send(edit); err != nil {
		// Editing a message that had a photo attached fails (can't convert
		// media message to text-only); fall back to a fresh message.
		b.reply(chatID, text, kb)
	}
}

func (b *Bot) answerCallback(id string, text string) {
	cb := tgbotapi.NewCallback(id, text)
	if _, err := b.API.Request(cb); err != nil {
		b.Logger.Error("answer callback", "error", err)
	}
}
