package handlers

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"popular-anime-bot/bot/internal/apiclient"
	"popular-anime-bot/bot/internal/i18n"
	"popular-anime-bot/bot/internal/keyboards"
)

func (b *Bot) sendAnimeDetail(ctx context.Context, chatID, telegramUserID int64, animeID int64) {
	userID, lang, _ := b.ensureUser(ctx, &tgbotapi.User{ID: telegramUserID})

	a, err := b.Client.GetAnime(ctx, animeID)
	if err != nil {
		b.Logger.Error("get anime", "error", err)
		b.reply(chatID, i18n.T(lang, "anime_not_found"), nil)
		return
	}

	_ = b.Client.RecordAnimeView(ctx, animeID)

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

	caption := formatAnimeCaption(lang, *a)
	kb := keyboards.AnimeDetailKeyboard(lang, *a, isFav, b.API.Self.UserName)

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

// sendWatchEntry is what the anime detail screen's "Watch" button leads
// to: if the anime has more than one season on record, the user picks
// one first, since seasons each restart at episode 1 and flattening them
// into a single list produces several identical-looking "Episode 1"
// buttons with no way to tell them apart. A single season (or no season
// data at all, e.g. a plain single-cour show) skips straight to the
// episode list, matching the old direct-to-episodes behavior.
func (b *Bot) sendWatchEntry(ctx context.Context, chatID int64, lang i18n.Lang, animeID int64) {
	seasons, err := b.Client.ListSeasons(ctx, animeID)
	if err != nil {
		b.Logger.Error("list seasons", "error", err)
		seasons = nil
	}

	if len(seasons) <= 1 {
		var seasonID int64
		if len(seasons) == 1 {
			seasonID = seasons[0].ID
		}
		b.sendEpisodesList(ctx, chatID, lang, animeID, seasonID, 1)
		return
	}

	kb := keyboards.SeasonsKeyboard(lang, animeID, seasons)
	b.reply(chatID, i18n.T(lang, "choose_season"), &kb)
}

func (b *Bot) sendEpisodesList(ctx context.Context, chatID int64, lang i18n.Lang, animeID, seasonID int64, page int) {
	resp, err := b.Client.ListEpisodes(ctx, animeID, seasonID, page)
	if err != nil {
		b.reply(chatID, i18n.T(lang, "episodes_error"), nil)
		return
	}
	if len(resp.Items) == 0 {
		b.reply(chatID, i18n.T(lang, "no_episodes"), nil)
		return
	}
	kb := keyboards.EpisodesKeyboard(lang, animeID, seasonID, resp.Items, page, resp.Total, 10)
	b.reply(chatID, i18n.Tf(lang, "episodes_header", page), &kb)
}

func (b *Bot) deliverEpisode(ctx context.Context, chatID, telegramUserID int64, episodeID int64) {
	userID, lang, _ := b.ensureUser(ctx, &tgbotapi.User{ID: telegramUserID})

	e, err := b.Client.GetEpisode(ctx, episodeID)
	if err != nil {
		b.reply(chatID, i18n.T(lang, "episode_not_found"), nil)
		return
	}

	copyMsg := tgbotapi.CopyMessageConfig{
		BaseChat:   tgbotapi.BaseChat{ChatID: chatID},
		FromChatID: e.StorageChatID,
		MessageID:  int(e.StorageMessageID),
	}
	if _, err := b.API.Send(copyMsg); err != nil {
		b.Logger.Error("deliver episode", "error", err, "episode_id", episodeID)
		b.reply(chatID, i18n.T(lang, "deliver_error"), nil)
		return
	}

	if userID != 0 {
		_ = b.Client.RecordProgress(ctx, userID, episodeID, 0, false)
	}
}

func (b *Bot) sendAnimeSearchResults(chatID int64, lang i18n.Lang, items []apiclient.Anime, total, page int, pagePrefix string, header string) {
	if len(items) == 0 {
		b.reply(chatID, i18n.T(lang, "nothing_found"), nil)
		return
	}
	kb := keyboards.AnimeListKeyboard(lang, items, page, total, 20, pagePrefix)
	b.reply(chatID, header, &kb)
}
