// Package trace records per-prediction fetcher outcomes (Recorder) and emits
// wire-level stderr lines (wirelog.go) so a finished prediction can be debugged
// from either the SQLite trace_json column or the launchd .log file.
package trace

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"unicode/utf8"
)

// kinds is the fixed ordered set of fetcher slots. Adding a new fetcher means
// adding here AND updating the "Adding a new fetcher" section in CLAUDE.md.
var kinds = []string{"odds", "news", "lineup", "context", "predict"}

const snippetMaxBytes = 400

type entry struct {
	Kind       string `json:"kind"`
	StartedAt  string `json:"started_at"`
	DurationMs int64  `json:"duration_ms"`
	OK         bool   `json:"ok"`
	Error      string `json:"error"`
	Snippet    string `json:"snippet"`

	startTime time.Time // internal — not serialized
	started   bool
	finished  bool
}

// Recorder collects per-fetcher trace entries for a single prediction.
// Not goroutine-safe — predict.Pipeline.Run calls Start/Finish from a single
// goroutine after the fan-in.
type Recorder struct {
	entries map[string]*entry
	now     func() time.Time // overridable in tests
}

func New() *Recorder {
	m := make(map[string]*entry, len(kinds))
	for _, k := range kinds {
		m[k] = &entry{Kind: k, Error: "not run"}
	}
	return &Recorder{entries: m, now: time.Now}
}

func (r *Recorder) Start(kind string) {
	e, ok := r.entries[kind]
	if !ok {
		panic(fmt.Sprintf("trace: unknown kind %q", kind))
	}
	e.startTime = r.now().UTC()
	e.StartedAt = e.startTime.Format("2006-01-02T15:04:05.000Z")
	e.started = true
	// reset the defensive "not run" default now that a real run began.
	e.Error = ""
}

func (r *Recorder) Finish(kind string, err error, snippet string) {
	e, ok := r.entries[kind]
	if !ok {
		panic(fmt.Sprintf("trace: unknown kind %q", kind))
	}
	if !e.started {
		panic(fmt.Sprintf("trace: Finish(%q) called before Start", kind))
	}
	if e.finished {
		panic(fmt.Sprintf("trace: Finish(%q) called twice", kind))
	}
	e.DurationMs = r.now().UTC().Sub(e.startTime).Milliseconds()
	e.OK = err == nil
	if err != nil {
		e.Error = err.Error()
	}
	e.Snippet = truncateUTF8(snippet, snippetMaxBytes)
	e.finished = true

	// Emit the per-fetcher summary line to stderr.
	logFetcherSummary(kind, e.OK, e.DurationMs, e.Error)
}

// JSON serializes the five entries in fixed order.
func (r *Recorder) JSON() ([]byte, error) {
	out := make([]entry, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, *r.entries[k])
	}
	return json.Marshal(out)
}

// truncateUTF8 returns s with at most maxBytes bytes, never splitting a rune.
func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

// logFetcherSummary writes the `[wcp:trace]` summary line. Kept in this file
// (not wirelog.go) because it is exclusively triggered by Recorder.Finish.
func logFetcherSummary(kind string, ok bool, durationMs int64, errMsg string) {
	if ok {
		fmt.Fprintf(os.Stderr, "[wcp:trace] %s ✓ ok in %dms\n", kind, durationMs)
		return
	}
	fmt.Fprintf(os.Stderr, "[wcp:trace] %s ✗ failed in %dms — %s\n", kind, durationMs, errMsg)
}
