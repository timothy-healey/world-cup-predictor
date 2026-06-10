package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadRequiresFootballDataKey(t *testing.T) {
	t.Setenv("FOOTBALL_DATA_API_KEY", "")
	_, err := Load(t.TempDir())
	require.Error(t, err)
	require.Contains(t, err.Error(), "FOOTBALL_DATA_API_KEY")
}

func TestLoadHappyPath(t *testing.T) {
	t.Setenv("FOOTBALL_DATA_API_KEY", "fd-key")
	t.Setenv("THE_ODDS_API_KEY", "odds-key")
	t.Setenv("SMTP_HOST", "smtp.gmail.com")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_USER", "u@example.com")
	t.Setenv("SMTP_PASSWORD", "pw")
	t.Setenv("NOTIFICATION_EMAIL_TO", "to@example.com")

	cfg, err := Load(t.TempDir())
	require.NoError(t, err)
	require.Equal(t, "fd-key", cfg.FootballDataAPIKey)
	require.True(t, cfg.EmailEnabled())
	require.True(t, cfg.OddsEnabled())
}

func TestLoadOptionalFeaturesDisabled(t *testing.T) {
	t.Setenv("FOOTBALL_DATA_API_KEY", "fd-key")
	t.Setenv("THE_ODDS_API_KEY", "")
	t.Setenv("SMTP_USER", "")

	cfg, err := Load(t.TempDir())
	require.NoError(t, err)
	require.False(t, cfg.OddsEnabled())
	require.False(t, cfg.EmailEnabled())
	require.NotEmpty(t, cfg.Warnings) // should have warned about missing keys
}
