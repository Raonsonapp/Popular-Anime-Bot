package keyboards

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"popular-anime-bot/api/internal/bot/apiclient"
)

func MainMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔍 جستجو", "m:search"),
			tgbotapi.NewInlineKeyboardButtonData("🎭 ژانرها", "g:list"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⭐ برترین‌ها", "s:top_rated:1"),
			tgbotapi.NewInlineKeyboardButtonData("🔥 پرطرفدار", "s:trending:1"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🆕 جدیدترین", "s:newest:1"),
			tgbotapi.NewInlineKeyboardButtonData("🎲 تصادفی", "m:random"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("▶️ ادامه تماشا", "m:continue"),
			tgbotapi.NewInlineKeyboardButtonData("❤️ علاقه‌مندی‌ها", "m:favorites:1"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🕒 تاریخچه", "m:history:1"),
		),
	)
}

func BackToMainRow() tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت به منو", "m:main")
}

func AnimeListKeyboard(items []apiclient.Anime, page, total, pageSize int, pagePrefix string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, a := range items {
		label := fmt.Sprintf("%s (%d قسمت)", a.Title, a.EpisodesCount)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("a:%d", a.ID)),
		))
	}

	var nav []tgbotapi.InlineKeyboardButton
	if page > 1 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("⬅️ قبلی", fmt.Sprintf("%s:%d", pagePrefix, page-1)))
	}
	if page*pageSize < total {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("➡️ بعدی", fmt.Sprintf("%s:%d", pagePrefix, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, nav)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(BackToMainRow()))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func AnimeDetailKeyboard(a apiclient.Anime, isFavorite bool, botUsername string) tgbotapi.InlineKeyboardMarkup {
	favLabel := "⭐ افزودن به علاقه‌مندی"
	if isFavorite {
		favLabel = "💔 حذف از علاقه‌مندی"
	}
	shareURL := fmt.Sprintf("https://t.me/%s?start=anime_%d", botUsername, a.ID)

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("▶️ تماشا", fmt.Sprintf("e:%d:1", a.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(favLabel, fmt.Sprintf("a:%d:fav", a.ID)),
			tgbotapi.NewInlineKeyboardButtonURL("📢 اشتراک‌گذاری", shareURL),
		),
		tgbotapi.NewInlineKeyboardRow(BackToMainRow()),
	)
}

func EpisodesKeyboard(animeID int64, episodes []apiclient.Episode, page, total, pageSize int) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton
	for i, e := range episodes {
		label := fmt.Sprintf("قسمت %d [%s]", e.EpisodeNumber, e.Quality)
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
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("⬅️ قبلی", fmt.Sprintf("e:%d:%d", animeID, page-1)))
	}
	if page*pageSize < total {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("➡️ بعدی", fmt.Sprintf("e:%d:%d", animeID, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, nav)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت به آنیمه", fmt.Sprintf("a:%d", animeID)),
	))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func GenresKeyboard(genres []apiclient.Genre) tgbotapi.InlineKeyboardMarkup {
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
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(BackToMainRow()))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
