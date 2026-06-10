package predict

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/timhealey/world-cup-predictor/backend/internal/claudec"
	"github.com/timhealey/world-cup-predictor/backend/internal/fetchers"
	"github.com/timhealey/world-cup-predictor/backend/internal/store"

	"github.com/stretchr/testify/require"
)

func TestRunHappyPath(t *testing.T) {
	s, _ := store.Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()
	require.NoError(t, s.UpsertTeam(store.Team{Code: "ARG", Name: "Argentina"}))
	require.NoError(t, s.UpsertTeam(store.Team{Code: "SAU", Name: "Saudi Arabia"}))
	require.NoError(t, s.UpsertMatch(store.Match{
		ID: "m1", HomeTeamCode: "ARG", AwayTeamCode: "SAU",
		KickoffUTC: "2026-06-25T11:00:00Z", Stage: "group",
	}))

	tmp := t.TempDir()
	fake := filepath.Join(tmp, "claude")
	require.NoError(t, os.WriteFile(fake, []byte(`#!/bin/sh
cat <<'EOF'
{"winner":"ARG","predicted_score":"2-0","win_probability":0.71,"reasoning":["a","b"]}
EOF
`), 0o755))

	d := claudec.NewDriver(fake, "test-model")
	pipeline := New(s, d, Deps{
		FetchOdds:    func(ctx context.Context, h, a, k string) (any, bool) { return map[string]float64{"home": 1.4}, true },
		FetchNews:    func(ctx context.Context, d any, h, a string) (fetchers.NewsResult, bool) { return fetchers.NewsResult{HomeSummary: "h", AwaySummary: "a"}, true },
		FetchLineup:  func(ctx context.Context, d any, h, a string) (fetchers.LineupResult, bool) { return fetchers.LineupResult{Confirmed: true}, true },
		FetchContext: func(s *store.Store, h, a string) (fetchers.ContextResult, bool) { return fetchers.ContextResult{}, true },
	})

	rec, err := pipeline.Run(context.Background(), "m1", "on_demand")
	require.NoError(t, err)
	require.Equal(t, "ARG", rec.PredictedWinner)
	require.Equal(t, "high", rec.Confidence) // confirmed XI + all inputs ok
	require.Greater(t, rec.ID, int64(0))
}
