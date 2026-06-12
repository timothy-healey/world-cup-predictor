package predict

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

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
		FetchOdds: func(ctx context.Context, h, a, k string) (any, error, string) {
			return map[string]float64{"home": 1.4}, nil, "bookmaker=fake home=1.4"
		},
		FetchNews: func(ctx context.Context, d any, h, a string) (fetchers.NewsResult, error, string) {
			return fetchers.NewsResult{HomeSummary: "h", AwaySummary: "a"}, nil, "h / a"
		},
		FetchLineup: func(ctx context.Context, d any, h, a string) (fetchers.LineupResult, error, string) {
			return fetchers.LineupResult{Confirmed: true}, nil, "confirmed=true"
		},
		FetchContext: func(s *store.Store, h, a string) (fetchers.ContextResult, error, string) {
			return fetchers.ContextResult{}, nil, "no completed matches yet"
		},
	})
	pipeline.SetNowFn(func() time.Time {
		return time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC) // 1h before kickoff
	})

	rec, err := pipeline.Run(context.Background(), "m1", "on_demand")
	require.NoError(t, err)
	require.Equal(t, "ARG", rec.PredictedWinner)
	require.Equal(t, "high", rec.Confidence)
	require.Greater(t, rec.ID, int64(0))

	// Trace should contain 5 entries, all ok=true, in fixed order.
	var entries []map[string]any
	require.NoError(t, json.Unmarshal([]byte(rec.TraceJSON), &entries))
	require.Len(t, entries, 5)
	wantKinds := []string{"odds", "news", "lineup", "context", "predict"}
	for i, k := range wantKinds {
		require.Equal(t, k, entries[i]["kind"])
		require.Equal(t, true, entries[i]["ok"], "kind=%s should be ok", k)
		require.Equal(t, "", entries[i]["error"])
	}
}

func TestRun_RejectsAtStartWhenPastKickoff(t *testing.T) {
	s, _ := store.Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()
	require.NoError(t, s.UpsertTeam(store.Team{Code: "ARG", Name: "Argentina"}))
	require.NoError(t, s.UpsertTeam(store.Team{Code: "SAU", Name: "Saudi Arabia"}))
	require.NoError(t, s.UpsertMatch(store.Match{
		ID: "m1", HomeTeamCode: "ARG", AwayTeamCode: "SAU",
		KickoffUTC: "2026-06-25T11:00:00Z", Stage: "group",
	}))

	tmp := t.TempDir()
	// Claude script that records each invocation so we can assert it was NOT called.
	callLog := filepath.Join(tmp, "calls.log")
	fake := filepath.Join(tmp, "claude")
	require.NoError(t, os.WriteFile(fake, []byte("#!/bin/sh\necho called >> "+callLog+"\n"), 0o755))

	d := claudec.NewDriver(fake, "test-model")
	pipeline := New(s, d, Deps{
		FetchOdds: func(ctx context.Context, h, a, k string) (any, error, string) {
			return nil, nil, ""
		},
		FetchNews: func(ctx context.Context, d any, h, a string) (fetchers.NewsResult, error, string) {
			return fetchers.NewsResult{}, nil, ""
		},
		FetchLineup: func(ctx context.Context, d any, h, a string) (fetchers.LineupResult, error, string) {
			return fetchers.LineupResult{}, nil, ""
		},
		FetchContext: func(s *store.Store, h, a string) (fetchers.ContextResult, error, string) {
			return fetchers.ContextResult{}, nil, ""
		},
	})
	// Pin "now" to 1 hour after kickoff.
	pipeline.SetNowFn(func() time.Time {
		return time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	})

	_, err := pipeline.Run(context.Background(), "m1", "on_demand")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrPredictionPastKickoff)

	// Claude must not have been invoked.
	_, statErr := os.Stat(callLog)
	require.True(t, os.IsNotExist(statErr), "claude was called but should not have been")

	// No prediction row should exist for the match.
	preds, err := s.ListPredictionsByMatch("m1")
	require.NoError(t, err)
	require.Len(t, preds, 0)
}

func TestRun_RejectsAtStartWhenExactlyKickoff(t *testing.T) {
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
	require.NoError(t, os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0o755))

	d := claudec.NewDriver(fake, "test-model")
	pipeline := New(s, d, Deps{
		FetchOdds: func(ctx context.Context, h, a, k string) (any, error, string) {
			return nil, nil, ""
		},
		FetchNews: func(ctx context.Context, d any, h, a string) (fetchers.NewsResult, error, string) {
			return fetchers.NewsResult{}, nil, ""
		},
		FetchLineup: func(ctx context.Context, d any, h, a string) (fetchers.LineupResult, error, string) {
			return fetchers.LineupResult{}, nil, ""
		},
		FetchContext: func(s *store.Store, h, a string) (fetchers.ContextResult, error, string) {
			return fetchers.ContextResult{}, nil, ""
		},
	})
	// Pin "now" to exactly kickoff.
	pipeline.SetNowFn(func() time.Time {
		return time.Date(2026, 6, 25, 11, 0, 0, 0, time.UTC)
	})

	_, err := pipeline.Run(context.Background(), "m1", "on_demand")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrPredictionPastKickoff)
}

func TestRunPartialFailureLowersConfidenceAndRecordsErrors(t *testing.T) {
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
{"winner":"ARG","predicted_score":"1-0","win_probability":0.55,"reasoning":["only odds available"]}
EOF
`), 0o755))

	d := claudec.NewDriver(fake, "test-model")
	pipeline := New(s, d, Deps{
		FetchOdds: func(ctx context.Context, h, a, k string) (any, error, string) {
			return map[string]float64{"home": 2.0}, nil, "ok"
		},
		FetchNews: func(ctx context.Context, d any, h, a string) (fetchers.NewsResult, error, string) {
			return fetchers.NewsResult{}, errors.New("claude invoke: context deadline exceeded"), ""
		},
		FetchLineup: func(ctx context.Context, d any, h, a string) (fetchers.LineupResult, error, string) {
			return fetchers.LineupResult{}, errors.New("malformed json"), ""
		},
		FetchContext: func(s *store.Store, h, a string) (fetchers.ContextResult, error, string) {
			return fetchers.ContextResult{}, nil, "context ok"
		},
	})
	pipeline.SetNowFn(func() time.Time {
		return time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC) // 1h before kickoff
	})

	rec, err := pipeline.Run(context.Background(), "m1", "scheduled")
	require.NoError(t, err)
	require.Equal(t, "low", rec.Confidence)
	require.Contains(t, rec.RenderedPrompt, "LINEUP: (not available)")
	require.Contains(t, rec.RenderedPrompt, "NEWS: (not available)")

	var entries []map[string]any
	require.NoError(t, json.Unmarshal([]byte(rec.TraceJSON), &entries))
	require.Equal(t, true, entries[0]["ok"], "odds")
	require.Equal(t, false, entries[1]["ok"], "news")
	require.Equal(t, "claude invoke: context deadline exceeded", entries[1]["error"])
	require.Equal(t, false, entries[2]["ok"], "lineup")
	require.Equal(t, "malformed json", entries[2]["error"])
	require.Equal(t, true, entries[3]["ok"], "context")
	require.Equal(t, true, entries[4]["ok"], "predict")
}
