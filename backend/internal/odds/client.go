package odds

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/timhealey/world-cup-predictor/backend/internal/ratelimit"
	"github.com/timhealey/world-cup-predictor/backend/internal/trace"
)

// OddsRateLimit is the most recent rate-limit observation from the-odds-api.
// -1 fields mean "no observation yet".
type OddsRateLimit struct {
	Remaining   int
	Used        int
	LastCost    int
	LastUpdated time.Time
}

// LowBudgetThreshold is the cutoff below which a warning is emitted when
// requests-remaining drops; the-odds-api free tier is 500/month so 50 is a
// roughly 10% reserve.
const LowBudgetThreshold = 50

type Client struct {
	baseURL string
	apiKey  string
	httpc   *http.Client

	rlMu sync.Mutex
	rl   OddsRateLimit
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpc:   &http.Client{Timeout: 15 * time.Second},
		rl:      OddsRateLimit{Remaining: -1, Used: -1, LastCost: -1},
	}
}

// LastRateLimit returns the most recent observation. Treat any -1 field /
// zero LastUpdated as "no observation yet".
func (c *Client) LastRateLimit() OddsRateLimit {
	c.rlMu.Lock()
	defer c.rlMu.Unlock()
	return c.rl
}

func (c *Client) setRateLimit(remaining, used, lastCost int) {
	c.rlMu.Lock()
	c.rl = OddsRateLimit{
		Remaining:   remaining,
		Used:        used,
		LastCost:    lastCost,
		LastUpdated: time.Now(),
	}
	c.rlMu.Unlock()
	ratelimit.RecordOdds(remaining, used, lastCost)
}

type Odds struct {
	Bookmaker       string
	HomeOdds        float64
	AwayOdds        float64
	DrawOdds        float64
	HomeImpliedProb float64
	AwayImpliedProb float64
	DrawImpliedProb float64
}

type rawEvent struct {
	ID         string `json:"id"`
	HomeTeam   string `json:"home_team"`
	AwayTeam   string `json:"away_team"`
	Commence   string `json:"commence_time"`
	Bookmakers []struct {
		Key     string `json:"key"`
		Markets []struct {
			Key      string `json:"key"`
			Outcomes []struct {
				Name  string  `json:"name"`
				Price float64 `json:"price"`
			} `json:"outcomes"`
		} `json:"markets"`
	} `json:"bookmakers"`
}

func (c *Client) GetForMatch(ctx context.Context, homeName, awayName, kickoffUTC string) (Odds, error) {
	q := url.Values{}
	q.Set("apiKey", c.apiKey)
	q.Set("regions", "uk,us,eu,au")
	q.Set("markets", "h2h")
	q.Set("dateFormat", "iso")
	u := fmt.Sprintf("%s/v4/sports/soccer_fifa_world_cup/odds/?%s", c.baseURL, q.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return Odds{}, err
	}

	trace.HTTPStart("odds", "GET", u)
	start := time.Now()
	resp, err := c.httpc.Do(req)
	if err != nil {
		trace.HTTPError("odds", time.Since(start), err)
		return Odds{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	trace.HTTPEnd("odds", resp.StatusCode, time.Since(start), len(body))

	// Capture rate-limit headers regardless of status (the-odds-api sets
	// them on errors too, and we want visibility into "ran out of budget"
	// scenarios). resp.Header.Get is case-insensitive so the documented
	// lowercase keys work as-is.
	c.captureRateLimit(resp.Header)

	if resp.StatusCode != 200 {
		return Odds{}, fmt.Errorf("odds api %d: %s", resp.StatusCode, string(body))
	}

	var events []rawEvent
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&events); err != nil {
		return Odds{}, err
	}
	for _, e := range events {
		if e.HomeTeam == homeName && e.AwayTeam == awayName {
			return pickFirstH2H(e)
		}
	}
	return Odds{}, fmt.Errorf("no odds found for %s vs %s @ %s", homeName, awayName, kickoffUTC)
}

// captureRateLimit reads the three documented headers, stores them, and
// emits a stderr warning if the remaining budget has crossed below the
// LowBudgetThreshold.
func (c *Client) captureRateLimit(h http.Header) {
	remaining := parseIntHeader(h, "x-requests-remaining")
	used := parseIntHeader(h, "x-requests-used")
	lastCost := parseIntHeader(h, "x-requests-last")
	// Only record if at least one header was present (avoid blowing away a
	// real observation with -1 noise on a malformed response).
	if remaining == -1 && used == -1 && lastCost == -1 {
		return
	}
	c.setRateLimit(remaining, used, lastCost)
	if remaining != -1 && remaining < LowBudgetThreshold {
		warn("the-odds-api: only %d requests left this period (used %d, last call cost %d)",
			remaining, used, lastCost)
	}
}

func parseIntHeader(h http.Header, key string) int {
	v := h.Get(key)
	if v == "" {
		return -1
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return -1
	}
	return n
}

func pickFirstH2H(e rawEvent) (Odds, error) {
	for _, bk := range e.Bookmakers {
		for _, m := range bk.Markets {
			if m.Key != "h2h" {
				continue
			}
			o := Odds{Bookmaker: bk.Key}
			for _, oc := range m.Outcomes {
				switch oc.Name {
				case e.HomeTeam:
					o.HomeOdds = oc.Price
				case e.AwayTeam:
					o.AwayOdds = oc.Price
				case "Draw":
					o.DrawOdds = oc.Price
				}
			}
			if o.HomeOdds > 0 {
				o.HomeImpliedProb = 1.0 / o.HomeOdds
			}
			if o.AwayOdds > 0 {
				o.AwayImpliedProb = 1.0 / o.AwayOdds
			}
			if o.DrawOdds > 0 {
				o.DrawImpliedProb = 1.0 / o.DrawOdds
			}
			return o, nil
		}
	}
	return Odds{}, fmt.Errorf("no h2h market for %s vs %s", e.HomeTeam, e.AwayTeam)
}
