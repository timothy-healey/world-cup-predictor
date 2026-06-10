package bootstrap

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/timhealey/world-cup-predictor/backend/internal/fdorg"
	"github.com/timhealey/world-cup-predictor/backend/internal/plist"
	"github.com/timhealey/world-cup-predictor/backend/internal/store"
)

// Run fetches teams + fixtures and writes them to the store. Idempotent.
// agentsDir is reserved for plist writing in the next task.
func Run(ctx context.Context, s *store.Store, c *fdorg.Client, agentsDir string) error {
	teams, err := c.GetTeams(ctx)
	if err != nil {
		return fmt.Errorf("get teams: %w", err)
	}
	for _, t := range teams {
		if err := s.UpsertTeam(store.Team{
			Code:         t.TLA,
			Name:         t.Name,
			FlagURL:      t.Crest,
			FixtureSrcID: fmt.Sprintf("%d", t.ID),
		}); err != nil {
			return fmt.Errorf("upsert team %s: %w", t.TLA, err)
		}
	}

	fixtures, err := c.GetFixtures(ctx)
	if err != nil {
		return fmt.Errorf("get fixtures: %w", err)
	}
	for _, m := range fixtures {
		id, err := matchID(m)
		if err != nil {
			return err
		}
		stage := normalizeStage(m.Stage)
		if err := s.UpsertMatch(store.Match{
			ID:           id,
			HomeTeamCode: m.HomeTLA,
			AwayTeamCode: m.AwayTLA,
			KickoffUTC:   m.UTCDate,
			Stage:        stage,
			Venue:        m.Venue,
			FixtureSrcID: fmt.Sprintf("%d", m.ID),
		}); err != nil {
			return fmt.Errorf("upsert match %s: %w", id, err)
		}
		// Write per-match launchd plist for scheduled prediction at T-30.
		if agentsDir != "" {
			t, err := time.Parse(time.RFC3339, m.UTCDate)
			if err == nil {
				binPath, _ := os.Executable()
				path, err := plist.WriteAgent(agentsDir, binPath, id, t)
				if err == nil {
					_ = plist.LoadAgent(path) // best-effort; no-op on non-macOS
				}
			}
		}
	}
	return nil
}

func matchID(m fdorg.Match) (string, error) {
	t, err := time.Parse(time.RFC3339, m.UTCDate)
	if err != nil {
		return "", fmt.Errorf("parse kickoff %q: %w", m.UTCDate, err)
	}
	return fmt.Sprintf("%s-%s-vs-%s", t.UTC().Format("2006-01-02"), m.HomeTLA, m.AwayTLA), nil
}

func normalizeStage(s string) string {
	switch strings.ToUpper(s) {
	case "GROUP_STAGE":
		return "group"
	case "LAST_32", "ROUND_OF_32":
		return "round-of-32"
	case "LAST_16", "ROUND_OF_16":
		return "round-of-16"
	case "QUARTER_FINALS":
		return "qf"
	case "SEMI_FINALS":
		return "sf"
	case "THIRD_PLACE":
		return "third-place"
	case "FINAL":
		return "final"
	default:
		return strings.ToLower(strings.ReplaceAll(s, "_", "-"))
	}
}
