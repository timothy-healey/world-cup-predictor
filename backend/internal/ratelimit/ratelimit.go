// Package ratelimit holds the most-recent rate-limit observations for the
// upstream HTTP clients (football-data.org, the-odds-api). Both clients
// record their per-response headers here; the doctor command reads them back
// to surface a human-readable summary.
//
// Global mutable state is intentional: wcp is a single-process CLI and we
// only need the latest reading, not history. Access is mutex-guarded so
// concurrent fetchers can record safely.
package ratelimit

import (
	"sync"
	"time"
)

// FDOrgInfo is the latest observation for football-data.org. RemainingMinute
// is the value of X-Requests-Available-Minute on the last successful
// response; -1 means "no observation yet".
type FDOrgInfo struct {
	RemainingMinute int
	LastUpdated     time.Time
}

// OddsInfo is the latest observation for the-odds-api. Remaining is the
// monthly budget left, Used is the cumulative count this period, and
// LastCost is how many credits the most-recent call consumed. -1 values
// mean "no observation yet".
type OddsInfo struct {
	Remaining   int
	Used        int
	LastCost    int
	LastUpdated time.Time
}

var (
	mu    sync.Mutex
	fdorg = FDOrgInfo{RemainingMinute: -1}
	odds  = OddsInfo{Remaining: -1, Used: -1, LastCost: -1}
)

// RecordFDOrg stores a fresh football-data.org observation.
func RecordFDOrg(remaining int) {
	mu.Lock()
	defer mu.Unlock()
	fdorg = FDOrgInfo{RemainingMinute: remaining, LastUpdated: time.Now()}
}

// RecordOdds stores a fresh the-odds-api observation.
func RecordOdds(remaining, used, lastCost int) {
	mu.Lock()
	defer mu.Unlock()
	odds = OddsInfo{Remaining: remaining, Used: used, LastCost: lastCost, LastUpdated: time.Now()}
}

// FDOrg returns the latest football-data.org observation.
func FDOrg() FDOrgInfo {
	mu.Lock()
	defer mu.Unlock()
	return fdorg
}

// Odds returns the latest the-odds-api observation.
func Odds() OddsInfo {
	mu.Lock()
	defer mu.Unlock()
	return odds
}

// Reset clears all observations. Tests use this to avoid cross-test
// contamination from the package-global state.
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	fdorg = FDOrgInfo{RemainingMinute: -1}
	odds = OddsInfo{Remaining: -1, Used: -1, LastCost: -1}
}
