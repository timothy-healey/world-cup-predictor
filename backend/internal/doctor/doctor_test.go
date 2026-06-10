package doctor

import (
	"path/filepath"
	"testing"

	"github.com/timhealey/world-cup-predictor/backend/internal/config"
	"github.com/timhealey/world-cup-predictor/backend/internal/store"

	"github.com/stretchr/testify/require"
)

func TestReportListsConfigStatus(t *testing.T) {
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
}
