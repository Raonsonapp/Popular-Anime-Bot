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

	_, lang, err := b.ensureUser(ctx, cq.From)
	if err != nil {
		b.Logger.Error("ensure user", "error", err)
	}

	parts := strings.Split(cq.Data, ":")
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "m":
		b.handleMenuCallback(ctx, chatID, telegramUserID, lang, parts)
	case "g":
		if len(parts) >= 2 && parts[1] == "list" {
			genres, err := b.Client.ListGenres(ctx)
			if err != nil {
				b.reply(chatID, i18n.T(lang, "genres_error"), nil)
				return
			}
			kb := keyboards.GenresKeyboard(lang, genres)
			b.reply(chatID, i18n.T(lang, "genres_prompt"), &kb)
		}
	case "gg":
		if len(parts) < 3 {
			return
		}
		genreID, _ := strconv.ParseInt(parts[1], 10, 64)
		page, _ := strconv.Atoi(parts[2])
		b.listByGenre(ctx, chatID, lang, genreID, page)
	case "s":
		if len(parts) < 3 {
			return
		}
		sort := parts[1]
		page, _ := strconv.Atoi(parts[2])
		b.listSorted(ctx, chatID, lang, sort, page)
	case "sq":
		if len(parts) < 3 {
			return
		}
		token, _ := strconv.ParseInt(parts[1], 10, 64)
		page, _ := strconv.Atoi(parts[2])
		b.repeatSearch(ctx, chatID, lang, token, page)
	case "lang":
		if len(parts) < 2 {
			return
		}
		b.changeLanguage(ctx, answer, telegramUserID, parts[1])
	case "a":
		if len(parts) < 2 {
			return
		}
		animeID, _ := strconv.ParseInt(parts[1], 10, 64)
		if len(parts) >= 3 && parts[2] == "fav" {
			b.toggleFavoriteCallback(ctx, cq, answer, lang, chatID, telegramUserID, animeID)
			return
		}
		b.sendAnimeDetail(ctx, chatID, telegramUserID, animeID)
	case "e":
		if len(parts) < 3 {
			return
		}
		animeID, _ := strconv.ParseInt(parts[1], 10, 64)
		page, _ := strconv.Atoi(parts[2])
		b.sendEpisodesList(ctx, chatID, lang, animeID, page)
	case "w":
		if len(parts) < 2 {
			return
		}
		episodeID, _ := strconv.ParseInt(parts[1], 10, 64)
		b.deliverEpisode(ctx, chatID, telegramUserID, episodeID)
	}
}

func (b *Bot) handleMenuCallback(ctx context.Context, chatID, telegramUserID int64, lang i18n.Lang, parts []string) {
	if len(parts) < 2 {
		return
	}
	switch parts[1] {
	case "main":
		kb := keyboards.MainMenu(lang)
		b.reply(chatID, i18n.T(lang, "welcome"), &kb)
	case "search":
		b.reply(chatID, i18n.T(lang, "search_prompt"), nil)
	case "lang":
		kb := keyboards.LanguageKeyboard()
		b.reply(chatID, i18n.T(lang, "choose_language"), &kb)
	case "random":
		a, err := b.Client.RandomAnime(ctx)
		if err != nil || a == nil {
			b.reply(chatID, i18n.T(lang, "random_not_found"), nil)
			return
		}
		b.sendAnimeDetail(ctx, chatID, telegramUserID, a.ID)
	case "favorites":
		page := 1
		if len(parts) >= 3 {
			page, _ = strconv.Atoi(parts[2])
		}
		b.listFavorites(ctx, chatID, lang, telegramUserID, page)
	case "history":
		page := 1
		if len(parts) >= 3 {
			page, _ = strconv.Atoi(parts[2])
		}
		b.listHistory(ctx, chatID, lang, telegramUserID, page)
	case "continue":
		b.listContinueWatching(ctx, chatID, lang, telegramUserID)
	}
}

func (b *Bot) changeLanguage(ctx context.Context, answer func(string), telegramUserID int64, code string) {
	newLang := i18n.Normalize(code)
	if err := b.Client.SetLanguage(ctx, telegramUserID, string(newLang)); err != nil {
		answer(i18n.T(newLang, "generic_error"))
		return
	}
	b.setUserLang(telegramUserID, newLang)
	answer(i18n.T(newLang, "language_changed"))
}

