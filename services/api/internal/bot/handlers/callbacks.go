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

func (b *Bot) handleCallback(ctx context.Context, cq *tgbotapi.CallbackQuery) {
	answered := false
	answer := func(text string) {
		if answered {
			return
		}
		answered = true
		b.answerCallback(cq.ID, text)
	}
	defer answer("")

	if cq.Message == nil {
		return
	}
	chatID := cq.Message.Chat.ID
	telegramUserID := cq.From.ID

	parts := strings.Split(cq.Data, ":")
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "m":
		b.handleMenuCallback(ctx, chatID, telegramUserID, parts)
	case "g":
		if len(parts) >= 2 && parts[1] == "list" {
			genres, err := b.Client.ListGenres(ctx)
			if err != nil {
				b.reply(chatID, "❌ خطا در دریافت ژانرها.", nil)
				return
			}
			kb := keyboards.GenresKeyboard(genres)
			b.reply(chatID, "🎭 یک ژانر را انتخاب کنید:", &kb)
		}
	case "gg":
		if len(parts) < 3 {
			return
		}
		genreID, _ := strconv.ParseInt(parts[1], 10, 64)
		page, _ := strconv.Atoi(parts[2])
		b.listByGenre(ctx, chatID, genreID, page)
	case "s":
		if len(parts) < 3 {
			return
		}
		sort := parts[1]
		page, _ := strconv.Atoi(parts[2])
		b.listSorted(ctx, chatID, sort, page)
	case "sq":
		if len(parts) < 3 {
			return
		}
		token, _ := strconv.ParseInt(parts[1], 10, 64)
		page, _ := strconv.Atoi(parts[2])
		b.repeatSearch(ctx, chatID, token, page)
	case "a":
		if len(parts) < 2 {
			return
		}
		animeID, _ := strconv.ParseInt(parts[1], 10, 64)
		if len(parts) >= 3 && parts[2] == "fav" {
			b.toggleFavoriteCallback(ctx, cq, answer, chatID, telegramUserID, animeID)
			return
		}
		b.sendAnimeDetail(ctx, chatID, telegramUserID, animeID)
	case "e":
		if len(parts) < 3 {
			return
		}
		animeID, _ := strconv.ParseInt(parts[1], 10, 64)
		page, _ := strconv.Atoi(parts[2])
		b.sendEpisodesList(ctx, chatID, animeID, page)
	case "w":
		if len(parts) < 2 {
			return
		}
		episodeID, _ := strconv.ParseInt(parts[1], 10, 64)
		b.deliverEpisode(ctx, chatID, telegramUserID, episodeID)
	}
}

func (b *Bot) handleMenuCallback(ctx context.Context, chatID, telegramUserID int64, parts []string) {
	if len(parts) < 2 {
		return
	}
	switch parts[1] {
	case "main":
		kb := keyboards.MainMenu()
		b.reply(chatID, welcomeText, &kb)
	case "search":
		b.reply(chatID, "🔍 نام آنیمه مورد نظر خود را تایپ کنید:", nil)
	case "random":
		a, err := b.Client.RandomAnime(ctx)
		if err != nil || a == nil {
			b.reply(chatID, "❌ آنیمه‌ای یافت نشد.", nil)
			return
		}
		b.sendAnimeDetail(ctx, chatID, telegramUserID, a.ID)
	case "favorites":
		page := 1
		if len(parts) >= 3 {
			page, _ = strconv.Atoi(parts[2])
		}
		b.listFavorites(ctx, chatID, telegramUserID, page)
	case "history":
		page := 1
		if len(parts) >= 3 {
			page, _ = strconv.Atoi(parts[2])
		}
		b.listHistory(ctx, chatID, telegramUserID, page)
	case "continue":
		b.listContinueWatching(ctx, chatID, telegramUserID)
	}
}

func (b *Bot) toggleFavoriteCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, answer func(string), chatID, telegramUserID, animeID int64) {
	userID, err := b.ensureUser(ctx, &tgbotapi.User{ID: telegramUserID})
	if err != nil || userID == 0 {
		answer("❌ خطا در ثبت علاقه‌مندی.")
		return
	}
	added, err := b.Client.ToggleFavorite(ctx, userID, animeID)
	if err != nil {
		answer("❌ خطا در ثبت علاقه‌مندی.")
		return
	}

	toast := "💔 از علاقه‌مندی‌ها حذف شد."
	if added {
		toast = "⭐ به علاقه‌مندی‌ها اضافه شد."
	}

	// Editing/replacing the message only makes sense in a 1-on-1 bot chat;
	// a channel showcase post should just get a toast, not a follow-up message.
	if cq.Message.Chat.IsChannel() {
		answer(toast)
		return
	}
	answer(toast)
	b.sendAnimeDetail(ctx, chatID, telegramUserID, animeID)
}

