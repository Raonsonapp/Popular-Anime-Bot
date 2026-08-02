package handlers

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"popular-anime-bot/api/internal/bot/keyboards"
)

const welcomeText = `👋 به <b>AnimeBot</b> خوش آمدید!

بهترین آنیمه‌های فارسی‌زبان را اینجا تماشا کنید 🎬

برای جستجو، فقط نام آنیمه را تایپ کنید یا از منوی زیر استفاده کنید.`

func (b *Bot) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	if _, err := b.ensureUser(ctx, msg.From); err != nil {
		b.Logger.Error("ensure user", "error", err)
	}

	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			b.handleStart(ctx, msg)
		default:
			kb := keyboards.MainMenu()
			b.reply(msg.Chat.ID, welcomeText, &kb)
		}
		return
	}

	query := strings.TrimSpace(msg.Text)
	if query == "" {
		return
	}
	b.handleSearch(ctx, msg.Chat.ID, query)
}

func (b *Bot) handleStart(ctx context.Context, msg *tgbotapi.Message) {
	payload := strings.TrimSpace(msg.CommandArguments())
	if strings.HasPrefix(payload, "anime_") {
		idStr := strings.TrimPrefix(payload, "anime_")
		if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			b.sendAnimeDetail(ctx, msg.Chat.ID, msg.From.ID, id)
			return
		}
	}
	kb := keyboards.MainMenu()
	b.reply(msg.Chat.ID, welcomeText, &kb)
}

func (b *Bot) handleSearch(ctx context.Context, chatID int64, query string) {
	q := url.Values{}
	q.Set("search", query)
	q.Set("page", "1")
	resp, err := b.Client.ListAnime(ctx, q)
	if err != nil {
		b.Logger.Error("search anime", "error", err)
		b.reply(chatID, "❌ خطا در جستجو. لطفاً دوباره تلاش کنید.", nil)
		return
	}
	token := b.storeSearchQuery(query)
	b.sendAnimeSearchResults(chatID, resp.Items, resp.Total, 1, fmt.Sprintf("sq:%d", token), "🔎 نتایج جستجو برای: "+query)
}
