package fdorg

import (
	"context"
	"encoding/json"
)

type Team struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	TLA   string `json:"tla"`
	Crest string `json:"crest"`
}

func (c *Client) GetTeams(ctx context.Context) ([]Team, error) {
	body, err := c.get(ctx, "/v4/competitions/WC/teams")
	if err != nil {
		return nil, err
	}
	var env struct {
		Teams []Team `json:"teams"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	return env.Teams, nil
}
