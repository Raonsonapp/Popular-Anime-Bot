package publisher

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"popular-anime-bot/scheduler/internal/client"
	"popular-anime-bot/scheduler/internal/config"
)

var statusLabels = map[string]string{
	"ongoing":   "🟢 در حال پخش",
	"completed": "✅ تکمیل‌شده",
	"announced": "📢 اعلام‌شده",
}

func formatCaption(a client.Anime) string {
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
	if a.DurationMinutes != nil {
		fmt.Fprintf(&b, "⏱ مدت هر قسمت: %d دقیقه\n", *a.DurationMinutes)
	}

	if a.SynopsisPersian != nil && *a.SynopsisPersian != "" {
		summary := *a.SynopsisPersian
		if len([]rune(summary)) > 400 {
			summary = string([]rune(summary)[:400]) + "…"
		}
		fmt.Fprintf(&b, "\n📝 %s\n", htmlEscape(summary))
	}

	return b.String()
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

func buildKeyboard(a client.Anime, botUsername string) tgbotapi.InlineKeyboardMarkup {
	watchURL := fmt.Sprintf("https://t.me/%s?start=anime_%d", botUsername, a.ID)
	shareURL := fmt.Sprintf("https://t.me/share/url?url=%s", watchURL)

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("▶️ تماشا در ربات", watchURL),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⭐ علاقه‌مندی", fmt.Sprintf("a:%d:fav", a.ID)),
			tgbotapi.NewInlineKeyboardButtonURL("📢 اشتراک‌گذاری", shareURL),
		),
	)
}

// Run publishes up to cfg.PostsPerRun not-yet-posted anime to the showcase channel.
func Run(ctx context.Context, cfg config.Config, api *tgbotapi.BotAPI, cl *client.Client, logger *slog.Logger) {
	if cfg.TargetChannelID == 0 {
		logger.Warn("TARGET_CHANNEL_ID not set, skipping publish run")
		return
	}

	pending, err := cl.PendingAnime(ctx, cfg.TargetChannelID, cfg.PostsPerRun)
	if err != nil {
		logger.Error("fetch pending anime", "error", err)
		return
	}
	if len(pending) == 0 {
		logger.Info("no pending anime to publish")
		return
	}

	for _, a := range pending {
		caption := formatCaption(a)
		kb := buildKeyboard(a, cfg.BotUsername)

		var messageID int
		if a.PosterStorageChatID != nil && a.PosterStorageMessageID != nil {
			copyMsg := tgbotapi.CopyMessageConfig{
				BaseChat:   tgbotapi.BaseChat{ChatID: cfg.TargetChannelID, ReplyMarkup: kb},
				FromChatID: *a.PosterStorageChatID,
				MessageID:  int(*a.PosterStorageMessageID),
				Caption:    caption,
				ParseMode:  tgbotapi.ModeHTML,
			}
			res, err := api.Send(copyMsg)
			if err != nil {
				logger.Error("publish anime (copy poster)", "anime_id", a.ID, "error", err)
				continue
			}
			messageID = res.MessageID
		} else {
			var res tgbotapi.Message
			var sendErr error
			if a.PosterURL != nil && *a.PosterURL != "" {
				photo := tgbotapi.NewPhoto(cfg.TargetChannelID, tgbotapi.FileURL(*a.PosterURL))
				photo.Caption = caption
				photo.ParseMode = tgbotapi.ModeHTML
				photo.ReplyMarkup = kb
				res, sendErr = api.Send(photo)
			} else {
				msg := tgbotapi.NewMessage(cfg.TargetChannelID, caption)
				msg.ParseMode = tgbotapi.ModeHTML
				msg.ReplyMarkup = kb
				res, sendErr = api.Send(msg)
			}
			if sendErr != nil {
				logger.Error("publish anime", "anime_id", a.ID, "error", sendErr)
				continue
			}
			messageID = res.MessageID
		}

		if err := cl.RecordPost(ctx, a.ID, cfg.TargetChannelID, int64(messageID)); err != nil {
			logger.Error("record post", "anime_id", a.ID, "error", err)
			continue
		}
		logger.Info("published anime", "anime_id", a.ID, "title", a.Title)
	}
}
