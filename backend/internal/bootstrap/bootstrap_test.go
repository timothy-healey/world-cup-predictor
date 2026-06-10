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
