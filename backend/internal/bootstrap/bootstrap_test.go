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

	require.NoError(t, Run(context.Background(), s, c, t.TempDir()))

	teams, _ := s.ListTeams()
	require.Len(t, teams, 2)
	matches, _ := s.ListMatches()
	require.Len(t, matches, 1)
	require.Equal(t, "2026-06-25-ARG-vs-SAU", matches[0].ID)
}
