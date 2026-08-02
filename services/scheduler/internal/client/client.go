package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Genre struct {
	NamePersian string `json:"name_persian"`
}

type Anime struct {
	ID                     int64   `json:"id"`
	Title                  string  `json:"title_persian"`
	TitleEnglish           *string `json:"title_english"`
	TitleJapanese          *string `json:"title_japanese"`
	SynopsisPersian        *string `json:"synopsis_persian"`
	Kind                   string  `json:"kind"`
	Status                 string  `json:"status"`
	Year                   *int    `json:"year"`
	StudioName             string  `json:"studio_name"`
	PosterStorageChatID    *int64  `json:"poster_storage_chat_id"`
	PosterStorageMessageID *int64  `json:"poster_storage_message_id"`
	PosterURL              *string `json:"poster_url"`
	EpisodesCount          int     `json:"episodes_count"`
	DurationMinutes        *int    `json:"duration_minutes"`
	RatingScore            float64 `json:"rating_score"`
	Genres                 []Genre `json:"genres"`
}

type Client struct {
	baseURL     string
	internalKey string
	http        *http.Client
}

func New(baseURL, internalKey string) *Client {
	return &Client{baseURL: baseURL, internalKey: internalKey, http: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body interface{}, out interface{}) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}

	u := c.baseURL + path
	if query != nil {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Key", c.internalKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("api error %d on %s %s: %s", resp.StatusCode, method, path, string(data))
	}

	if out != nil && resp.StatusCode != http.StatusNoContent {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (c *Client) PendingAnime(ctx context.Context, targetChannelID int64, limit int) ([]Anime, error) {
	q := url.Values{}
	q.Set("target_channel_id", fmt.Sprint(targetChannelID))
	q.Set("limit", fmt.Sprint(limit))
	var out []Anime
	if err := c.do(ctx, http.MethodGet, "/api/v1/publish/pending", q, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) RecordPost(ctx context.Context, animeID, targetChannelID, telegramMessageID int64) error {
	body := map[string]interface{}{
		"anime_id": animeID, "target_channel_id": targetChannelID, "telegram_message_id": telegramMessageID,
	}
	return c.do(ctx, http.MethodPost, "/api/v1/publish/record", nil, body, nil)
}

func (c *Client) RefreshPopularity(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/api/v1/publish/refresh-popularity", nil, nil, nil)
}
