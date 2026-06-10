package fetchers

import (
	"fmt"
	"strings"

	"github.com/timhealey/world-cup-predictor/backend/internal/store"
)

type ContextResult struct {
	TournamentContext string
	TrackRecord       string
}

func FetchContext(s *store.Store, homeCode, awayCode string) (ContextResult, error) {
	matches, err := s.ListMatches()
	if err != nil {
		return ContextResult{}, err
	}
	var completed []store.Match
	for _, m := range matches {
		if m.HomeScore != nil && m.AwayScore != nil {
			completed = append(completed, m)
		}
	}

	tournament := buildTournamentContext(completed, homeCode, awayCode)
	track := buildTrackRecord(s, completed)
	return ContextResult{TournamentContext: tournament, TrackRecord: track}, nil
}

func buildTournamentContext(completed []store.Match, home, away string) string {
	if len(completed) == 0 {
		return "No completed matches yet — first matchday of the tournament."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Tournament context (as of %d completed matches):\n", len(completed))
	for _, m := range completed {
		if m.HomeTeamCode == home || m.AwayTeamCode == home ||
			m.HomeTeamCode == away || m.AwayTeamCode == away {
			fmt.Fprintf(&b, "  - %s: %s %d-%d %s\n",
				m.KickoffUTC, m.HomeTeamCode, derefInt(m.HomeScore),
				derefInt(m.AwayScore), m.AwayTeamCode)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func buildTrackRecord(s *store.Store, completed []store.Match) string {
	if len(completed) == 0 {
		return "No predictions with results yet."
	}
	correct, total := 0, 0
	for _, m := range completed {
		preds, err := s.ListPredictionsByMatch(m.ID)
		if err != nil || len(preds) == 0 {
			continue
		}
		p := preds[0] // latest
		total++
		actual := actualWinner(m)
		if p.PredictedWinner == actual {
			correct++
		}
	}
	if total == 0 {
		return "No predictions with results yet."
	}
	return fmt.Sprintf("Predictor track record: %d of %d winner predictions correct (%.0f%%).", correct, total, 100*float64(correct)/float64(total))
}

func actualWinner(m store.Match) string {
	h, a := derefInt(m.HomeScore), derefInt(m.AwayScore)
	if h > a {
		return m.HomeTeamCode
	} else if a > h {
		return m.AwayTeamCode
	}
	return "draw"
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
