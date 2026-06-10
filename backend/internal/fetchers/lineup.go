package fetchers

import (
	"context"
	"fmt"
)

type LineupResult struct {
	Confirmed bool     `json:"confirmed"`
	HomeXI    []string `json:"home_xi"`
	AwayXI    []string `json:"away_xi"`
	Notes     string   `json:"notes"`
}

// FetchLineup runs `claude -p` with a lineup-specific prompt and parses
// the JSON response.
func FetchLineup(ctx context.Context, d claudeBin, home, away string) (LineupResult, error) {
	prompt := fmt.Sprintf(`You are scouting a football match: %s vs %s.

Search the web for the most recent confirmed starting XI for both teams (typically
posted ~60 minutes before kickoff). If only the 26-player squad is available, set
confirmed=false.

Reply with ONLY this JSON, no prose:
{
  "confirmed": true | false,
  "home_xi": ["player", ...],
  "away_xi": ["player", ...],
  "notes": "short reason / source"
}`, home, away)
	return runJSON[LineupResult](ctx, d, prompt)
}
