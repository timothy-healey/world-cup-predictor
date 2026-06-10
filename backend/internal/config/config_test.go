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
	// Pin the "odds disabled when key missing" behavior with all SMTP vars
	// fully set, so we know EmailEnabled() is true and only OddsEnabled() flips.
	t.Setenv("FOOTBALL_DATA_API_KEY", "fd-key")
	t.Setenv("THE_ODDS_API_KEY", "")
	t.Setenv("SMTP_HOST", "smtp.gmail.com")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_USER", "u@example.com")
	t.Setenv("SMTP_PASSWORD", "pw")
	t.Setenv("NOTIFICATION_EMAIL_TO", "to@example.com")

	cfg, err := Load(t.TempDir())
	require.NoError(t, err)
	require.False(t, cfg.OddsEnabled())
	require.True(t, cfg.EmailEnabled())
	require.NotEmpty(t, cfg.Warnings) // should have warned about missing odds key
}

// TestEmailDisabledWhenAnySMTPVarMissing pins the "all five SMTP vars required"
// contract of EmailEnabled() by individually clearing each one while leaving the
// other four populated, and asserting that EmailEnabled() is false specifically
// because that one var is missing.
func TestEmailDisabledWhenAnySMTPVarMissing(t *testing.T) {
	smtpVars := []string{
		"SMTP_HOST",
		"SMTP_PORT",
		"SMTP_USER",
		"SMTP_PASSWORD",
		"NOTIFICATION_EMAIL_TO",
	}
	defaults := map[string]string{
		"SMTP_HOST":             "smtp.gmail.com",
		"SMTP_PORT":             "587",
		"SMTP_USER":             "u@example.com",
		"SMTP_PASSWORD":         "pw",
		"NOTIFICATION_EMAIL_TO": "to@example.com",
	}

	for _, missing := range smtpVars {
		missing := missing
		t.Run("missing="+missing, func(t *testing.T) {
			t.Setenv("FOOTBALL_DATA_API_KEY", "fd-key")
			t.Setenv("THE_ODDS_API_KEY", "odds-key")
			for _, v := range smtpVars {
				if v == missing {
					t.Setenv(v, "")
				} else {
					t.Setenv(v, defaults[v])
				}
			}

			cfg, err := Load(t.TempDir())
			require.NoError(t, err)
			require.True(t, cfg.OddsEnabled())
			require.Falsef(t, cfg.EmailEnabled(),
				"EmailEnabled() should be false when %s is empty", missing)
			require.NotEmpty(t, cfg.Warnings)
		})
	}
}
