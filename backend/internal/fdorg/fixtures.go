package fdorg

import (
	"context"
	"encoding/json"
	"strings"
)

type Match struct {
	ID      int    `json:"id"`
	UTCDate string `json:"utcDate"`
	Status  string `json:"status"`
	Stage   string `json:"stage"`
	Group   string `json:"group"`
	HomeTLA string `json:"-"`
	AwayTLA string `json:"-"`
	Venue   string `json:"venue"`

	HomeScore *int `json:"-"`
	AwayScore *int `json:"-"`
}

type rawMatch struct {
	ID       int    `json:"id"`
	UTCDate  string `json:"utcDate"`
	Status   string `json:"status"`
	Stage    string `json:"stage"`
	Group    string `json:"group"`
	HomeTeam struct {
		TLA string `json:"tla"`
	} `json:"homeTeam"`
	AwayTeam struct {
		TLA string `json:"tla"`
	} `json:"awayTeam"`
	Venue string `json:"venue"`
	Score struct {
		FullTime struct {
			Home *int `json:"home"`
			Away *int `json:"away"`
		} `json:"fullTime"`
	} `json:"score"`
}

func (c *Client) GetFixtures(ctx context.Context) ([]Match, error) {
	body, err := c.get(ctx, "/v4/competitions/WC/matches")
	if err != nil {
		return nil, err
	}
	return parseMatches(body)
}

func parseMatches(body []byte) ([]Match, error) {
	var env struct {
		Matches []rawMatch `json:"matches"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	out := make([]Match, 0, len(env.Matches))
	for _, m := range env.Matches {
		out = append(out, Match{
			ID:      m.ID,
			UTCDate: m.UTCDate,
			// Defensive normalization: football-data.org documents status as already uppercase, but we normalize defensively.
			Status:    strings.ToUpper(m.Status),
			Stage:     m.Stage,
			Group:     m.Group,
			HomeTLA:   m.HomeTeam.TLA,
			AwayTLA:   m.AwayTeam.TLA,
			Venue:     m.Venue,
			HomeScore: m.Score.FullTime.Home,
			AwayScore: m.Score.FullTime.Away,
		})
	}
	return out, nil
}
