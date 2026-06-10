package fdorg

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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
