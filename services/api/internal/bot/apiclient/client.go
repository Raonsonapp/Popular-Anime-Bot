package apiclient

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

type Client struct {
	baseURL     string
	internalKey string
	http        *http.Client
}

func New(baseURL, internalKey string) *Client {
	return &Client{
		baseURL:     baseURL,
		internalKey: internalKey,
		http:        &http.Client{Timeout: 10 * time.Second},
	}
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
	if c.internalKey != "" {
		req.Header.Set("X-Internal-Key", c.internalKey)
	}

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

func (c *Client) ListAnime(ctx context.Context, query url.Values) (*ListResponse, error) {
	var out ListResponse
	if err := c.do(ctx, http.MethodGet, "/api/v1/anime", query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RandomAnime(ctx context.Context) (*Anime, error) {
	var out Anime
	if err := c.do(ctx, http.MethodGet, "/api/v1/anime/random", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetAnime(ctx context.Context, id int64) (*Anime, error) {
	var out Anime
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/anime/%d", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RecordAnimeView(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/anime/%d/view", id), nil, nil, nil)
}

func (c *Client) ListEpisodes(ctx context.Context, animeID int64, page int) (*EpisodeListResponse, error) {
	q := url.Values{}
	q.Set("page", fmt.Sprint(page))
	q.Set("page_size", "10")
	var out EpisodeListResponse
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/anime/%d/episodes", animeID), q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetEpisode(ctx context.Context, id int64) (*Episode, error) {
	var out Episode
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/episodes/%d", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListGenres(ctx context.Context) ([]Genre, error) {
	var out []Genre
	if err := c.do(ctx, http.MethodGet, "/api/v1/genres", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) TouchUser(ctx context.Context, telegramUserID int64, username, firstName, lang string) (*User, error) {
	body := map[string]interface{}{
		"telegram_user_id": telegramUserID,
		"username":         username,
		"first_name":       firstName,
		"language":         lang,
	}
	var out User
	if err := c.do(ctx, http.MethodPost, "/api/v1/users", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ToggleFavorite(ctx context.Context, userID, animeID int64) (bool, error) {
	body := map[string]interface{}{"user_id": userID, "anime_id": animeID}
	var out struct {
		Added bool `json:"added"`
	}
	if err := c.do(ctx, http.MethodPost, "/api/v1/favorites/toggle", nil, body, &out); err != nil {
		return false, err
	}
	return out.Added, nil
}

func (c *Client) ListFavorites(ctx context.Context, userID int64, page int) (*ListResponse, error) {
	q := url.Values{}
	q.Set("page", fmt.Sprint(page))
	var out ListResponse
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/users/%d/favorites", userID), q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RecordProgress(ctx context.Context, userID, episodeID int64, progressSeconds int, completed bool) error {
	body := map[string]interface{}{
		"user_id": userID, "episode_id": episodeID,
		"progress_seconds": progressSeconds, "completed": completed,
	}
	return c.do(ctx, http.MethodPost, "/api/v1/history", nil, body, nil)
}

func (c *Client) ContinueWatching(ctx context.Context, userID int64) ([]WatchHistoryEntry, error) {
	var out []WatchHistoryEntry
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/users/%d/continue-watching", userID), nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
