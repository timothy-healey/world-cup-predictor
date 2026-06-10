package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	FootballDataAPIKey string
	OddsAPIKey         string
	SMTPHost           string
	SMTPPort           string
	SMTPUser           string
	SMTPPassword       string
	NotificationTo     string
	DBPath             string
	ServePort          string
	ClaudeBin          string

	Warnings []string
}

func (c *Config) OddsEnabled() bool {
	return c.OddsAPIKey != ""
}

func (c *Config) EmailEnabled() bool {
	return c.SMTPHost != "" && c.SMTPPort != "" && c.SMTPUser != "" && c.SMTPPassword != "" && c.NotificationTo != ""
}

func Load(repoRoot string) (*Config, error) {
	// Best-effort: load .env from repoRoot or backend/.env if present.
	// Buffer any parse-error warnings here so we can attach them to cfg.Warnings below.
	var envWarnings []string
	for _, p := range []string{filepath.Join(repoRoot, ".env"), filepath.Join(repoRoot, "backend", ".env")} {
		if _, err := os.Stat(p); err == nil {
			if err := godotenv.Load(p); err != nil {
				envWarnings = append(envWarnings, fmt.Sprintf("failed to parse %s: %v", p, err))
			}
		}
	}

	cfg := &Config{
		FootballDataAPIKey: os.Getenv("FOOTBALL_DATA_API_KEY"),
		OddsAPIKey:         os.Getenv("THE_ODDS_API_KEY"),
		SMTPHost:           os.Getenv("SMTP_HOST"),
		SMTPPort:           os.Getenv("SMTP_PORT"),
		SMTPUser:           os.Getenv("SMTP_USER"),
		SMTPPassword:       os.Getenv("SMTP_PASSWORD"),
		NotificationTo:     os.Getenv("NOTIFICATION_EMAIL_TO"),
		DBPath:             firstNonEmpty(os.Getenv("WCP_DB_PATH"), "./wcp.db"),
		ServePort:          firstNonEmpty(os.Getenv("WCP_SERVE_PORT"), "8765"),
		ClaudeBin:          firstNonEmpty(os.Getenv("WCP_CLAUDE_BIN"), "claude"),
	}

	// Surface any .env parse errors before required-var validation so users see them
	// even when the load fails for a missing required key.
	cfg.Warnings = append(cfg.Warnings, envWarnings...)

	if cfg.FootballDataAPIKey == "" {
		return nil, errors.New("FOOTBALL_DATA_API_KEY is required (see .env.example)")
	}
	if !cfg.OddsEnabled() {
		cfg.Warnings = append(cfg.Warnings, "THE_ODDS_API_KEY not set — odds fetcher will be skipped")
	}
	if !cfg.EmailEnabled() {
		cfg.Warnings = append(cfg.Warnings, "SMTP env vars not fully set — email notifications disabled")
	}
	return cfg, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func (c *Config) PrintWarnings() {
	for _, w := range c.Warnings {
		fmt.Fprintf(os.Stderr, "[warn] %s\n", w)
	}
}