func (b *Bot) toggleFavoriteCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, answer func(string), lang i18n.Lang, chatID, telegramUserID, animeID int64) {
	userID, _, err := b.ensureUser(ctx, &tgbotapi.User{ID: telegramUserID})
	if err != nil || userID == 0 {
		answer(i18n.T(lang, "favorite_error"))
		return
	}
	added, err := b.Client.ToggleFavorite(ctx, userID, animeID)
	if err != nil {
		answer(i18n.T(lang, "favorite_error"))
		return
	}

	toast := i18n.T(lang, "favorite_removed")
	if added {
		toast = i18n.T(lang, "favorite_added")
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

func (b *Bot) listSorted(ctx context.Context, chatID int64, lang i18n.Lang, sort string, page int) {
	q := url.Values{}
	q.Set("sort", sort)
	q.Set("page", fmt.Sprint(page))
	resp, err := b.Client.ListAnime(ctx, q)
	if err != nil {
		b.reply(chatID, i18n.T(lang, "list_error"), nil)
		return
	}
	titleKeys := map[string]string{
		"top_rated": "top_rated_title",
		"trending":  "trending_title",
		"newest":    "newest_title",
	}
	b.sendAnimeSearchResults(chatID, lang, resp.Items, resp.Total, page, "s:"+sort, i18n.T(lang, titleKeys[sort]))
}

func (b *Bot) listByGenre(ctx context.Context, chatID int64, lang i18n.Lang, genreID int64, page int) {
	q := url.Values{}
	q.Set("genre_id", fmt.Sprint(genreID))
	q.Set("page", fmt.Sprint(page))
	resp, err := b.Client.ListAnime(ctx, q)
	if err != nil {
		b.reply(chatID, i18n.T(lang, "list_error"), nil)
		return
	}
	b.sendAnimeSearchResults(chatID, lang, resp.Items, resp.Total, page, fmt.Sprintf("gg:%d", genreID), i18n.T(lang, "genre_results_title"))
}

func (b *Bot) repeatSearch(ctx context.Context, chatID int64, lang i18n.Lang, token int64, page int) {
	query, ok := b.loadSearchQuery(token)
	if !ok {
		b.reply(chatID, i18n.T(lang, "search_error"), nil)
		return
	}
	q := url.Values{}
	q.Set("search", query)
	q.Set("page", fmt.Sprint(page))
	resp, err := b.Client.ListAnime(ctx, q)
	if err != nil {
		b.reply(chatID, i18n.T(lang, "search_error"), nil)
		return
	}
	b.sendAnimeSearchResults(chatID, lang, resp.Items, resp.Total, page, fmt.Sprintf("sq:%d", token), i18n.Tf(lang, "search_results_header", query))
}

func (b *Bot) listFavorites(ctx context.Context, chatID int64, lang i18n.Lang, telegramUserID int64, page int) {
	userID, _, err := b.ensureUser(ctx, &tgbotapi.User{ID: telegramUserID})
	if err != nil || userID == 0 {
		b.reply(chatID, i18n.T(lang, "generic_error"), nil)
		return
	}
	resp, err := b.Client.ListFavorites(ctx, userID, page)
	if err != nil {
		b.reply(chatID, i18n.T(lang, "favorite_error"), nil)
		return
	}
	if len(resp.Items) == 0 {
		b.reply(chatID, i18n.T(lang, "favorites_empty"), nil)
		return
	}
	b.sendAnimeSearchResults(chatID, lang, resp.Items, resp.Total, page, "m:favorites", i18n.T(lang, "favorites_header"))
}

func (b *Bot) listHistory(ctx context.Context, chatID int64, lang i18n.Lang, telegramUserID int64, page int) {
	userID, _, err := b.ensureUser(ctx, &tgbotapi.User{ID: telegramUserID})
	if err != nil || userID == 0 {
		b.reply(chatID, i18n.T(lang, "generic_error"), nil)
		return
	}
	resp, err := b.Client.ContinueWatching(ctx, userID)
	if err != nil {
		b.reply(chatID, i18n.T(lang, "history_error"), nil)
		return
	}
	if len(resp) == 0 {
		b.reply(chatID, i18n.T(lang, "history_empty"), nil)
		return
	}
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, entry := range resp {
		label := i18n.Tf(lang, "history_episode_label", entry.EpisodeID)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("w:%d", entry.EpisodeID)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(keyboards.BackToMainRow(lang)))
	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.reply(chatID, i18n.T(lang, "history_header"), &kb)
}

func (b *Bot) listContinueWatching(ctx context.Context, chatID int64, lang i18n.Lang, telegramUserID int64) {
	b.listHistory(ctx, chatID, lang, telegramUserID, 1)
}
