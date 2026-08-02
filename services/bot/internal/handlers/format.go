package handlers

import (
	"fmt"
	"strings"

	"popular-anime-bot/bot/internal/apiclient"
)

var statusLabels = map[string]string{
	"ongoing":   "🟢 در حال پخش",
	"completed": "✅ تکمیل‌شده",
	"announced": "📢 اعلام‌شده",
}

func formatAnimeCaption(a apiclient.Anime) string {
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
		fmt.Fprintf(&b, "🎭 ژانر: %s\n", strings.Join(names, "، "))
	}
	if a.StudioName != "" {
		fmt.Fprintf(&b, "🏢 استودیو: %s\n", htmlEscape(a.StudioName))
	}
	if a.Year != nil {
		fmt.Fprintf(&b, "📅 سال: %d\n", *a.Year)
	}
	if label, ok := statusLabels[a.Status]; ok {
		fmt.Fprintf(&b, "📺 وضعیت: %s\n", label)
	}
	fmt.Fprintf(&b, "⭐ امتیاز: %.1f\n", a.RatingScore)
	fmt.Fprintf(&b, "🎞 تعداد قسمت‌ها: %d\n", a.EpisodesCount)

	if a.SynopsisPersian != nil && *a.SynopsisPersian != "" {
		fmt.Fprintf(&b, "\n📝 خلاصه داستان:\n%s\n", htmlEscape(*a.SynopsisPersian))
	}

	return b.String()
}

func htmlEscape(s string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(s)
}
