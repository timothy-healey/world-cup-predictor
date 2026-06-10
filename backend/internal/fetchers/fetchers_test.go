package fetchers

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/timhealey/world-cup-predictor/backend/internal/claudec"
	"github.com/timhealey/world-cup-predictor/backend/internal/store"
	"github.com/timhealey/world-cup-predictor/backend/internal/trace"
)

func TestLineupFetcherConfirmedXI(t *testing.T) {
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "claude")
	require.NoError(t, os.WriteFile(fake, []byte(`#!/bin/sh
cat <<'EOF'
{"confirmed": true, "home_xi": ["A","B","C","D","E","F","G","H","I","J","K"], "away_xi": ["a","b","c","d","e","f","g","h","i","j","k"], "notes": "Confirmed via team announcement on X"}
EOF
`), 0o755))

	d := claudec.NewDriver(fake, "test")
	r, err := FetchLineup(context.Background(), d, "Argentina", "Saudi Arabia")
	require.NoError(t, err)
	require.True(t, r.Confirmed)
	require.Len(t, r.HomeXI, 11)
}

func TestLineupFetcherFallback(t *testing.T) {
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "claude")
	require.NoError(t, os.WriteFile(fake, []byte(`#!/bin/sh
cat <<'EOF'
{"confirmed": false, "notes": "No XI posted yet; using 26-man squad."}
EOF
`), 0o755))

	d := claudec.NewDriver(fake, "test")
	r, err := FetchLineup(context.Background(), d, "Argentina", "Saudi Arabia")
	require.NoError(t, err)
	require.False(t, r.Confirmed)
}

func TestNewsFetcher(t *testing.T) {
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "claude")
	require.NoError(t, os.WriteFile(fake, []byte(`#!/bin/sh
cat <<'EOF'
{"home_summary": "Argentina in form.", "away_summary": "Saudi Arabia missing two starters."}
EOF
`), 0o755))

	d := claudec.NewDriver(fake, "test")
	r, err := FetchNews(context.Background(), d, "Argentina", "Saudi Arabia")
	require.NoError(t, err)
	require.Contains(t, r.HomeSummary, "Argentina")
}

func TestContextFetcherEarlyTournament(t *testing.T) {
	s, _ := store.Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()
	require.NoError(t, s.UpsertTeam(store.Team{Code: "ARG", Name: "Argentina"}))
	require.NoError(t, s.UpsertTeam(store.Team{Code: "SAU", Name: "Saudi Arabia"}))

	ctx, err := FetchContext(s, "ARG", "SAU")
	require.NoError(t, err)
	require.Contains(t, ctx.TournamentContext, "No completed")
	require.Contains(t, ctx.TrackRecord, "No predictions")
}

func TestContextFetcherWithHistory(t *testing.T) {
	s, _ := store.Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()
	require.NoError(t, s.UpsertTeam(store.Team{Code: "ARG", Name: "Argentina"}))
	require.NoError(t, s.UpsertTeam(store.Team{Code: "MEX", Name: "Mexico"}))
	require.NoError(t, s.UpsertTeam(store.Team{Code: "SAU", Name: "Saudi Arabia"}))
	require.NoError(t, s.UpsertMatch(store.Match{
		ID: "m1", HomeTeamCode: "ARG", AwayTeamCode: "MEX",
		KickoffUTC: "2026-06-12T18:00:00Z", Stage: "group",
	}))
	require.NoError(t, s.SetMatchResult("m1", 2, 0, "2026-06-12T20:00:00Z"))
	_, _ = s.InsertPrediction(store.Prediction{
		MatchID: "m1", CreatedAt: "2026-06-12T17:30:00Z",
		Trigger: "scheduled", Confidence: "high",
		PredictedWinner: "ARG", PredictedScore: "2-1",
		WinProbability: 0.7, Reasoning: "r", InputsJSON: "{}",
		RenderedPrompt: "", ModelID: "test", PromptVersion: "v1",
	})

	ctx, err := FetchContext(s, "ARG", "SAU")
	require.NoError(t, err)
	require.Contains(t, ctx.TournamentContext, "ARG")
	require.Contains(t, ctx.TournamentContext, "MEX")
	require.Contains(t, ctx.TrackRecord, "1") // some "X of Y"
}

func TestFetchNewsEmitsWirelogWithNewsNamespace(t *testing.T) {
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "claude")
	require.NoError(t, os.WriteFile(fake, []byte(`#!/bin/sh
cat <<'EOF'
{"home_summary":"h","away_summary":"a"}
EOF
`), 0o755))

	var buf bytes.Buffer
	prev := trace.SetWriter(&buf)
	defer trace.SetWriter(prev)

	d := claudec.NewDriver(fake, "test-model")
	_, err := FetchNews(t.Context(), d, "Argentina", "Saudi Arabia")
	require.NoError(t, err)
	require.Contains(t, buf.String(), "[wcp:news] → claude -p")
	require.Contains(t, buf.String(), "[wcp:news] ✓ ok")
}

func TestFetchLineupEmitsWirelogWithLineupNamespace(t *testing.T) {
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "claude")
	require.NoError(t, os.WriteFile(fake, []byte(`#!/bin/sh
cat <<'EOF'
{"confirmed":true,"home_xi":[],"away_xi":[],"notes":""}
EOF
`), 0o755))

	var buf bytes.Buffer
	prev := trace.SetWriter(&buf)
	defer trace.SetWriter(prev)

	d := claudec.NewDriver(fake, "test-model")
	_, err := FetchLineup(t.Context(), d, "ARG", "SAU")
	require.NoError(t, err)
	require.Contains(t, buf.String(), "[wcp:lineup] → claude -p")
}
