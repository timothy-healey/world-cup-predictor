package fdorg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	httpc   *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpc:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", c.apiKey)
	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fdorg %s: %d %s", path, resp.StatusCode, string(body))
	}
	return io.ReadAll(resp.Body)
}
