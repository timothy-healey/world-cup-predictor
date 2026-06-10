package fdorg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/timhealey/world-cup-predictor/backend/internal/ratelimit"
	"github.com/timhealey/world-cup-predictor/backend/internal/trace"
)

// RateLimitInfo is the most recent rate-limit observation made by this
// client. RemainingMinute is parsed from X-Requests-Available-Minute on the
// last successful response; -1 means "no observation yet".
type RateLimitInfo struct {
	RemainingMinute int
	LastUpdated     time.Time
}

type Client struct {
	baseURL string
	apiKey  string
	httpc   *http.Client

	// sleeper is overridable in tests so we don't actually pause when the
	// per-minute quota is exhausted or on 429 retry. Defaults to time.Sleep.
	sleeper func(time.Duration)
	// logger is the writer warnings are emitted to (defaults to os.Stderr
	// inside warn helpers); only the package-level warn function uses it.

	rlMu sync.Mutex
	rl   RateLimitInfo
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpc:   &http.Client{Timeout: 15 * time.Second},
		sleeper: time.Sleep,
		rl:      RateLimitInfo{RemainingMinute: -1},
	}
}

// LastRateLimit returns the latest observation. Callers should treat
// RemainingMinute == -1 / LastUpdated.IsZero() as "no observation yet".
func (c *Client) LastRateLimit() RateLimitInfo {
	c.rlMu.Lock()
	defer c.rlMu.Unlock()
	return c.rl
}

func (c *Client) setRateLimit(remaining int) {
	c.rlMu.Lock()
	c.rl = RateLimitInfo{RemainingMinute: remaining, LastUpdated: time.Now()}
	c.rlMu.Unlock()
	ratelimit.RecordFDOrg(remaining)
}

// get fetches the given path. On a 429 it parses Retry-After and retries
// once after sleeping. After any successful response it inspects
// X-Requests-Available-Minute and, if 0 or 1, sleeps 60s to stay under the
// per-minute cap conservatively.
func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	body, status, headers, err := c.doRequest(ctx, path)
	if err != nil {
		return nil, err
	}

	if status == http.StatusTooManyRequests {
		retry := parseRetryAfter(headers.Get("Retry-After"))
		warn("fdorg: rate limited, sleeping %ds before retry", int(retry/time.Second))
		c.sleeper(retry)
		body, status, headers, err = c.doRequest(ctx, path)
		if err != nil {
			return nil, err
		}
	}

	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("fdorg %s: %d %s", path, status, string(body))
	}

	// Successful response — record and possibly throttle.
	if v := headers.Get("X-Requests-Available-Minute"); v != "" {
		if n, perr := strconv.Atoi(v); perr == nil {
			c.setRateLimit(n)
			if n <= 1 {
				warn("fdorg: %d req/min remaining, sleeping 60s", n)
				c.sleeper(60 * time.Second)
			}
		}
	}
	return body, nil
}

// doRequest performs a single HTTP GET and returns body + status + headers.
// Body is fully read so the caller does not need to manage the response.
func (c *Client) doRequest(ctx context.Context, path string) ([]byte, int, http.Header, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, nil, err
	}
	req.Header.Set("X-Auth-Token", c.apiKey)

	trace.HTTPStart("fdorg", "GET", url)
	start := time.Now()
	resp, err := c.httpc.Do(req)
	if err != nil {
		trace.HTTPError("fdorg", time.Since(start), err)
		return nil, 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	dur := time.Since(start)
	if err != nil {
		trace.HTTPError("fdorg", dur, err)
		return nil, resp.StatusCode, resp.Header, err
	}
	trace.HTTPEnd("fdorg", resp.StatusCode, dur, len(body))
	return body, resp.StatusCode, resp.Header, nil
}

// parseRetryAfter interprets the Retry-After header value as a duration in
// seconds. Falls back to 1s if missing/unparseable so we still pause briefly
// instead of hot-looping.
func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return time.Second
	}
	if n, err := strconv.Atoi(v); err == nil && n >= 0 {
		return time.Duration(n) * time.Second
	}
	return time.Second
}
