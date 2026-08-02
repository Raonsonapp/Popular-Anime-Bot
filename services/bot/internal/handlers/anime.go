package handlers

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"popular-anime-bot/bot/internal/apiclient"
	"popular-anime-bot/bot/internal/keyboards"
)

func (b *Bot) sendAnimeDetail(ctx context.Context, chatID, telegramUserID int64, animeID int64) {
	a, err := b.Client.GetAnime(ctx, animeID)
	if err != nil {
		b.Logger.Error("get anime", "error", err)
		b.reply(chatID, "❌ متأسفانه این آنیمه یافت نشد.", nil)
		return
	}

	_ = b.Client.RecordAnimeView(ctx, animeID)

	userID, _ := b.ensureUser(ctx, &tgbotapi.User{ID: telegramUserID})
	isFav := false
	if userID != 0 {
		favs, err := b.Client.ListFavorites(ctx, userID, 1)
		if err == nil {
			for _, f := range favs.Items {
				if f.ID == animeID {
					isFav = true
					break
				}
			}
		}
	}

	caption := formatAnimeCaption(*a)
	kb := keyboards.AnimeDetailKeyboard(*a, isFav, b.API.Self.UserName)

	if a.PosterStorageChatID != nil && a.PosterStorageMessageID != nil {
		copyMsg := tgbotapi.CopyMessageConfig{
			BaseChat:   tgbotapi.BaseChat{ChatID: chatID, ReplyMarkup: kb},
			FromChatID: *a.PosterStorageChatID,
			MessageID:  int(*a.PosterStorageMessageID),
			Caption:    caption,
			ParseMode:  tgbotapi.ModeHTML,
		}
		if _, err := b.API.Send(copyMsg); err == nil {
			return
		}
	} else if a.PosterURL != nil && *a.PosterURL != "" {
		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(*a.PosterURL))
		photo.Caption = caption
		photo.ParseMode = tgbotapi.ModeHTML
		photo.ReplyMarkup = kb
		if _, err := b.API.Send(photo); err == nil {
			return
		}
	}

	b.reply(chatID, caption, &kb)
}

func (b *Bot) sendEpisodesList(ctx context.Context, chatID int64, animeID int64, page int) {
	resp, err := b.Client.ListEpisodes(ctx, animeID, page)
	if err != nil {
		b.reply(chatID, "❌ خطا در دریافت لیست قسمت‌ها.", nil)
		return
	}
	if len(resp.Items) == 0 {
		b.reply(chatID, "هنوز قسمتی برای این آنیمه منتشر نشده است.", nil)
		return
	}
	kb := keyboards.EpisodesKeyboard(animeID, resp.Items, page, resp.Total, 10)
	b.reply(chatID, fmt.Sprintf("🎞 قسمت‌ها (صفحه %d):", page), &kb)
}

func (b *Bot) deliverEpisode(ctx context.Context, chatID, telegramUserID int64, episodeID int64) {
	e, err := b.Client.GetEpisode(ctx, episodeID)
	if err != nil {
		b.reply(chatID, "❌ این قسمت یافت نشد.", nil)
		return
	}

	copyMsg := tgbotapi.CopyMessageConfig{
		BaseChat:   tgbotapi.BaseChat{ChatID: chatID},
		FromChatID: e.StorageChatID,
		MessageID:  int(e.StorageMessageID),
	}
	if _, err := b.API.Send(copyMsg); err != nil {
		b.Logger.Error("deliver episode", "error", err, "episode_id", episodeID)
		b.reply(chatID, "❌ در ارسال فایل خطایی رخ داد. لطفاً بعداً دوباره تلاش کنید.", nil)
		return
	}

	if userID, err := b.ensureUser(ctx, &tgbotapi.User{ID: telegramUserID}); err == nil && userID != 0 {
		_ = b.Client.RecordProgress(ctx, userID, episodeID, 0, false)
	}
}

func (b *Bot) sendAnimeSearchResults(chatID int64, items []apiclient.Anime, total, page int, pagePrefix string, header string) {
	if len(items) == 0 {
		b.reply(chatID, "چیزی پیدا نشد. 🔎", nil)
		return
	}
	kb := keyboards.AnimeListKeyboard(items, page, total, 20, pagePrefix)
	b.reply(chatID, header, &kb)
}