func (b *Bot) listSorted(ctx context.Context, chatID int64, sort string, page int) {
	q := url.Values{}
	q.Set("sort", sort)
	q.Set("page", fmt.Sprint(page))
	resp, err := b.Client.ListAnime(ctx, q)
	if err != nil {
		b.reply(chatID, "❌ خطا در دریافت لیست.", nil)
		return
	}
	titles := map[string]string{
		"top_rated": "⭐ برترین آنیمه‌ها",
		"trending":  "🔥 پرطرفدارترین‌ها",
		"newest":    "🆕 جدیدترین‌ها",
	}
	b.sendAnimeSearchResults(chatID, resp.Items, resp.Total, page, "s:"+sort, titles[sort])
}

func (b *Bot) listByGenre(ctx context.Context, chatID int64, genreID int64, page int) {
	q := url.Values{}
	q.Set("genre_id", fmt.Sprint(genreID))
	q.Set("page", fmt.Sprint(page))
	resp, err := b.Client.ListAnime(ctx, q)
	if err != nil {
		b.reply(chatID, "❌ خطا در دریافت لیست.", nil)
		return
	}
	b.sendAnimeSearchResults(chatID, resp.Items, resp.Total, page, fmt.Sprintf("gg:%d", genreID), "🎭 آنیمه‌های این ژانر:")
}

func (b *Bot) repeatSearch(ctx context.Context, chatID int64, token int64, page int) {
	query, ok := b.loadSearchQuery(token)
	if !ok {
		b.reply(chatID, "⌛️ این جستجو منقضی شده. لطفاً دوباره تایپ کنید.", nil)
		return
	}
	q := url.Values{}
	q.Set("search", query)
	q.Set("page", fmt.Sprint(page))
	resp, err := b.Client.ListAnime(ctx, q)
	if err != nil {
		b.reply(chatID, "❌ خطا در جستجو.", nil)
		return
	}
	b.sendAnimeSearchResults(chatID, resp.Items, resp.Total, page, fmt.Sprintf("sq:%d", token), "🔎 نتایج جستجو برای: "+query)
}

func (b *Bot) listFavorites(ctx context.Context, chatID, telegramUserID int64, page int) {
	userID, err := b.ensureUser(ctx, &tgbotapi.User{ID: telegramUserID})
	if err != nil || userID == 0 {
		b.reply(chatID, "❌ خطا.", nil)
		return
	}
	resp, err := b.Client.ListFavorites(ctx, userID, page)
	if err != nil {
		b.reply(chatID, "❌ خطا در دریافت علاقه‌مندی‌ها.", nil)
		return
	}
	if len(resp.Items) == 0 {
		b.reply(chatID, "هنوز چیزی به علاقه‌مندی‌ها اضافه نکرده‌اید. ❤️", nil)
		return
	}
	b.sendAnimeSearchResults(chatID, resp.Items, resp.Total, page, "m:favorites", "❤️ علاقه‌مندی‌های شما:")
}

func (b *Bot) listHistory(ctx context.Context, chatID, telegramUserID int64, page int) {
	userID, err := b.ensureUser(ctx, &tgbotapi.User{ID: telegramUserID})
	if err != nil || userID == 0 {
		b.reply(chatID, "❌ خطا.", nil)
		return
	}
	resp, err := b.Client.ContinueWatching(ctx, userID)
	if err != nil {
		b.reply(chatID, "❌ خطا در دریافت تاریخچه.", nil)
		return
	}
	if len(resp) == 0 {
		b.reply(chatID, "تاریخچه‌ای موجود نیست. 🕒", nil)
		return
	}
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, entry := range resp {
		label := fmt.Sprintf("قسمت #%d", entry.EpisodeID)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("w:%d", entry.EpisodeID)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(keyboards.BackToMainRow()))
	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.reply(chatID, "🕒 تاریخچه تماشای شما:", &kb)
}

func (b *Bot) listContinueWatching(ctx context.Context, chatID, telegramUserID int64) {
	b.listHistory(ctx, chatID, telegramUserID, 1)
}
