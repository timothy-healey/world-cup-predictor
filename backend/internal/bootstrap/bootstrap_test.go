package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/timhealey/world-cup-predictor/backend/internal/fdorg"
	"github.com/timhealey/world-cup-predictor/backend/internal/store"

	"github.com/stretchr/testify/require"
)

func TestRunPopulatesTeamsAndMatches(t *testing.T) {
	teamsBody, _ := os.ReadFile("../../testdata/fdorg-teams.json")
	matchesBody, _ := os.ReadFile("../../testdata/fdorg-matches.json")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if r.URL.Path == "/v4/competitions/WC/teams" {
			w.Write(teamsBody)
			return
		}
		w.Write(matchesBody)
	}))
	defer srv.Close()

	s, _ := store.Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()
	c := fdorg.NewClient(srv.URL, "k")

	agentsDir := t.TempDir()
	workDir := t.TempDir()
	require.NoError(t, Run(context.Background(), s, c, agentsDir, workDir))

	teams, _ := s.ListTeams()
	require.Len(t, teams, 2)
	matches, _ := s.ListMatches()
	require.Len(t, matches, 1)
	require.Equal(t, "2026-06-25-ARG-vs-SAU", matches[0].ID)

	// Each team that played a group-stage match should have its group_id
	// populated (derived from the match's `group` field, normalized to the
	// bare letter — fdorg returns e.g. "Group F").
	byCode := map[string]string{}
	for _, t := range teams {
		byCode[t.Code] = t.GroupID
	}
	require.Equal(t, "F", byCode["ARG"], "ARG should have group_id from match")
	require.Equal(t, "F", byCode["SAU"], "SAU should have group_id from match")

	// Second run should be a no-op
	require.NoError(t, Run(context.Background(), s, c, t.TempDir(), workDir))
	teamsAgain, _ := s.ListTeams()
	require.Len(t, teamsAgain, 2)
	matchesAgain, _ := s.ListMatches()
	require.Len(t, matchesAgain, 1)

	// Plist file should exist for the single match
	entries, err := os.ReadDir(agentsDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Contains(t, entries[0].Name(), "2026-06-25-ARG-vs-SAU")
}

// TestRunHandlesUnknownTeams reproduces the football-data.org quirk where
// the /matches endpoint returns a TLA that doesn't match any TLA from
// /teams (e.g. Curaçao: CUW from /teams, CUR from /matches), AND a second
// match where neither the TLA nor the numeric ID resolve to any known team.
// The first match should be salvaged via the fixture_src_id fallback; the
// second should be skipped with a warning. Bootstrap must return nil.
func TestRunHandlesUnknownTeams(t *testing.T) {
	teamsBody := []byte(`{
		"teams": [
			{"id": 760, "name": "Argentina", "tla": "ARG", "crest": ""},
			{"id": 802, "name": "Saudi Arabia", "tla": "SAU", "crest": ""},
			{"id": 1538, "name": "Curaçao", "tla": "CUW", "crest": ""}
		]
	}`)
	// Three matches:
	//  1. Normal ARG vs SAU — resolves via TLA.
	//  2. Curaçao quirk: /matches returns "CUR" but team is stored under "CUW"
	//     — must resolve via fixture_src_id=1538.
	//  3. Totally unknown team — must be skipped, not fatal.
	matchesBody := []byte(`{
		"matches": [
			{
				"id": 1,
				"utcDate": "2026-06-25T11:00:00Z",
				"status": "SCHEDULED",
				"stage": "GROUP_STAGE",
				"homeTeam": {"id": 760, "tla": "ARG"},
				"awayTeam": {"id": 802, "tla": "SAU"},
				"venue": "MetLife"
			},
			{
				"id": 2,
				"utcDate": "2026-06-26T11:00:00Z",
				"status": "SCHEDULED",
				"stage": "GROUP_STAGE",
				"homeTeam": {"id": 1538, "tla": "CUR"},
				"awayTeam": {"id": 802, "tla": "SAU"},
				"venue": "MetLife"
			},
			{
				"id": 3,
				"utcDate": "2026-06-27T11:00:00Z",
				"status": "SCHEDULED",
				"stage": "GROUP_STAGE",
				"homeTeam": {"id": 9999, "tla": "ZZZ"},
				"awayTeam": {"id": 802, "tla": "SAU"},
				"venue": "MetLife"
			}
		]
	}`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if r.URL.Path == "/v4/competitions/WC/teams" {
			w.Write(teamsBody)
			return
		}
		w.Write(matchesBody)
	}))
	defer srv.Close()

	s, _ := store.Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()
	c := fdorg.NewClient(srv.URL, "k")

	// agentsDir empty: we don't care about plists in this test, and writing
	// them would try to LoadAgent via launchctl in non-test environments.
	require.NoError(t, Run(context.Background(), s, c, "", t.TempDir()))

	matches, err := s.ListMatches()
	require.NoError(t, err)
	// Match 1 (ARG/SAU) and match 2 (resolved CUW via fixture_src_id) should
	// be present. Match 3 (unknown team) should be skipped.
	require.Len(t, matches, 2)

	// Verify the Curaçao match was stored with the canonical code "CUW",
	// not the bogus /matches TLA "CUR".
	ids := []string{matches[0].ID, matches[1].ID}
	require.Contains(t, ids, "2026-06-25-ARG-vs-SAU")
	require.Contains(t, ids, "2026-06-26-CUW-vs-SAU")
	for _, m := range matches {
		require.NotEqual(t, "CUR", m.HomeTeamCode, "match should use canonical team code, not /matches TLA")
	}
}

func TestNormalizeGroupID(t *testing.T) {
	cases := map[string]string{
		"Group A":  "A",
		"Group F":  "F",
		"GROUP_A":  "A",
		"GROUP_F":  "F",
		"F":        "F",
		"group b":  "B",
		"":         "",
		"  Group H ": "H",
	}
	for in, want := range cases {
		require.Equal(t, want, normalizeGroupID(in), "input=%q", in)
	}
}

func TestNormalizeStage(t *testing.T) {
	cases := map[string]string{
		"GROUP_STAGE":    "group",
		"LAST_32":        "round-of-32",
		"ROUND_OF_32":    "round-of-32",
		"LAST_16":        "round-of-16",
		"ROUND_OF_16":    "round-of-16",
		"QUARTER_FINALS": "qf",
		"SEMI_FINALS":    "sf",
		"THIRD_PLACE":    "third-place",
		"FINAL":          "final",
	}
	for in, want := range cases {
		require.Equal(t, want, normalizeStage(in), "input=%s", in)
	}
}
