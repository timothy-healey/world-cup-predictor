package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/timhealey/world-cup-predictor/backend/internal/config"
	"github.com/timhealey/world-cup-predictor/backend/internal/store"
)

func Run(cfg *config.Config, s *store.Store, agentsDir string) string {
	var b strings.Builder
	b.WriteString("wcp doctor — system check\n")
	b.WriteString("==========================\n\n")

	// Required vars
	if cfg.FootballDataAPIKey != "" {
		b.WriteString("FOOTBALL_DATA_API_KEY: OK\n")
	} else {
		b.WriteString("FOOTBALL_DATA_API_KEY: MISSING (required)\n")
	}

	// Optional features
	if cfg.OddsEnabled() {
		b.WriteString("odds fetcher: enabled\n")
	} else {
		b.WriteString("odds fetcher: disabled (THE_ODDS_API_KEY not set)\n")
	}
	if cfg.EmailEnabled() {
		b.WriteString("email: enabled\n")
	} else {
		b.WriteString("email: disabled (SMTP_* not fully set)\n")
	}

	// Claude CLI present?
	if _, err := exec.LookPath(cfg.ClaudeBin); err == nil {
		b.WriteString(fmt.Sprintf("claude binary: found (%s)\n", cfg.ClaudeBin))
	} else {
		b.WriteString(fmt.Sprintf("claude binary: NOT FOUND on PATH (%s)\n", cfg.ClaudeBin))
	}

	// Matches scheduled
	matches, _ := s.ListMatches()
	b.WriteString(fmt.Sprintf("\nMatches in store: %d\n", len(matches)))

	// Agent files in agentsDir
	if agentsDir != "" {
		entries, _ := os.ReadDir(agentsDir)
		count := 0
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "com.wcp.") {
				count++
			}
		}
		b.WriteString(fmt.Sprintf("Launchd agents loaded: %d in %s\n", count, agentsDir))

		// Cross-check: matches in next 7 days without a plist
		now := nowMinusFuture()
		matchesNext7 := 0
		missing := 0
		for _, m := range matches {
			if m.KickoffUTC < now.now || m.KickoffUTC > now.in7Days {
				continue
			}
			matchesNext7++
			path := filepath.Join(agentsDir, "com.wcp."+m.ID+".plist")
			if _, err := os.Stat(path); err != nil {
				missing++
			}
		}
		if matchesNext7 > 0 {
			b.WriteString(fmt.Sprintf("Matches in next 7 days: %d (missing plists: %d)\n", matchesNext7, missing))
		}
	}

	return b.String()
}

type window struct{ now, in7Days string }

func nowMinusFuture() window {
	// Pulled out so we can stub if ever needed; minimal helper.
	return window{
		now:     timeNowUTC(),
		in7Days: timeIn7DaysUTC(),
	}
}
