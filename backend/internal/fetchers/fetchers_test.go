package fetchers

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/timhealey/world-cup-predictor/backend/internal/claudec"
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
