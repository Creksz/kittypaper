package wallhaven

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"kittypaper/internal/domain/online"
)

const baseURL = "https://wallhaven.cc/api/v1"

type Client struct {
	APIKey string
	HTTP   *http.Client
}

type searchResponse struct {
	Data []struct {
		ID         string `json:"id"`
		URL        string `json:"url"`
		Resolution string `json:"resolution"`
		Path       string `json:"path"`
		Thumbs     struct {
			Small string `json:"small"`
			Large string `json:"large"`
		} `json:"thumbs"`
	} `json:"data"`
	Meta struct {
		CurrentPage int `json:"current_page"`
		LastPage    int `json:"last_page"`
		Total       int `json:"total"`
	} `json:"meta"`
}

func (c Client) client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second}
}

func (c Client) Search(ctx context.Context, query string, page int) (online.SearchResult, error) {
	if page < 1 {
		page = 1
	}
	values := url.Values{}
	values.Set("page", strconv.Itoa(page))
	query = strings.TrimSpace(query)
	if query != "" {
		values.Set("q", query)
	}
	if c.APIKey != "" {
		values.Set("api_key", c.APIKey)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/search?"+values.Encode(), nil)
	if err != nil {
		return online.SearchResult{}, err
	}
	req.Header.Set("User-Agent", "kittypaper/0.1")

	resp, err := c.client().Do(req)
	if err != nil {
		return online.SearchResult{}, fmt.Errorf("wallhaven search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return online.SearchResult{}, fmt.Errorf("wallhaven search: HTTP %d", resp.StatusCode)
	}

	var payload searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return online.SearchResult{}, fmt.Errorf("wallhaven search decode: %w", err)
	}

	result := online.SearchResult{
		Page:     payload.Meta.CurrentPage,
		LastPage: payload.Meta.LastPage,
		Total:    payload.Meta.Total,
	}
	for _, item := range payload.Data {
		thumb := item.Thumbs.Large
		if thumb == "" {
			thumb = item.Thumbs.Small
		}
		result.Items = append(result.Items, online.Item{
			ID:         item.ID,
			PageURL:    item.URL,
			ThumbURL:   thumb,
			ImageURL:   item.Path,
			Resolution: item.Resolution,
			Source:     "Wallhaven",
		})
	}
	return result, nil
}
