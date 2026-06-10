package odds

import (
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
