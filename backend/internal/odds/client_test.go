package odds

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientGetForMatch(t *testing.T) {
	body, _ := os.ReadFile("../../testdata/odds-event.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Contains(t, r.URL.RawQuery, "apiKey=k")
		w.Write(body)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k")
	o, err := c.GetForMatch(t.Context(), "Argentina", "Saudi Arabia", "2026-06-25T11:00:00Z")
	require.NoError(t, err)
	require.Equal(t, 1.40, o.HomeOdds)
	require.Equal(t, 7.50, o.AwayOdds)
	require.Equal(t, 4.20, o.DrawOdds)
	require.InDelta(t, 1.0/1.40, o.HomeImpliedProb, 0.01)
}

// TestClientCapturesRateLimitHeaders verifies that the three documented
// headers are parsed into LastRateLimit and that crossing the low-budget
// threshold emits a warning.
func TestClientCapturesRateLimitHeaders(t *testing.T) {
	body, _ := os.ReadFile("../../testdata/odds-event.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-requests-remaining", "42")
		w.Header().Set("x-requests-used", "458")
		w.Header().Set("x-requests-last", "1")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	prev := SetWarnWriter(&buf)
	defer SetWarnWriter(prev)

	c := NewClient(srv.URL, "k")
	_, err := c.GetForMatch(t.Context(), "Argentina", "Saudi Arabia", "2026-06-25T11:00:00Z")
	require.NoError(t, err)

	rl := c.LastRateLimit()
	require.Equal(t, 42, rl.Remaining)
	require.Equal(t, 458, rl.Used)
	require.Equal(t, 1, rl.LastCost)
	require.False(t, rl.LastUpdated.IsZero())
	require.Contains(t, buf.String(), "only 42 requests left")
}

// TestClientNoWarningAboveThreshold verifies no warning fires when budget
// is comfortably above the threshold.
func TestClientNoWarningAboveThreshold(t *testing.T) {
	body, _ := os.ReadFile("../../testdata/odds-event.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-requests-remaining", "400")
		w.Header().Set("x-requests-used", "100")
		w.Header().Set("x-requests-last", "1")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	prev := SetWarnWriter(&buf)
	defer SetWarnWriter(prev)

	c := NewClient(srv.URL, "k")
	_, err := c.GetForMatch(t.Context(), "Argentina", "Saudi Arabia", "2026-06-25T11:00:00Z")
	require.NoError(t, err)
	require.Empty(t, buf.String(), "no warning should fire when budget is healthy")
	require.Equal(t, 400, c.LastRateLimit().Remaining)
}
