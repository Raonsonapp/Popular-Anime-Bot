package keyboards

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"popular-anime-bot/api/internal/bot/apiclient"
	"popular-anime-bot/api/internal/bot/i18n"
)

func MainMenu(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	t := func(key string) string { return i18n.T(lang, key) }
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(t("menu_search"), "m:search"),
			tgbotapi.NewInlineKeyboardButtonData(t("menu_genres"), "g:list"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(t("menu_top_rated"), "s:top_rated:1"),
			tgbotapi.NewInlineKeyboardButtonData(t("menu_trending"), "s:trending:1"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(t("menu_newest"), "s:newest:1"),
			tgbotapi.NewInlineKeyboardButtonData(t("menu_random"), "m:random"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(t("menu_continue"), "m:continue"),
			tgbotapi.NewInlineKeyboardButtonData(t("menu_favorites"), "m:favorites:1"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(t("menu_history"), "m:history:1"),
			tgbotapi.NewInlineKeyboardButtonData(t("menu_language"), "m:lang"),
		),
	)
}

func BackToMainRow(lang i18n.Lang) tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "back_to_menu"), "m:main")
}

func LanguageKeyboard() tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, l := range i18n.All {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(l.DisplayName(), fmt.Sprintf("lang:%s", l)),
		))
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func AnimeListKeyboard(lang i18n.Lang, items []apiclient.Anime, page, total, pageSize int, pagePrefix string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, a := range items {
		label := fmt.Sprintf("%s %s", a.Title, i18n.Tf(lang, "caption_episodes_suffix", a.EpisodesCount))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("a:%d", a.ID)),
		))
	}

	var nav []tgbotapi.InlineKeyboardButton
	if page > 1 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "prev"), fmt.Sprintf("%s:%d", pagePrefix, page-1)))
	}
	if page*pageSize < total {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "next"), fmt.Sprintf("%s:%d", pagePrefix, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, nav)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(BackToMainRow(lang)))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func AnimeDetailKeyboard(lang i18n.Lang, a apiclient.Anime, isFavorite bool, botUsername string) tgbotapi.InlineKeyboardMarkup {
	favLabel := i18n.T(lang, "add_favorite")
	if isFavorite {
		favLabel = i18n.T(lang, "remove_favorite")
	}
	shareURL := fmt.Sprintf("https://t.me/%s?start=anime_%d", botUsername, a.ID)

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "watch"), fmt.Sprintf("sn:%d", a.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(favLabel, fmt.Sprintf("a:%d:fav", a.ID)),
			tgbotapi.NewInlineKeyboardButtonURL(i18n.T(lang, "share"), shareURL),
		),
		tgbotapi.NewInlineKeyboardRow(BackToMainRow(lang)),
	)
}

// SeasonsKeyboard lets the user pick which season to browse, when an
// anime has more than one on record (see Bot.sendWatchEntry).
func SeasonsKeyboard(lang i18n.Lang, animeID int64, seasons []apiclient.Season) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, s := range seasons {
		label := i18n.Tf(lang, "season_button", s.SeasonNumber, s.EpisodesCount)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("e:%d:%d:1", animeID, s.ID)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "back_to_anime"), fmt.Sprintf("a:%d", animeID)),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func EpisodesKeyboard(lang i18n.Lang, animeID, seasonID int64, episodes []apiclient.Episode, page, total, pageSize int) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton
	for i, e := range episodes {
		label := i18n.Tf(lang, "episode_button", e.EpisodeNumber, e.Quality)
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("w:%d", e.ID)))
		if (i+1)%2 == 0 {
			rows = append(rows, row)
			row = nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}

	var nav []tgbotapi.InlineKeyboardButton
	if page > 1 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "prev"), fmt.Sprintf("e:%d:%d:%d", animeID, seasonID, page-1)))
	}
	if page*pageSize < total {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "next"), fmt.Sprintf("e:%d:%d:%d", animeID, seasonID, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, nav)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "back_to_anime"), fmt.Sprintf("a:%d", animeID)),
	))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func GenresKeyboard(lang i18n.Lang, genres []apiclient.Genre) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton
	for i, g := range genres {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(g.NamePersian, fmt.Sprintf("gg:%d:1", g.ID)))
		if (i+1)%2 == 0 {
			rows = append(rows, row)
			row = nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(BackToMainRow(lang)))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
