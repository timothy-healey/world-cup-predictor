package bootstrap

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/timhealey/world-cup-predictor/backend/internal/fdorg"
	"github.com/timhealey/world-cup-predictor/backend/internal/plist"
	"github.com/timhealey/world-cup-predictor/backend/internal/store"
)

// Run fetches teams + fixtures and writes them to the store. Idempotent.
// agentsDir is the LaunchAgents directory for per-match plists.
// workDir is the absolute path of the backend directory (where .env and
// wcp.db live); it is baked into each plist so launchd-spawned predictions
// can locate config + DB regardless of launchd's cwd resolution.
func Run(ctx context.Context, s *store.Store, c *fdorg.Client, agentsDir, workDir string) error {
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
		// football-data.org's /matches endpoint sometimes returns a TLA that
		// doesn't match the TLA returned by /teams (e.g. Curaçao: CUW vs CUR),
		// which would blow up the FK constraint on matches.home_team_code.
		// Resolve the canonical team codes via TLA first, falling back to the
		// numeric fixture_src_id; skip the match (warn, don't fail) if neither
		// resolution finds a team.
		homeCode, err := resolveTeamCode(s, m.HomeTLA, m.HomeID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[warn] bootstrap: skipping match %d: home team unresolved (%v)\n", m.ID, err)
			continue
		}
		awayCode, err := resolveTeamCode(s, m.AwayTLA, m.AwayID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[warn] bootstrap: skipping match %d: away team unresolved (%v)\n", m.ID, err)
			continue
		}

		id, err := matchID(m, homeCode, awayCode)
		if err != nil {
			return err
		}
		stage := normalizeStage(m.Stage)
		if err := s.UpsertMatch(store.Match{
			ID:           id,
			HomeTeamCode: homeCode,
			AwayTeamCode: awayCode,
			KickoffUTC:   m.UTCDate,
			Stage:        stage,
			Venue:        m.Venue,
			FixtureSrcID: fmt.Sprintf("%d", m.ID),
		}); err != nil {
			return fmt.Errorf("upsert match %s: %w", id, err)
		}
		// Backfill group_id on each team from the match data: fdorg's /teams
		// endpoint doesn't return groups, but each group-stage match carries
		// the group letter on `match.group`. Idempotent — last write wins,
		// but every group-stage match for a given team yields the same group.
		if stage == "group" {
			if g := normalizeGroupID(m.Group); g != "" {
				_ = s.UpdateTeamGroup(homeCode, g)
				_ = s.UpdateTeamGroup(awayCode, g)
			}
		}
		// Write per-match launchd plist for scheduled prediction at T-30.
		if agentsDir != "" {
			t, err := time.Parse(time.RFC3339, m.UTCDate)
			if err == nil {
				binPath, _ := os.Executable()
				path, err := plist.WriteAgent(agentsDir, binPath, id, workDir, t)
				if err == nil {
					// `launchctl load -w` fails if the label is already loaded,
					// which would leave launchd's in-memory schedule pointing at
					// the OLD plist on disk after re-running bootstrap. Unload
					// first; ignore the error since it's expected on first-ever
					// bootstrap when nothing is loaded yet.
					_ = plist.UnloadAgent(path) // ignore: fails when not previously loaded, which is normal
					_ = plist.LoadAgent(path)
				}
			}
		}
	}
	// Daily results agent: pulls finished scores + refreshes predictions.json
	// once per day. Same unload-then-load dance so re-running bootstrap picks
	// up any template changes.
	if agentsDir != "" {
		binPath, _ := os.Executable()
		path, err := plist.WriteResultsAgent(agentsDir, binPath, workDir)
		if err == nil {
			_ = plist.UnloadAgent(path)
			_ = plist.LoadAgent(path)
		}
	}
	return nil
}

// resolveTeamCode returns the canonical team.code for a match team, first
// trying the TLA (the common case) then falling back to the numeric
// football-data.org team ID stored in teams.fixture_src_id. Returns an error
// only when neither lookup finds a row.
func resolveTeamCode(s *store.Store, tla string, srcID int) (string, error) {
	if tla != "" {
		if _, err := s.GetTeam(tla); err == nil {
			return tla, nil
		}
	}
	if srcID > 0 {
		if t, err := s.GetTeamByFixtureSrcID(strconv.Itoa(srcID)); err == nil {
			return t.Code, nil
		}
	}
	return "", fmt.Errorf("no team for tla=%q srcID=%d", tla, srcID)
}

// matchID formats the canonical match ID using the resolved team codes
// (which may differ from m.HomeTLA / m.AwayTLA when the /matches endpoint
// returned a TLA that didn't match /teams — see resolveTeamCode).
func matchID(m fdorg.Match, homeCode, awayCode string) (string, error) {
	t, err := time.Parse(time.RFC3339, m.UTCDate)
	if err != nil {
		return "", fmt.Errorf("parse kickoff %q: %w", m.UTCDate, err)
	}
	return fmt.Sprintf("%s-%s-vs-%s", t.UTC().Format("2006-01-02"), homeCode, awayCode), nil
}

// normalizeGroupID extracts the bare group letter from the various shapes
// football-data.org returns on a match's `group` field. Real responses use
// "Group F"; older fixtures / future schema changes might use "GROUP_F" or
// already-normalized letters. Returns "" if input is empty.
func normalizeGroupID(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	upper := strings.ToUpper(s)
	upper = strings.TrimPrefix(upper, "GROUP_")
	upper = strings.TrimPrefix(upper, "GROUP ")
	return strings.TrimSpace(upper)
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
