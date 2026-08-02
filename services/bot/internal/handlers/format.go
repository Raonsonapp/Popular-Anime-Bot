package handlers

import (
	"fmt"
	"strings"

	"popular-anime-bot/bot/internal/apiclient"
	"popular-anime-bot/bot/internal/i18n"
)

var statusKeys = map[string]string{
	"ongoing":   "status_ongoing",
	"completed": "status_completed",
	"announced": "status_announced",
}

func formatAnimeCaption(lang i18n.Lang, a apiclient.Anime) string {
	var b strings.Builder

	fmt.Fprintf(&b, "🎬 <b>%s</b>\n", htmlEscape(a.Title))
	if a.TitleJapanese != nil && *a.TitleJapanese != "" {
		fmt.Fprintf(&b, "🇯🇵 %s\n", htmlEscape(*a.TitleJapanese))
	}
	if a.TitleEnglish != nil && *a.TitleEnglish != "" {
		fmt.Fprintf(&b, "🇬🇧 %s\n", htmlEscape(*a.TitleEnglish))
	}
	b.WriteString("\n")

	if len(a.Genres) > 0 {
		names := make([]string, 0, len(a.Genres))
		for _, g := range a.Genres {
			names = append(names, g.NamePersian)
		}
		fmt.Fprintf(&b, "%s\n", i18n.Tf(lang, "caption_genre", strings.Join(names, "، ")))
	}
	if a.StudioName != "" {
		fmt.Fprintf(&b, "%s\n", i18n.Tf(lang, "caption_studio", htmlEscape(a.StudioName)))
	}
	if a.Year != nil {
		fmt.Fprintf(&b, "%s\n", i18n.Tf(lang, "caption_year", *a.Year))
	}
	if key, ok := statusKeys[a.Status]; ok {
		fmt.Fprintf(&b, "%s\n", i18n.Tf(lang, "caption_status", i18n.T(lang, key)))
	}
	fmt.Fprintf(&b, "%s\n", i18n.Tf(lang, "caption_rating", a.RatingScore))
	fmt.Fprintf(&b, "%s\n", i18n.Tf(lang, "caption_episodes_count", a.EpisodesCount))

	if a.SynopsisPersian != nil && *a.SynopsisPersian != "" {
		fmt.Fprintf(&b, "\n%s\n", i18n.Tf(lang, "caption_synopsis", htmlEscape(*a.SynopsisPersian)))
	}

	return b.String()
}

func htmlEscape(s string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(s)
}
