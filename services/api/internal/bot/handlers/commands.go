package handlers

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"popular-anime-bot/api/internal/bot/i18n"
	"popular-anime-bot/api/internal/bot/keyboards"
)

func (b *Bot) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	_, lang, err := b.ensureUser(ctx, msg.From)
	if err != nil {
		b.Logger.Error("ensure user", "error", err)
	}

	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			b.handleStart(ctx, msg, lang)
		default:
			kb := keyboards.MainMenu(lang)
			b.reply(msg.Chat.ID, i18n.T(lang, "welcome"), &kb)
		}
		return
	}

	query := strings.TrimSpace(msg.Text)
	if query == "" {
		return
	}
	b.handleSearch(ctx, msg.Chat.ID, lang, query)
}

func (b *Bot) handleStart(ctx context.Context, msg *tgbotapi.Message, lang i18n.Lang) {
	payload := strings.TrimSpace(msg.CommandArguments())
	if strings.HasPrefix(payload, "anime_") {
		idStr := strings.TrimPrefix(payload, "anime_")
		if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			b.sendAnimeDetail(ctx, msg.Chat.ID, msg.From.ID, id)
			// Coming from a channel's "Watch in bot" deep link, the user
			// already tapped "watch" once to get here - showing the
			// season/episode picker immediately (same as tapping the
			// "Watch" button would) skips a redundant second tap.
			b.sendWatchEntry(ctx, msg.Chat.ID, lang, id)
			return
		}
	}
	kb := keyboards.MainMenu(lang)
	b.reply(msg.Chat.ID, i18n.T(lang, "welcome"), &kb)
}

func (b *Bot) handleSearch(ctx context.Context, chatID int64, lang i18n.Lang, query string) {
	q := url.Values{}
	q.Set("search", query)
	q.Set("page", "1")
	resp, err := b.Client.ListAnime(ctx, q)
	if err != nil {
		b.Logger.Error("search anime", "error", err)
		b.reply(chatID, i18n.T(lang, "search_error"), nil)
		return
	}
	token := b.storeSearchQuery(query)
	b.sendAnimeSearchResults(chatID, lang, resp.Items, resp.Total, 1, fmt.Sprintf("sq:%d", token), i18n.Tf(lang, "search_results_header", query))
}
