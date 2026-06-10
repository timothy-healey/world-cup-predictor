package doctor

import (
	"path/filepath"
	"testing"

	"github.com/timhealey/world-cup-predictor/backend/internal/config"
	"github.com/timhealey/world-cup-predictor/backend/internal/ratelimit"
	"github.com/timhealey/world-cup-predictor/backend/internal/store"

	"github.com/stretchr/testify/require"
)

func TestReportListsConfigStatus(t *testing.T) {
	ratelimit.Reset()
	defer ratelimit.Reset()

	s, _ := store.Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()
	cfg := &config.Config{
		FootballDataAPIKey: "fd-key",
		OddsAPIKey:         "", // optional, off
		SMTPUser:           "u",
		SMTPPassword:       "p",
		SMTPHost:           "h",
		SMTPPort:           "p",
		NotificationTo:     "t",
	}

	report := Run(cfg, s, t.TempDir())
	require.Contains(t, report, "FOOTBALL_DATA_API_KEY")
	require.Contains(t, report, "OK")
	require.Contains(t, report, "odds")
	require.Contains(t, report, "disabled")

	// Rate-limit section is always present.
	require.Contains(t, report, "Rate limits (last observed):")
	// With no observations recorded the empty-state messages render.
	require.Contains(t, report, "football-data.org: no observations yet")
	require.Contains(t, report, "the-odds-api: no observations yet")
}

func TestReportShowsRecordedRateLimits(t *testing.T) {
	ratelimit.Reset()
	defer ratelimit.Reset()

	ratelimit.RecordFDOrg(7)
	ratelimit.RecordOdds(123, 377, 1)

	s, _ := store.Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()
	cfg := &config.Config{FootballDataAPIKey: "fd-key"}

	report := Run(cfg, s, t.TempDir())
	require.Contains(t, report, "football-data.org: 7 req/min remaining")
	require.Contains(t, report, "the-odds-api: 123 of 500 remaining")
}
