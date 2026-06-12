package odds

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/timhealey/world-cup-predictor/backend/internal/trace"
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

func TestClientEmitsHTTPWirelog(t *testing.T) {
	body, _ := os.ReadFile("../../testdata/odds-event.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	prev := trace.SetWriter(&buf)
	defer trace.SetWriter(prev)

	c := NewClient(srv.URL, "k")
	_, err := c.GetForMatch(t.Context(), "Argentina", "Saudi Arabia", "2026-06-25T11:00:00Z")
	require.NoError(t, err)

	out := buf.String()
	require.Contains(t, out, "[wcp:odds] → GET "+srv.URL)
	require.Contains(t, out, "[wcp:odds] ✓ 200 (")
}

func TestClientEmitsWirelogOnHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"message":"too many requests"}`))
	}))
	defer srv.Close()

	var buf bytes.Buffer
	prev := trace.SetWriter(&buf)
	defer trace.SetWriter(prev)

	c := NewClient(srv.URL, "k")
	_, _ = c.GetForMatch(t.Context(), "Argentina", "Saudi Arabia", "2026-06-25T11:00:00Z")

	out := buf.String()
	require.Contains(t, out, "[wcp:odds] ✗ 429 (")
}

// TestGetForMatch_AliasedTeams verifies that DB-style names for the five
// teams whose names differ between football-data.org and the-odds-api are
// translated through oddsAPIName before the event lookup. Uses the captured
// live response so the strings on the wire are real, not synthesized.
func TestGetForMatch_AliasedTeams(t *testing.T) {
	body, err := os.ReadFile("../../testdata/odds-wc2026.json")
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	cases := []struct {
		home, away string
	}{
		{"United States", "Paraguay"},
		{"Canada", "Bosnia-Herzegovina"},
		{"Spain", "Cape Verde Islands"},
		{"Portugal", "Congo DR"},
		{"Czechia", "South Africa"},
	}
	for _, c := range cases {
		t.Run(c.home+" vs "+c.away, func(t *testing.T) {
			client := NewClient(srv.URL, "k")
			o, err := client.GetForMatch(t.Context(), c.home, c.away, "2026-06-15T00:00:00Z")
			require.NoError(t, err)
			require.Greater(t, o.HomeOdds, 0.0, "home odds should be populated")
		})
	}
}

// TestGetForMatch_NonAliasedTeams verifies that teams whose names already
// match across both providers still resolve via the identity path.
func TestGetForMatch_NonAliasedTeams(t *testing.T) {
	body, err := os.ReadFile("../../testdata/odds-wc2026.json")
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "k")
	o, err := client.GetForMatch(t.Context(), "Qatar", "Switzerland", "2026-06-13T19:00:00Z")
	require.NoError(t, err)
	require.Greater(t, o.HomeOdds, 0.0)
}
