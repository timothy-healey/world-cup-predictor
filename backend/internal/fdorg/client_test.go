package fdorg

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClientGetTeamsParsesResponse(t *testing.T) {
	body, err := os.ReadFile("../../testdata/fdorg-teams.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "test-key", r.Header.Get("X-Auth-Token"))
		require.Contains(t, r.URL.Path, "/competitions/WC/teams")
		w.WriteHeader(200)
		w.Write(body)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key")
	teams, err := c.GetTeams(t.Context())
	require.NoError(t, err)
	require.Len(t, teams, 2)
	require.Equal(t, "ARG", teams[0].TLA)
}

func TestClientGetFixturesParsesResponse(t *testing.T) {
	body, err := os.ReadFile("../../testdata/fdorg-matches.json")
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k")
	matches, err := c.GetFixtures(t.Context())
	require.NoError(t, err)
	require.Len(t, matches, 1)
	require.Equal(t, "ARG", matches[0].HomeTLA)
	require.Equal(t, "MetLife Stadium", matches[0].Venue)
	// HomeID/AwayID are required so bootstrap can fall back to fixture_src_id
	// when the /matches TLA doesn't match any team.code (see resolveTeamCode).
	require.Greater(t, matches[0].HomeID, 0)
	require.Greater(t, matches[0].AwayID, 0)

	// Sanity check on JSON round-trip if needed
	_, err = json.Marshal(matches)
	require.NoError(t, err)
}

func TestClientGetFinishedResults(t *testing.T) {
	body := []byte(`{
		"matches": [
			{"id": 12345, "status": "FINISHED", "homeTeam": {"tla": "ARG"}, "awayTeam": {"tla": "SAU"},
			 "utcDate": "2026-06-25T11:00:00Z", "stage": "GROUP_STAGE",
			 "score": {"fullTime": {"home": 2, "away": 0}}}
		]
	}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Contains(t, r.URL.RawQuery, "status=FINISHED")
		w.Write(body)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k")
	results, err := c.GetFinishedResults(t.Context())
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.NotNil(t, results[0].HomeScore)
	require.Equal(t, 2, *results[0].HomeScore)
}

func TestClientReturnsErrorOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"bad key"}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "k")
	_, err := c.GetTeams(t.Context())
	require.Error(t, err)
	require.Contains(t, err.Error(), "401")
}

// TestClientThrottlesOnLowQuota verifies that when the per-minute counter
// reports 1 remaining, the client (a) records the observation into
// LastRateLimit and (b) invokes its sleeper for the 60s throttle. We swap
// in a no-op sleeper so the test runs instantly but still observes the call.
func TestClientThrottlesOnLowQuota(t *testing.T) {
	body, err := os.ReadFile("../../testdata/fdorg-teams.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Requests-Available-Minute", "1")
		w.WriteHeader(200)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	var slept atomic.Int64
	c := NewClient(srv.URL, "k")
	c.sleeper = func(d time.Duration) { slept.Add(int64(d)) }

	var buf bytes.Buffer
	prev := SetWarnWriter(&buf)
	defer SetWarnWriter(prev)

	_, err = c.GetTeams(t.Context())
	require.NoError(t, err)

	rl := c.LastRateLimit()
	require.Equal(t, 1, rl.RemainingMinute)
	require.False(t, rl.LastUpdated.IsZero())
	require.Equal(t, int64(60*time.Second), slept.Load(), "expected a 60s throttle sleep")
	require.Contains(t, buf.String(), "1 req/min remaining")

	// A SECOND call should still succeed (we don't assert sleep duration
	// here — only that no fatal error came back).
	_, err = c.GetTeams(t.Context())
	require.NoError(t, err)
}

// TestClient429Retry checks that a 429 with Retry-After: 0 is retried once
// and the second (successful) response is returned to the caller.
func TestClient429Retry(t *testing.T) {
	body, err := os.ReadFile("../../testdata/fdorg-teams.json")
	require.NoError(t, err)

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message":"rate limited"}`))
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k")
	c.sleeper = func(d time.Duration) {} // no-op

	var buf bytes.Buffer
	prev := SetWarnWriter(&buf)
	defer SetWarnWriter(prev)

	teams, err := c.GetTeams(t.Context())
	require.NoError(t, err)
	require.Len(t, teams, 2)
	require.Equal(t, int32(2), calls.Load(), "expected exactly one retry")
	require.Contains(t, buf.String(), "rate limited")
}

// TestClient429RetryFailsTwice verifies that if the retry ALSO fails the
// error is returned (we don't retry a second time).
func TestClient429RetryFailsTwice(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message":"still limited"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k")
	c.sleeper = func(d time.Duration) {}

	_, err := c.GetTeams(t.Context())
	require.Error(t, err)
	require.Contains(t, err.Error(), "429")
	require.Equal(t, int32(2), calls.Load(), "expected one retry, not more")
}
