package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/timhealey/world-cup-predictor/backend/internal/config"
	"github.com/timhealey/world-cup-predictor/backend/internal/ratelimit"
	"github.com/timhealey/world-cup-predictor/backend/internal/server"
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

	// Latest rate-limit observations recorded by the upstream HTTP clients.
	// Both lines always render so users get a consistent layout even when
	// no calls have been made yet this process.
	b.WriteString("\nRate limits (last observed):\n")
	fd := ratelimit.FDOrg()
	if fd.LastUpdated.IsZero() || fd.RemainingMinute < 0 {
		b.WriteString("  football-data.org: no observations yet\n")
	} else {
		b.WriteString(fmt.Sprintf("  football-data.org: %d req/min remaining (observed %s ago)\n",
			fd.RemainingMinute, formatDuration(time.Since(fd.LastUpdated))))
	}
	od := ratelimit.Odds()
	if od.LastUpdated.IsZero() || od.Remaining < 0 {
		b.WriteString("  the-odds-api: no observations yet\n")
	} else {
		b.WriteString(fmt.Sprintf("  the-odds-api: %d of 500 remaining (observed %s ago)\n",
			od.Remaining, formatDuration(time.Since(od.LastUpdated))))
	}

	b.WriteString("\nFrontend bundle:\n")
	if server.DistHasIndex() {
		b.WriteString("  embedded frontend present\n")
	} else {
		b.WriteString("  [warn] frontend not embedded — run `make build` to bundle it\n")
	}

	return b.String()
}

// formatDuration renders a Duration with second precision (we don't care
// about sub-second resolution here — observations are usually minutes old).
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "just now"
	}
	return d.Truncate(time.Second).String()
}

type window struct{ now, in7Days string }

func nowMinusFuture() window {
	// Pulled out so we can stub if ever needed; minimal helper.
	return window{
		now:     timeNowUTC(),
		in7Days: timeIn7DaysUTC(),
	}
}
