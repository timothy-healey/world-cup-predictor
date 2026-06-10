package odds

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	httpc   *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{baseURL: baseURL, apiKey: apiKey, httpc: &http.Client{Timeout: 15 * time.Second}}
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
	resp, err := c.httpc.Do(req)
	if err != nil {
		return Odds{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return Odds{}, fmt.Errorf("odds api %d: %s", resp.StatusCode, string(body))
	}

	var events []rawEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return Odds{}, err
	}
	for _, e := range events {
		if e.HomeTeam == homeName && e.AwayTeam == awayName {
			return pickFirstH2H(e)
		}
	}
	return Odds{}, fmt.Errorf("no odds found for %s vs %s @ %s", homeName, awayName, kickoffUTC)
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
