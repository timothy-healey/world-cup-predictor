# Prediction input tracing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When a prediction lands with degraded inputs, the user can both (a) click into the prediction in the dashboard and see per-fetcher status, error, and a truncated snippet, and (b) tail `~/Library/Logs/wcp/<match-id>.log` to see every HTTP request and `claude -p` subprocess that ran. Today these failures are collapsed to `(data, ok)` booleans in `cmd/wcp/main.go` and the reason is lost.

**Architecture:** Add a new `internal/trace` package that owns both per-prediction trace recording (`Recorder`) and stderr wire-level logging helpers. `predict.Pipeline.Run` constructs a `Recorder`, threads it through new `(data, err, snippet)` fetcher closure signatures in `main.go`, serializes the five-entry array to a new `predictions.trace_json` column, and ships it through `predictions.json` to a new `<PredictionTrace>` accordion in the prediction card. The accordion has two triggers (header pill and `(i)` icon next to "Win probability") that share local `traceOpen` state — neutral grey when 5/5 healthy, primary orange when degraded.

**Tech Stack:** Go 1.22, `modernc.org/sqlite`, `stretchr/testify`, React 18, TypeScript, Vitest, Tailwind. No new dependencies.

---

## File Structure

**New files:**

- `backend/internal/trace/recorder.go` — the per-prediction `Recorder` type and `JSON()` serializer.
- `backend/internal/trace/recorder_test.go` — table-driven tests for `Recorder`.
- `backend/internal/trace/wirelog.go` — package-level stderr helpers (`HTTPStart`/`HTTPEnd`/`HTTPError`, `SubprocessStart`/`SubprocessEnd`/`SubprocessError`) plus a `SetWriter` test seam.
- `backend/internal/trace/wirelog_test.go` — golden-line tests against a `bytes.Buffer`.
- `frontend/src/lib/traceFormat.ts` — pure helpers: `formatDuration`, `okCount`, `pillTone`. Tested.
- `frontend/tests/traceFormat.test.ts` — unit tests for the helpers.
- `frontend/src/components/PredictionTrace.tsx` — presentational drawer (no internal state).

**Modified files:**

- `backend/internal/store/store.go:42-55` — add idempotent `ALTER TABLE predictions ADD COLUMN trace_json TEXT` in `migrate`.
- `backend/internal/store/predictions.go` — add `TraceJSON string` field; update `InsertPrediction`, `ListPredictionsByMatch`, `ListLatestPredictions`.
- `backend/internal/store/store_test.go` — extend with a migration test for `trace_json` and a round-trip test.
- `backend/internal/store/export.go` — add `Trace *json.RawMessage` field to `ExportPrediction`; copy from `Prediction.TraceJSON` (preserving NULL as JSON null).
- `backend/internal/predict/pipeline.go` — change `Deps` signatures from `(data, ok)` to `(data, err, snippet)`; integrate `trace.Recorder`; populate `Prediction.TraceJSON`.
- `backend/internal/predict/pipeline_test.go` — update fakes to the new signatures; assert recorder population.
- `backend/cmd/wcp/main.go:148-171` — rewrite the four fetcher closures to the new signatures with snippet derivation; add the `[wcp:<ns>]` namespace to the subprocess wirelog calls happening one layer down.
- `backend/internal/odds/client.go` — wrap `c.httpc.Do` with `trace.HTTPStart`/`HTTPEnd`/`HTTPError`.
- `backend/internal/odds/client_test.go` — extend to assert wirelog output.
- `backend/internal/fdorg/client.go:102-118` — wrap `c.httpc.Do` in `doRequest` with the same wirelog helpers.
- `backend/internal/claudec/driver.go` — wrap the subprocess call with `trace.SubprocessStart`/`SubprocessEnd`/`SubprocessError`. Namespace comes from the caller (a new `Driver.PredictWithNS` or extending `Predict` to accept ns).
- `backend/internal/fetchers/shared.go` — wrap `cmd.Run()` in `runJSON` with subprocess wirelog helpers; accept namespace via the existing `claudeBin` interface (extended) or via a new parameter.
- `backend/internal/fetchers/news.go` and `lineup.go` — pass namespace strings (`"news"`, `"lineup"`) through to `runJSON`.
- `frontend/src/types/api.ts` — add `TraceEntry` interface and `trace: TraceEntry[] | null` field on `Prediction`.
- `frontend/src/components/PredictionStats.tsx` — accept optional `onTraceClick` and a boolean `traceAvailable`; render the `(i)` icon next to "Win probability" when both are set.
- `frontend/src/components/PredictionBody.tsx` — own `traceOpen` state via `useState`; render the header pill (when trace available); pass `onTraceClick` to `<PredictionStats>`; render `<PredictionTrace>` between the grid and the bottom action bar.
- `CLAUDE.md` — extend the "Adding a new fetcher" section.

**Out of scope (do not touch):** any `predictions_traces` table; bootstrap-time DB persistence; CLI subcommand for past traces; retry/rate-limit-header recording in `trace_json`.

---

## Task 1: `trace.Recorder` — type + happy path

**Files:**
- Create: `backend/internal/trace/recorder.go`
- Create: `backend/internal/trace/recorder_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/trace/recorder_test.go`:

```go
package trace

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fixedNow lets us control timestamps in tests.
func fixedNow(start time.Time, advance time.Duration) func() time.Time {
	calls := 0
	return func() time.Time {
		t := start.Add(time.Duration(calls) * advance)
		calls++
		return t
	}
}

func TestRecorderHappyPath(t *testing.T) {
	r := New()
	r.now = fixedNow(time.Date(2026, 6, 25, 17, 30, 0, 0, time.UTC), 100*time.Millisecond)

	r.Start("odds")
	r.Finish("odds", nil, "bookmaker=william home=1.40")
	r.Start("news")
	r.Finish("news", nil, "h news / a news")
	r.Start("lineup")
	r.Finish("lineup", nil, "confirmed=true notes=")
	r.Start("context")
	r.Finish("context", nil, "Tournament... / Track...")
	r.Start("predict")
	r.Finish("predict", nil, `{"winner":"ARG"}`)

	raw, err := r.JSON()
	require.NoError(t, err)
	var entries []map[string]any
	require.NoError(t, json.Unmarshal(raw, &entries))
	require.Len(t, entries, 5)
	require.Equal(t, "odds", entries[0]["kind"])
	require.Equal(t, "news", entries[1]["kind"])
	require.Equal(t, "lineup", entries[2]["kind"])
	require.Equal(t, "context", entries[3]["kind"])
	require.Equal(t, "predict", entries[4]["kind"])
	require.Equal(t, true, entries[0]["ok"])
	require.Equal(t, "", entries[0]["error"])
	require.Equal(t, float64(100), entries[0]["duration_ms"])
	require.Equal(t, "bookmaker=william home=1.40", entries[0]["snippet"])
	require.Equal(t, "2026-06-25T17:30:00.000Z", entries[0]["started_at"])
}

func TestRecorderErrorIsRecorded(t *testing.T) {
	r := New()
	r.now = fixedNow(time.Date(2026, 6, 25, 17, 30, 0, 0, time.UTC), 380*time.Millisecond)
	r.Start("odds")
	r.Finish("odds", errors.New("no odds found for ARG vs SAU"), "")

	raw, _ := r.JSON()
	var entries []map[string]any
	require.NoError(t, json.Unmarshal(raw, &entries))
	require.Equal(t, false, entries[0]["ok"])
	require.Equal(t, "no odds found for ARG vs SAU", entries[0]["error"])
	require.Equal(t, "", entries[0]["snippet"])
}

func TestRecorderSnippetTruncatedTo400Chars(t *testing.T) {
	r := New()
	r.Start("news")
	long := strings.Repeat("a", 500)
	r.Finish("news", nil, long)
	raw, _ := r.JSON()
	var entries []map[string]any
	require.NoError(t, json.Unmarshal(raw, &entries))
	got := entries[1]["snippet"].(string)
	require.Equal(t, 400, len(got))
}

func TestRecorderSnippetTruncationIsUTF8Safe(t *testing.T) {
	r := New()
	r.Start("news")
	// 200× "ñ" (2 bytes) = 400 bytes, but only 200 runes.
	// 201× "ñ" = 402 bytes, must trim back to 400 bytes WITHOUT splitting a rune.
	s := strings.Repeat("ñ", 201)
	r.Finish("news", nil, s)
	raw, _ := r.JSON()
	var entries []map[string]any
	require.NoError(t, json.Unmarshal(raw, &entries))
	got := entries[1]["snippet"].(string)
	require.LessOrEqual(t, len(got), 400)
	// Truncated string must still be valid UTF-8 (no replacement char appears).
	require.NotContains(t, got, "�")
}

func TestRecorderUnstartedKindMarkedNotRun(t *testing.T) {
	r := New()
	// Start nothing — JSON should still emit five entries with "not run".
	raw, _ := r.JSON()
	var entries []map[string]any
	require.NoError(t, json.Unmarshal(raw, &entries))
	require.Len(t, entries, 5)
	for _, e := range entries {
		require.Equal(t, false, e["ok"])
		require.Equal(t, "not run", e["error"])
		require.Equal(t, float64(0), e["duration_ms"])
	}
}

func TestRecorderFinishWithoutStartPanics(t *testing.T) {
	r := New()
	require.Panics(t, func() { r.Finish("odds", nil, "") })
}

func TestRecorderDoubleFinishPanics(t *testing.T) {
	r := New()
	r.Start("odds")
	r.Finish("odds", nil, "")
	require.Panics(t, func() { r.Finish("odds", nil, "") })
}

func TestRecorderUnknownKindPanics(t *testing.T) {
	r := New()
	require.Panics(t, func() { r.Start("bogus") })
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/trace/...`
Expected: FAIL — package does not exist yet.

- [ ] **Step 3: Implement `Recorder`**

Create `backend/internal/trace/recorder.go`:

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/trace/...`
Expected: PASS — all 8 tests.

- [ ] **Step 5: Run vet**

Run: `cd backend && go vet ./internal/trace/...`
Expected: no output, exit 0.

- [ ] **Step 6: Commit**

```bash
cd backend
git add internal/trace/recorder.go internal/trace/recorder_test.go
git commit -m "feat(trace): add Recorder for per-prediction fetcher tracing"
```

---

## Task 2: `trace.wirelog` — HTTP + subprocess stderr helpers

**Files:**
- Create: `backend/internal/trace/wirelog.go`
- Create: `backend/internal/trace/wirelog_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/trace/wirelog_test.go`:

```go
package trace

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHTTPStartEmitsArrowLine(t *testing.T) {
	var buf bytes.Buffer
	prev := SetWriter(&buf)
	defer SetWriter(prev)

	HTTPStart("odds", "GET", "https://api.the-odds-api.com/v4/sports/soccer_fifa_world_cup/odds/?regions=uk")
	require.Equal(t,
		"[wcp:odds] → GET https://api.the-odds-api.com/v4/sports/soccer_fifa_world_cup/odds/?regions=uk\n",
		buf.String(),
	)
}

func TestHTTPEndEmitsCheckLineWithStatusDurationBytes(t *testing.T) {
	var buf bytes.Buffer
	prev := SetWriter(&buf)
	defer SetWriter(prev)

	HTTPEnd("odds", 200, 412*time.Millisecond, 14336)
	require.Equal(t, "[wcp:odds] ✓ 200 (412ms, 14KB)\n", buf.String())
}

func TestHTTPEndNon2xxEmitsCross(t *testing.T) {
	var buf bytes.Buffer
	prev := SetWriter(&buf)
	defer SetWriter(prev)

	HTTPEnd("odds", 429, 50*time.Millisecond, 128)
	require.Equal(t, "[wcp:odds] ✗ 429 (50ms, 128B)\n", buf.String())
}

func TestHTTPErrorEmitsCrossLine(t *testing.T) {
	var buf bytes.Buffer
	prev := SetWriter(&buf)
	defer SetWriter(prev)

	HTTPError("odds", 15*time.Second, errors.New("dial tcp: i/o timeout"))
	require.Equal(t, "[wcp:odds] ✗ failed after 15000ms: dial tcp: i/o timeout\n", buf.String())
}

func TestSubprocessStartEmitsPromptSize(t *testing.T) {
	var buf bytes.Buffer
	prev := SetWriter(&buf)
	defer SetWriter(prev)

	SubprocessStart("news", 287)
	require.Equal(t, "[wcp:news] → claude -p (prompt: 287 chars)\n", buf.String())
}

func TestSubprocessEndEmitsDurationAndOutputSize(t *testing.T) {
	var buf bytes.Buffer
	prev := SetWriter(&buf)
	defer SetWriter(prev)

	SubprocessEnd("news", 3614*time.Millisecond, 1024)
	require.Equal(t, "[wcp:news] ✓ ok (3614ms, 1KB)\n", buf.String())
}

func TestSubprocessErrorEmitsDurationAndError(t *testing.T) {
	var buf bytes.Buffer
	prev := SetWriter(&buf)
	defer SetWriter(prev)

	SubprocessError("news", 91240*time.Millisecond, errors.New("context deadline exceeded"))
	require.Equal(t, "[wcp:news] ✗ failed after 91240ms: context deadline exceeded\n", buf.String())
}

func TestFormatBytesHumanReadable(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "0B"},
		{1, "1B"},
		{999, "999B"},
		{1000, "1000B"},
		{1024, "1KB"},
		{14336, "14KB"},
		{1048576, "1024KB"}, // 1MB but we only go to KB
	}
	for _, c := range cases {
		require.Equal(t, c.want, formatBytes(c.in), "in=%d", c.in)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/trace/... -run Wirelog`
Expected: FAIL — `SetWriter`, `HTTPStart`, etc. undefined.

- [ ] **Step 3: Implement wirelog**

Create `backend/internal/trace/wirelog.go`:

```go
package trace

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// writer is the destination for all wirelog lines. Defaults to os.Stderr;
// SetWriter replaces it for tests. Guarded by writerMu so SetWriter and the
// logging helpers don't race when tests run in parallel.
var (
	writerMu sync.Mutex
	writer   io.Writer = os.Stderr
)

// SetWriter installs w as the wirelog destination and returns the previous one.
// Pattern matches odds.SetWarnWriter — call SetWriter(prev) in a defer.
func SetWriter(w io.Writer) io.Writer {
	writerMu.Lock()
	defer writerMu.Unlock()
	prev := writer
	writer = w
	return prev
}

func emit(format string, args ...any) {
	writerMu.Lock()
	w := writer
	writerMu.Unlock()
	fmt.Fprintf(w, format+"\n", args...)
}

// HTTPStart logs the outgoing request line.
func HTTPStart(ns, method, url string) {
	emit("[wcp:%s] → %s %s", ns, method, url)
}

// HTTPEnd logs the response status, duration, and body byte count.
// 2xx renders ✓; anything else renders ✗.
func HTTPEnd(ns string, status int, duration time.Duration, bytes int) {
	mark := "✓"
	if status < 200 || status >= 300 {
		mark = "✗"
	}
	emit("[wcp:%s] %s %d (%dms, %s)", ns, mark, status, duration.Milliseconds(), formatBytes(bytes))
}

// HTTPError logs a transport-level failure (no HTTP status was ever received).
func HTTPError(ns string, duration time.Duration, err error) {
	emit("[wcp:%s] ✗ failed after %dms: %s", ns, duration.Milliseconds(), err.Error())
}

// SubprocessStart logs a claude -p invocation about to run.
func SubprocessStart(ns string, promptBytes int) {
	emit("[wcp:%s] → claude -p (prompt: %d chars)", ns, promptBytes)
}

// SubprocessEnd logs a successful exit with duration and stdout size.
func SubprocessEnd(ns string, duration time.Duration, outBytes int) {
	emit("[wcp:%s] ✓ ok (%dms, %s)", ns, duration.Milliseconds(), formatBytes(outBytes))
}

// SubprocessError logs a non-zero exit / context deadline / stderr-bearing failure.
func SubprocessError(ns string, duration time.Duration, err error) {
	emit("[wcp:%s] ✗ failed after %dms: %s", ns, duration.Milliseconds(), err.Error())
}

// formatBytes returns a compact size: B for <1024, KB otherwise (integer KB).
func formatBytes(n int) string {
	if n < 1024 {
		return fmt.Sprintf("%dB", n)
	}
	return fmt.Sprintf("%dKB", n/1024)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/trace/...`
Expected: PASS — all tests including the new wirelog ones.

- [ ] **Step 5: Commit**

```bash
cd backend
git add internal/trace/wirelog.go internal/trace/wirelog_test.go
git commit -m "feat(trace): add wirelog helpers for HTTP and subprocess stderr lines"
```

---

## Task 3: `store` migration — add `trace_json` column

**Files:**
- Modify: `backend/internal/store/store.go`
- Modify: `backend/internal/store/store_test.go`

- [ ] **Step 1: Write the failing migration test**

Add to `backend/internal/store/store_test.go` (after `TestOpenAppliesSchema`):

```go
func TestMigrateAddsTraceJSONColumn(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wcp_test.db")
	s, err := Open(dbPath)
	require.NoError(t, err)
	defer s.Close()

	// Confirm the column exists on the predictions table.
	rows, err := s.DB().Query(`PRAGMA table_info(predictions)`)
	require.NoError(t, err)
	defer rows.Close()

	found := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		require.NoError(t, rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk))
		if name == "trace_json" {
			require.Equal(t, "TEXT", ctype)
			require.Equal(t, 0, notnull, "trace_json must be nullable")
			found = true
		}
	}
	require.True(t, found, "trace_json column missing from predictions")
}

func TestMigrateIsIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wcp_test.db")
	s1, err := Open(dbPath)
	require.NoError(t, err)
	require.NoError(t, s1.Close())

	// Re-open — must not error on the second migration pass.
	s2, err := Open(dbPath)
	require.NoError(t, err)
	require.NoError(t, s2.Close())
}
```

At the top of the file, ensure `"database/sql"` is imported (add if missing).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/store/... -run TestMigrateAddsTraceJSONColumn`
Expected: FAIL — `trace_json column missing`.

- [ ] **Step 3: Add the migration**

Modify `backend/internal/store/store.go`. Replace the `migrate` function body so it runs both the existing `variant` ALTER and the new `trace_json` ALTER:

```go
// migrate runs idempotent ALTER TABLE migrations for columns added after
// the initial schema. CREATE TABLE IF NOT EXISTS in schema.sql doesn't
// touch existing tables, so any post-v1 column lives here.
func migrate(db *sql.DB) error {
	// predictions.variant — added when ablation experiments were planned.
	// Default 'full' so any pre-migration row reads as a production prediction.
	if _, err := db.Exec(`ALTER TABLE predictions ADD COLUMN variant TEXT NOT NULL DEFAULT 'full'`); err != nil {
		if !isDuplicateColumnErr(err) {
			return err
		}
	}
	// predictions.trace_json — per-prediction debug trace (5-entry JSON array).
	// Nullable: predictions written before this column exists read as null and
	// the dashboard hides the trace trigger for them.
	if _, err := db.Exec(`ALTER TABLE predictions ADD COLUMN trace_json TEXT`); err != nil {
		if !isDuplicateColumnErr(err) {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/store/...`
Expected: PASS — all store tests including the two new ones.

- [ ] **Step 5: Commit**

```bash
cd backend
git add internal/store/store.go internal/store/store_test.go
git commit -m "feat(store): add trace_json column to predictions"
```

---

## Task 4: `store.Prediction` — persist and round-trip `TraceJSON`

**Files:**
- Modify: `backend/internal/store/predictions.go`
- Modify: `backend/internal/store/store_test.go`

- [ ] **Step 1: Write the failing round-trip test**

Add to `backend/internal/store/store_test.go`:

```go
func TestPredictionTraceJSONRoundTrip(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()

	require.NoError(t, s.UpsertTeam(Team{Code: "ARG", Name: "Argentina"}))
	require.NoError(t, s.UpsertTeam(Team{Code: "SAU", Name: "Saudi Arabia"}))
	require.NoError(t, s.UpsertMatch(Match{
		ID: "m1", HomeTeamCode: "ARG", AwayTeamCode: "SAU",
		KickoffUTC: "2026-06-25T11:00:00Z", Stage: "group",
	}))

	want := `[{"kind":"odds","ok":false,"error":"x"}]`
	id, err := s.InsertPrediction(Prediction{
		MatchID:         "m1",
		CreatedAt:       "2026-06-25T10:30:00Z",
		Trigger:         "on_demand",
		Confidence:      "low",
		PredictedWinner: "ARG",
		PredictedScore:  "1-0",
		WinProbability:  0.55,
		Reasoning:       "n/a",
		InputsJSON:      "{}",
		RenderedPrompt:  "",
		ModelID:         "test-model",
		PromptVersion:   "v1",
		TraceJSON:       want,
	})
	require.NoError(t, err)
	require.Greater(t, id, int64(0))

	preds, err := s.ListPredictionsByMatch("m1")
	require.NoError(t, err)
	require.Len(t, preds, 1)
	require.Equal(t, want, preds[0].TraceJSON)
}

func TestPredictionTraceJSONIsNullableWhenOmitted(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()

	require.NoError(t, s.UpsertTeam(Team{Code: "ARG", Name: "Argentina"}))
	require.NoError(t, s.UpsertTeam(Team{Code: "SAU", Name: "Saudi Arabia"}))
	require.NoError(t, s.UpsertMatch(Match{
		ID: "m2", HomeTeamCode: "ARG", AwayTeamCode: "SAU",
		KickoffUTC: "2026-06-25T11:00:00Z", Stage: "group",
	}))

	_, err := s.InsertPrediction(Prediction{
		MatchID: "m2", CreatedAt: "x", Trigger: "on_demand", Confidence: "low",
		PredictedWinner: "ARG", PredictedScore: "1-0", WinProbability: 0.5,
		Reasoning: "", InputsJSON: "{}", RenderedPrompt: "", ModelID: "m",
		PromptVersion: "v", // TraceJSON omitted on purpose
	})
	require.NoError(t, err)

	preds, _ := s.ListPredictionsByMatch("m2")
	require.Len(t, preds, 1)
	require.Equal(t, "", preds[0].TraceJSON, "missing trace must read as empty string")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/store/... -run TestPredictionTraceJSONRoundTrip`
Expected: FAIL — `Prediction has no field TraceJSON`.

- [ ] **Step 3: Add `TraceJSON` field and persist it**

Modify `backend/internal/store/predictions.go`. Add the field to the struct:

```go
type Prediction struct {
	ID              int64   `json:"id"`
	MatchID         string  `json:"match_id"`
	CreatedAt       string  `json:"created_at"`
	Trigger         string  `json:"trigger"`
	Confidence      string  `json:"confidence"`
	PredictedWinner string  `json:"predicted_winner"`
	PredictedScore  string  `json:"predicted_score"`
	WinProbability  float64 `json:"win_probability"`
	Reasoning       string  `json:"reasoning"`
	InputsJSON      string  `json:"inputs_json"`
	RenderedPrompt  string  `json:"rendered_prompt"`
	ModelID         string  `json:"model_id"`
	PromptVersion   string  `json:"prompt_version"`
	Variant         string  `json:"variant"`
	TraceJSON       string  `json:"trace_json"` // empty string when NULL in the DB
}
```

Update `InsertPrediction` to write the new column. Replace the function:

```go
func (s *Store) InsertPrediction(p Prediction) (int64, error) {
	variant := p.Variant
	if variant == "" {
		variant = "full"
	}
	// trace_json is nullable; map empty string to SQL NULL so future queries
	// like `WHERE trace_json IS NULL` work for legacy rows.
	var trace sql.NullString
	if p.TraceJSON != "" {
		trace = sql.NullString{String: p.TraceJSON, Valid: true}
	}
	res, err := s.db.Exec(
		`INSERT INTO predictions (match_id, created_at, trigger, confidence, predicted_winner, predicted_score, win_probability, reasoning, inputs_json, rendered_prompt, model_id, prompt_version, variant, trace_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.MatchID, p.CreatedAt, p.Trigger, p.Confidence, p.PredictedWinner, p.PredictedScore,
		p.WinProbability, p.Reasoning, p.InputsJSON, p.RenderedPrompt, p.ModelID, p.PromptVersion, variant, trace,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
```

Update `ListPredictionsByMatch` to read it:

```go
func (s *Store) ListPredictionsByMatch(matchID string) ([]Prediction, error) {
	rows, err := s.db.Query(
		`SELECT id, match_id, created_at, trigger, confidence, predicted_winner, predicted_score, win_probability, reasoning, inputs_json, rendered_prompt, model_id, prompt_version, variant, trace_json
		 FROM predictions WHERE match_id = ? ORDER BY created_at DESC`,
		matchID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Prediction
	for rows.Next() {
		var p Prediction
		var prob sql.NullFloat64
		var trace sql.NullString
		if err := rows.Scan(&p.ID, &p.MatchID, &p.CreatedAt, &p.Trigger, &p.Confidence,
			&p.PredictedWinner, &p.PredictedScore, &prob, &p.Reasoning,
			&p.InputsJSON, &p.RenderedPrompt, &p.ModelID, &p.PromptVersion, &p.Variant, &trace); err != nil {
			return nil, err
		}
		p.WinProbability = prob.Float64
		if trace.Valid {
			p.TraceJSON = trace.String
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
```

Apply the same change to `ListLatestPredictions` — add `trace_json` to the SELECT, declare a `sql.NullString trace`, scan it, and assign on `Valid`.

- [ ] **Step 4: Run all store tests**

Run: `cd backend && go test ./internal/store/...`
Expected: PASS — including the two new round-trip tests.

- [ ] **Step 5: Run vet**

Run: `cd backend && go vet ./internal/store/...`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
cd backend
git add internal/store/predictions.go internal/store/store_test.go
git commit -m "feat(store): persist and read trace_json on predictions"
```

---

## Task 5: `predict.Deps` — change fetcher signatures and integrate `Recorder`

This is the biggest single backend change. We change `Deps` from `(data, ok)` to `(data, err, snippet)`, wire `trace.Recorder` into `Pipeline.Run`, and update the test fakes. The closure call sites in `main.go` move in Task 7.

**Files:**
- Modify: `backend/internal/predict/pipeline.go`
- Modify: `backend/internal/predict/pipeline_test.go`

- [ ] **Step 1: Update the test fakes and add a trace assertion**

Modify `backend/internal/predict/pipeline_test.go`. Replace the two existing tests with versions that match the new `Deps` signatures and assert `TraceJSON` content:

```go
package predict

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/timhealey/world-cup-predictor/backend/internal/claudec"
	"github.com/timhealey/world-cup-predictor/backend/internal/fetchers"
	"github.com/timhealey/world-cup-predictor/backend/internal/store"

	"github.com/stretchr/testify/require"
)

func TestRunHappyPath(t *testing.T) {
	s, _ := store.Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()
	require.NoError(t, s.UpsertTeam(store.Team{Code: "ARG", Name: "Argentina"}))
	require.NoError(t, s.UpsertTeam(store.Team{Code: "SAU", Name: "Saudi Arabia"}))
	require.NoError(t, s.UpsertMatch(store.Match{
		ID: "m1", HomeTeamCode: "ARG", AwayTeamCode: "SAU",
		KickoffUTC: "2026-06-25T11:00:00Z", Stage: "group",
	}))

	tmp := t.TempDir()
	fake := filepath.Join(tmp, "claude")
	require.NoError(t, os.WriteFile(fake, []byte(`#!/bin/sh
cat <<'EOF'
{"winner":"ARG","predicted_score":"2-0","win_probability":0.71,"reasoning":["a","b"]}
EOF
`), 0o755))

	d := claudec.NewDriver(fake, "test-model")
	pipeline := New(s, d, Deps{
		FetchOdds: func(ctx context.Context, h, a, k string) (any, error, string) {
			return map[string]float64{"home": 1.4}, nil, "bookmaker=fake home=1.4"
		},
		FetchNews: func(ctx context.Context, d any, h, a string) (fetchers.NewsResult, error, string) {
			return fetchers.NewsResult{HomeSummary: "h", AwaySummary: "a"}, nil, "h / a"
		},
		FetchLineup: func(ctx context.Context, d any, h, a string) (fetchers.LineupResult, error, string) {
			return fetchers.LineupResult{Confirmed: true}, nil, "confirmed=true"
		},
		FetchContext: func(s *store.Store, h, a string) (fetchers.ContextResult, error, string) {
			return fetchers.ContextResult{}, nil, "no completed matches yet"
		},
	})

	rec, err := pipeline.Run(context.Background(), "m1", "on_demand")
	require.NoError(t, err)
	require.Equal(t, "ARG", rec.PredictedWinner)
	require.Equal(t, "high", rec.Confidence)
	require.Greater(t, rec.ID, int64(0))

	// Trace should contain 5 entries, all ok=true, in fixed order.
	var entries []map[string]any
	require.NoError(t, json.Unmarshal([]byte(rec.TraceJSON), &entries))
	require.Len(t, entries, 5)
	wantKinds := []string{"odds", "news", "lineup", "context", "predict"}
	for i, k := range wantKinds {
		require.Equal(t, k, entries[i]["kind"])
		require.Equal(t, true, entries[i]["ok"], "kind=%s should be ok", k)
		require.Equal(t, "", entries[i]["error"])
	}
}

func TestRunPartialFailureLowersConfidenceAndRecordsErrors(t *testing.T) {
	s, _ := store.Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()
	require.NoError(t, s.UpsertTeam(store.Team{Code: "ARG", Name: "Argentina"}))
	require.NoError(t, s.UpsertTeam(store.Team{Code: "SAU", Name: "Saudi Arabia"}))
	require.NoError(t, s.UpsertMatch(store.Match{
		ID: "m1", HomeTeamCode: "ARG", AwayTeamCode: "SAU",
		KickoffUTC: "2026-06-25T11:00:00Z", Stage: "group",
	}))

	tmp := t.TempDir()
	fake := filepath.Join(tmp, "claude")
	require.NoError(t, os.WriteFile(fake, []byte(`#!/bin/sh
cat <<'EOF'
{"winner":"ARG","predicted_score":"1-0","win_probability":0.55,"reasoning":["only odds available"]}
EOF
`), 0o755))

	d := claudec.NewDriver(fake, "test-model")
	pipeline := New(s, d, Deps{
		FetchOdds: func(ctx context.Context, h, a, k string) (any, error, string) {
			return map[string]float64{"home": 2.0}, nil, "ok"
		},
		FetchNews: func(ctx context.Context, d any, h, a string) (fetchers.NewsResult, error, string) {
			return fetchers.NewsResult{}, errors.New("claude invoke: context deadline exceeded"), ""
		},
		FetchLineup: func(ctx context.Context, d any, h, a string) (fetchers.LineupResult, error, string) {
			return fetchers.LineupResult{}, errors.New("malformed json"), ""
		},
		FetchContext: func(s *store.Store, h, a string) (fetchers.ContextResult, error, string) {
			return fetchers.ContextResult{}, nil, "context ok"
		},
	})

	rec, err := pipeline.Run(context.Background(), "m1", "scheduled")
	require.NoError(t, err)
	require.Equal(t, "low", rec.Confidence)
	require.Contains(t, rec.RenderedPrompt, "LINEUP: (not available)")
	require.Contains(t, rec.RenderedPrompt, "NEWS: (not available)")

	var entries []map[string]any
	require.NoError(t, json.Unmarshal([]byte(rec.TraceJSON), &entries))
	require.Equal(t, true, entries[0]["ok"], "odds")
	require.Equal(t, false, entries[1]["ok"], "news")
	require.Equal(t, "claude invoke: context deadline exceeded", entries[1]["error"])
	require.Equal(t, false, entries[2]["ok"], "lineup")
	require.Equal(t, "malformed json", entries[2]["error"])
	require.Equal(t, true, entries[3]["ok"], "context")
	require.Equal(t, true, entries[4]["ok"], "predict")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/predict/... -run TestRunHappyPath`
Expected: FAIL — `cannot use func(... any, bool)` (signature mismatch).

- [ ] **Step 3: Update `Deps` and `Pipeline.Run`**

Modify `backend/internal/predict/pipeline.go`. Update imports:

```go
import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/timhealey/world-cup-predictor/backend/internal/claudec"
	"github.com/timhealey/world-cup-predictor/backend/internal/fetchers"
	"github.com/timhealey/world-cup-predictor/backend/internal/store"
	"github.com/timhealey/world-cup-predictor/backend/internal/trace"
)
```

Replace the `Deps` struct:

```go
// Deps holds injectable fetcher functions for testing. Each fetcher returns:
//   - data: the fetcher result (typed per kind)
//   - err: non-nil iff the fetcher failed; the pipeline treats nil as "ok"
//   - snippet: a human-readable preview for the trace (caller is free to
//     return "" on failure; truncation to 400 bytes happens in trace.Recorder)
type Deps struct {
	FetchOdds    func(ctx context.Context, homeName, awayName, kickoff string) (any, error, string)
	FetchNews    func(ctx context.Context, d any, home, away string) (fetchers.NewsResult, error, string)
	FetchLineup  func(ctx context.Context, d any, home, away string) (fetchers.LineupResult, error, string)
	FetchContext func(s *store.Store, homeCode, awayCode string) (fetchers.ContextResult, error, string)
}
```

Replace `Pipeline.Run` with a version that uses `trace.Recorder`. Keep the goroutine fan-out for parallelism; capture per-fetcher `err` + `snippet` alongside `data`:

```go
func (p *Pipeline) Run(ctx context.Context, matchID, trigger string) (store.Prediction, error) {
	m, err := p.store.GetMatch(matchID)
	if err != nil {
		return store.Prediction{}, fmt.Errorf("get match: %w", err)
	}
	home, _ := p.store.GetTeam(m.HomeTeamCode)
	away, _ := p.store.GetTeam(m.AwayTeamCode)

	rec := trace.New()

	type oddsR struct {
		data    any
		err     error
		snippet string
	}
	type newsR struct {
		data    fetchers.NewsResult
		err     error
		snippet string
	}
	type lineupR struct {
		data    fetchers.LineupResult
		err     error
		snippet string
	}
	type ctxR struct {
		data    fetchers.ContextResult
		err     error
		snippet string
	}

	oCh := make(chan oddsR, 1)
	nCh := make(chan newsR, 1)
	lCh := make(chan lineupR, 1)
	cCh := make(chan ctxR, 1)

	// Start the four trace timers up front so each kind has a started_at
	// regardless of which goroutine finishes first.
	rec.Start("odds")
	rec.Start("news")
	rec.Start("lineup")
	rec.Start("context")

	go func() {
		d, e, s := p.deps.FetchOdds(ctx, home.Name, away.Name, m.KickoffUTC)
		oCh <- oddsR{d, e, s}
	}()
	go func() {
		d, e, s := p.deps.FetchNews(ctx, p.claude, home.Name, away.Name)
		nCh <- newsR{d, e, s}
	}()
	go func() {
		d, e, s := p.deps.FetchLineup(ctx, p.claude, home.Name, away.Name)
		lCh <- lineupR{d, e, s}
	}()
	go func() {
		d, e, s := p.deps.FetchContext(p.store, m.HomeTeamCode, m.AwayTeamCode)
		cCh <- ctxR{d, e, s}
	}()

	odds := <-oCh
	news := <-nCh
	lineup := <-lCh
	context_ := <-cCh

	rec.Finish("odds", odds.err, odds.snippet)
	rec.Finish("news", news.err, news.snippet)
	rec.Finish("lineup", lineup.err, lineup.snippet)
	rec.Finish("context", context_.err, context_.snippet)

	conf := Confidence(Inputs{
		LineupOK:        lineup.err == nil,
		LineupConfirmed: lineup.err == nil && lineup.data.Confirmed,
		OddsOK:          odds.err == nil,
		NewsOK:          news.err == nil,
		ContextOK:       context_.err == nil,
	})

	inputsRaw, _ := json.Marshal(map[string]any{
		"odds":    odds.data,
		"news":    news.data,
		"lineup":  lineup.data,
		"context": context_.data,
	})

	prompt := claudec.BuildPrompt(claudec.PromptInputs{
		SystemPrompt: p.systemPrompt,
		HomeName:     home.Name,
		AwayName:     away.Name,
		KickoffUTC:   m.KickoffUTC,
		Stage:        m.Stage,
		OddsBlock: func() string {
			if odds.err != nil {
				return ""
			}
			return blockify(odds.data)
		}(),
		NewsBlock: func() string {
			if news.err != nil {
				return ""
			}
			return fmt.Sprintf("Home: %s\nAway: %s", news.data.HomeSummary, news.data.AwaySummary)
		}(),
		LineupBlock: func() string {
			if lineup.err != nil {
				return ""
			}
			return fmt.Sprintf("Confirmed: %v\nNotes: %s", lineup.data.Confirmed, lineup.data.Notes)
		}(),
		ContextBlock: func() string {
			if context_.err != nil {
				return ""
			}
			return strings.TrimSpace(context_.data.TournamentContext + "\n\n" + context_.data.TrackRecord)
		}(),
	})

	rec.Start("predict")
	res, err := p.claude.Predict(ctx, prompt)
	rec.Finish("predict", err, predictSnippet(res, err))
	if err != nil {
		return store.Prediction{}, fmt.Errorf("claude predict: %w", err)
	}

	traceBytes, _ := rec.JSON()

	pred := store.Prediction{
		MatchID:         matchID,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
		Trigger:         trigger,
		Confidence:      conf,
		PredictedWinner: res.Winner,
		PredictedScore:  res.PredictedScore,
		WinProbability:  res.WinProbability,
		Reasoning:       strings.Join(res.Reasoning, "\n- "),
		InputsJSON:      string(inputsRaw),
		RenderedPrompt:  prompt,
		ModelID:         p.claude.ModelID(),
		PromptVersion:   p.promptVersion,
		Variant:         "full",
		TraceJSON:       string(traceBytes),
	}
	id, err := p.store.InsertPrediction(pred)
	if err != nil {
		return store.Prediction{}, fmt.Errorf("insert prediction: %w", err)
	}
	pred.ID = id
	return pred, nil
}

// predictSnippet derives a short preview from the claude predict result.
// On error the result is the zero value, so we return an empty snippet — the
// error string already carries the diagnostic context.
func predictSnippet(res claudec.Result, err error) string {
	if err != nil {
		return ""
	}
	b, jerr := json.Marshal(map[string]any{
		"winner":          res.Winner,
		"predicted_score": res.PredictedScore,
		"win_probability": res.WinProbability,
	})
	if jerr != nil {
		return ""
	}
	return string(b)
}
```

Note: the `predict` trace entry will be missing if claude itself fails — that is intentional because the prediction row is not written in that case, so there is no `trace_json` to ship.

The result type is `claudec.Result` (confirmed in [backend/internal/claudec/driver.go:31-36](backend/internal/claudec/driver.go#L31-L36)).

- [ ] **Step 4: Run pipeline tests to verify they pass**

Run: `cd backend && go test ./internal/predict/...`
Expected: PASS — both updated tests plus existing confidence tests.

- [ ] **Step 5: Try a full build to catch ripple errors in main.go (expected to fail)**

Run: `cd backend && go build ./...`
Expected: FAIL — `cmd/wcp/main.go` still uses the old `(data, ok)` signatures. We'll fix that in Task 7.

- [ ] **Step 6: Commit**

```bash
cd backend
git add internal/predict/pipeline.go internal/predict/pipeline_test.go
git commit -m "feat(predict): integrate trace.Recorder; change Deps to (data, err, snippet)"
```

The tree is intentionally not buildable after this commit — Task 7 closes the gap. We accept a one-commit break to keep each commit focused. If a green-tree-per-commit policy applies later, squash Tasks 5 + 6 + 7 before merging.

---

## Task 6: `claudec.Driver` — wire subprocess wirelog around `invoke()`

**Files:**
- Modify: `backend/internal/claudec/driver.go`
- Modify: `backend/internal/claudec/driver_test.go`

The driver's actual `exec.CommandContext(...).Run()` call lives in the unexported `invoke()` method (see [backend/internal/claudec/driver.go:70-95](backend/internal/claudec/driver.go#L70-L95)), not `Predict()`. `Predict()` calls `invoke()` and may retry once on malformed JSON. Wirelog goes inside `invoke()` so each subprocess attempt gets its own start/end pair — useful when debugging a retry.

- [ ] **Step 1: Write a failing wirelog test**

Add to `backend/internal/claudec/driver_test.go`:

```go
func TestPredictEmitsWirelogLines(t *testing.T) {
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "claude")
	require.NoError(t, os.WriteFile(fake, []byte(`#!/bin/sh
cat <<'EOF'
{"winner":"ARG","predicted_score":"1-0","win_probability":0.5,"reasoning":["x"]}
EOF
`), 0o755))

	var buf bytes.Buffer
	prev := trace.SetWriter(&buf)
	defer trace.SetWriter(prev)

	d := NewDriver(fake, "test-model")
	_, err := d.Predict(t.Context(), "some prompt")
	require.NoError(t, err)

	out := buf.String()
	require.Contains(t, out, "[wcp:predict] → claude -p (prompt: 11 chars)")
	require.Contains(t, out, "[wcp:predict] ✓ ok")
}

func TestPredictEmitsWirelogErrorOnMalformedJSON(t *testing.T) {
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "claude")
	// Always emit garbage so both the initial invoke and the retry fail.
	require.NoError(t, os.WriteFile(fake, []byte(`#!/bin/sh
echo "not json at all"
`), 0o755))

	var buf bytes.Buffer
	prev := trace.SetWriter(&buf)
	defer trace.SetWriter(prev)

	d := NewDriver(fake, "test-model")
	_, err := d.Predict(t.Context(), "p")
	require.Error(t, err)

	out := buf.String()
	// Two invocations happen (initial + retry); we should see at least two
	// start lines and at least one error line for the malformed JSON.
	require.GreaterOrEqual(t, strings.Count(out, "[wcp:predict] → claude -p"), 2)
	require.Contains(t, out, "[wcp:predict] ✗ failed after")
}
```

Add imports to `driver_test.go`: `"bytes"`, `"os"`, `"path/filepath"`, `"strings"`, and `"github.com/timhealey/world-cup-predictor/backend/internal/trace"`. The existing driver test file already imports `testing` and `require` from testify; check before re-adding.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/claudec/... -run TestPredictEmits`
Expected: FAIL — no wirelog output captured.

- [ ] **Step 3: Wire the wirelog into `invoke()`**

Modify `backend/internal/claudec/driver.go`. Add the trace import:

```go
import (
	// ... existing imports ...
	"github.com/timhealey/world-cup-predictor/backend/internal/trace"
)
```

Rewrite `invoke()` to record start/end/error around the `cmd.Run()` call. Note: when `cmd.Run()` succeeds but the JSON parse fails, that is an error caused inside this method (not the subprocess) — log a successful subprocess end and let `Predict()`'s retry path handle the malformed-json error. The retry will issue another `invoke()` call which gets its own wirelog pair.

```go
func (d *Driver) invoke(ctx context.Context, prompt string) (Result, error) {
	timed, cancel := context.WithTimeout(ctx, d.invokeTimeout())
	defer cancel()
	cmd := exec.CommandContext(timed, d.binPath, "-p", prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	trace.SubprocessStart("predict", len(prompt))
	start := time.Now()
	runErr := cmd.Run()
	dur := time.Since(start)
	if runErr != nil {
		wrapped := fmt.Errorf("claude invoke: %w (stderr: %s)", runErr, strings.TrimSpace(stderr.String()))
		trace.SubprocessError("predict", dur, wrapped)
		return Result{}, wrapped
	}
	trace.SubprocessEnd("predict", dur, stdout.Len())

	out := stdout.Bytes()
	startIdx := bytes.IndexByte(out, '{')
	end := bytes.LastIndexByte(out, '}')
	if startIdx < 0 || end <= startIdx {
		trace.SubprocessError("predict", dur, errMalformedJSON)
		return Result{}, errMalformedJSON
	}
	var r Result
	if err := json.Unmarshal(out[startIdx:end+1], &r); err != nil {
		trace.SubprocessError("predict", dur, errMalformedJSON)
		return Result{}, errMalformedJSON
	}
	if r.Winner == "" || r.PredictedScore == "" {
		trace.SubprocessError("predict", dur, errMalformedJSON)
		return Result{}, errMalformedJSON
	}
	return r, nil
}
```

Now both the subprocess-level failure (non-zero exit / timeout) and the parse-level failure emit a clear wirelog line. The namespace is hardcoded to `"predict"` because this driver is dedicated to the main predict call; fetcher subprocesses go through `internal/fetchers/shared.go` (Task 7).

- [ ] **Step 5: Run claudec tests to verify they pass**

Run: `cd backend && go test ./internal/claudec/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
cd backend
git add internal/claudec/driver.go internal/claudec/driver_test.go
git commit -m "feat(claudec): emit subprocess wirelog around claude -p predict call"
```

---

## Task 7: `fetchers/shared.go` — parameterize subprocess namespace + wirelog

**Files:**
- Modify: `backend/internal/fetchers/shared.go`
- Modify: `backend/internal/fetchers/news.go`
- Modify: `backend/internal/fetchers/lineup.go`
- Modify: `backend/internal/fetchers/fetchers_test.go`

The current `runJSON[T]` takes a `claudeBin` and a prompt. We need it to also know which namespace (`news`, `lineup`) so the wirelog line is correctly tagged. Read [backend/internal/fetchers/fetchers_test.go](backend/internal/fetchers/fetchers_test.go) first to see what shape the tests are in.

- [ ] **Step 1: Write the failing test**

Add to `backend/internal/fetchers/fetchers_test.go` (adjust imports as needed):

```go
func TestFetchNewsEmitsWirelogWithNewsNamespace(t *testing.T) {
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "claude")
	require.NoError(t, os.WriteFile(fake, []byte(`#!/bin/sh
cat <<'EOF'
{"home_summary":"h","away_summary":"a"}
EOF
`), 0o755))

	var buf bytes.Buffer
	prev := trace.SetWriter(&buf)
	defer trace.SetWriter(prev)

	d := claudec.NewDriver(fake, "test-model")
	_, err := FetchNews(t.Context(), d, "Argentina", "Saudi Arabia")
	require.NoError(t, err)
	require.Contains(t, buf.String(), "[wcp:news] → claude -p")
	require.Contains(t, buf.String(), "[wcp:news] ✓ ok")
}

func TestFetchLineupEmitsWirelogWithLineupNamespace(t *testing.T) {
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "claude")
	require.NoError(t, os.WriteFile(fake, []byte(`#!/bin/sh
cat <<'EOF'
{"confirmed":true,"home_xi":[],"away_xi":[],"notes":""}
EOF
`), 0o755))

	var buf bytes.Buffer
	prev := trace.SetWriter(&buf)
	defer trace.SetWriter(prev)

	d := claudec.NewDriver(fake, "test-model")
	_, err := FetchLineup(t.Context(), d, "ARG", "SAU")
	require.NoError(t, err)
	require.Contains(t, buf.String(), "[wcp:lineup] → claude -p")
}
```

Imports needed (add to `fetchers_test.go`): `"bytes"`, `"os"`, `"path/filepath"`, `"github.com/timhealey/world-cup-predictor/backend/internal/claudec"`, `"github.com/timhealey/world-cup-predictor/backend/internal/trace"`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/fetchers/... -run TestFetchNewsEmitsWirelog`
Expected: FAIL — no wirelog output.

- [ ] **Step 3: Add namespace parameter to `runJSON`**

Modify `backend/internal/fetchers/shared.go`:

```go
package fetchers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/timhealey/world-cup-predictor/backend/internal/trace"
)

type claudeBin interface {
	BinPathRaw() string
}

func runJSON[T any](ctx context.Context, d claudeBin, ns, prompt string) (T, error) {
	var zero T
	timed, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(timed, d.BinPathRaw(), "-p", prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	trace.SubprocessStart(ns, len(prompt))
	start := time.Now()
	err := cmd.Run()
	dur := time.Since(start)
	if err != nil {
		wrapped := fmt.Errorf("claude invoke: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
		trace.SubprocessError(ns, dur, wrapped)
		return zero, wrapped
	}
	trace.SubprocessEnd(ns, dur, stdout.Len())

	out := stdout.Bytes()
	startIdx := bytes.IndexByte(out, '{')
	end := bytes.LastIndexByte(out, '}')
	if startIdx < 0 || end <= startIdx {
		return zero, errors.New("malformed json")
	}
	if err := json.Unmarshal(out[startIdx:end+1], &zero); err != nil {
		return zero, err
	}
	return zero, nil
}
```

Update `backend/internal/fetchers/news.go`:

```go
func FetchNews(ctx context.Context, d claudeBin, home, away string) (NewsResult, error) {
	prompt := fmt.Sprintf(`Summarize the most relevant football news for %s and %s
from the last 14 days. Focus on: injuries, suspensions, form, off-field issues
that could affect the match. Keep each summary under 80 words.

Reply with ONLY this JSON, no prose:
{
  "home_summary": "...",
  "away_summary": "..."
}`, home, away)
	return runJSON[NewsResult](ctx, d, "news", prompt)
}
```

Update `backend/internal/fetchers/lineup.go` similarly — pass `"lineup"` as the namespace:

```go
return runJSON[LineupResult](ctx, d, "lineup", prompt)
```

- [ ] **Step 4: Run fetcher tests**

Run: `cd backend && go test ./internal/fetchers/...`
Expected: PASS — including the two new wirelog tests.

- [ ] **Step 5: Commit**

```bash
cd backend
git add internal/fetchers/shared.go internal/fetchers/news.go internal/fetchers/lineup.go internal/fetchers/fetchers_test.go
git commit -m "feat(fetchers): parameterize runJSON namespace and emit subprocess wirelog"
```

---

## Task 8: `odds.Client` — HTTP wirelog around `httpc.Do`

**Files:**
- Modify: `backend/internal/odds/client.go`
- Modify: `backend/internal/odds/client_test.go`

- [ ] **Step 1: Write the failing test**

Add to `backend/internal/odds/client_test.go`:

```go
func TestClientEmitsHTTPWirelog(t *testing.T) {
	body, _ := os.ReadFile("../../testdata/odds-event.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	prev := trace.SetWriter(&buf)
	defer trace.SetWriter(prev)

	c := NewClient(srv.URL, "k")
	_, err := c.GetForMatch(t.Context(), "Argentina", "Saudi Arabia", "2026-06-25T11:00:00Z")
	require.NoError(t, err)

	out := buf.String()
	require.Contains(t, out, "[wcp:odds] → GET "+srv.URL)
	require.Contains(t, out, "[wcp:odds] ✓ 200 (")
}

func TestClientEmitsWirelogOnHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"message":"too many requests"}`))
	}))
	defer srv.Close()

	var buf bytes.Buffer
	prev := trace.SetWriter(&buf)
	defer trace.SetWriter(prev)

	c := NewClient(srv.URL, "k")
	_, _ = c.GetForMatch(t.Context(), "Argentina", "Saudi Arabia", "2026-06-25T11:00:00Z")

	out := buf.String()
	require.Contains(t, out, "[wcp:odds] ✗ 429 (")
}
```

Add the import: `"github.com/timhealey/world-cup-predictor/backend/internal/trace"`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/odds/... -run TestClientEmitsHTTPWirelog`
Expected: FAIL.

- [ ] **Step 3: Wire the wirelog into `GetForMatch`**

Modify `backend/internal/odds/client.go`. Replace the request execution block in `GetForMatch`:

```go
import (
	// ... existing imports ...
	"github.com/timhealey/world-cup-predictor/backend/internal/trace"
)

// ...

func (c *Client) GetForMatch(ctx context.Context, homeName, awayName, kickoffUTC string) (Odds, error) {
	q := url.Values{}
	q.Set("apiKey", c.apiKey)
	q.Set("regions", "uk,us,eu,au")
	q.Set("markets", "h2h")
	q.Set("dateFormat", "iso")
	u := fmt.Sprintf("%s/v4/sports/soccer_fifa_world_cup/odds/?%s", c.baseURL, q.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return Odds{}, err
	}

	trace.HTTPStart("odds", "GET", u)
	start := time.Now()
	resp, err := c.httpc.Do(req)
	if err != nil {
		trace.HTTPError("odds", time.Since(start), err)
		return Odds{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	trace.HTTPEnd("odds", resp.StatusCode, time.Since(start), len(body))

	// Capture rate-limit headers regardless of status.
	c.captureRateLimit(resp.Header)

	if resp.StatusCode != 200 {
		return Odds{}, fmt.Errorf("odds api %d: %s", resp.StatusCode, string(body))
	}

	var events []rawEvent
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&events); err != nil {
		return Odds{}, err
	}
	for _, e := range events {
		if e.HomeTeam == homeName && e.AwayTeam == awayName {
			return pickFirstH2H(e)
		}
	}
	return Odds{}, fmt.Errorf("no odds found for %s vs %s @ %s", homeName, awayName, kickoffUTC)
}
```

Add imports: `"bytes"`. The body is now read once into a buffer so wirelog can report size and the decoder can still consume it.

- [ ] **Step 4: Run odds tests**

Run: `cd backend && go test ./internal/odds/...`
Expected: PASS — including the two new wirelog tests and all existing tests.

- [ ] **Step 5: Commit**

```bash
cd backend
git add internal/odds/client.go internal/odds/client_test.go
git commit -m "feat(odds): emit HTTP wirelog around httpc.Do"
```

---

## Task 9: `fdorg.Client` — HTTP wirelog around `doRequest`

**Files:**
- Modify: `backend/internal/fdorg/client.go`
- Modify: `backend/internal/fdorg/client_test.go`

- [ ] **Step 1: Write the failing test**

Append to `backend/internal/fdorg/client_test.go`:

```go
func TestDoRequestEmitsHTTPWirelog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Requests-Available-Minute", "10")
		_, _ = w.Write([]byte(`{"competition":{"name":"WC"}}`))
	}))
	defer srv.Close()

	var buf bytes.Buffer
	prev := trace.SetWriter(&buf)
	defer trace.SetWriter(prev)

	c := NewClient(srv.URL, "k")
	_, err := c.get(t.Context(), "/v4/competitions/WC/matches")
	require.NoError(t, err)

	out := buf.String()
	require.Contains(t, out, "[wcp:fdorg] → GET "+srv.URL+"/v4/competitions/WC/matches")
	require.Contains(t, out, "[wcp:fdorg] ✓ 200 (")
}
```

Add the import: `"github.com/timhealey/world-cup-predictor/backend/internal/trace"` and `"bytes"`.

If `Client.get` is unexported, either test through a public method (e.g. `GetFinishedResults`) that hits the same path or expose a thin test helper. Adapt accordingly — the assertion content stays the same.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/fdorg/... -run TestDoRequestEmitsHTTPWirelog`
Expected: FAIL.

- [ ] **Step 3: Wire into `doRequest`**

Modify `backend/internal/fdorg/client.go`:

```go
import (
	// ... existing imports ...
	"github.com/timhealey/world-cup-predictor/backend/internal/trace"
)

// doRequest performs a single HTTP GET and returns body + status + headers.
func (c *Client) doRequest(ctx context.Context, path string) ([]byte, int, http.Header, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, nil, err
	}
	req.Header.Set("X-Auth-Token", c.apiKey)

	trace.HTTPStart("fdorg", "GET", url)
	start := time.Now()
	resp, err := c.httpc.Do(req)
	if err != nil {
		trace.HTTPError("fdorg", time.Since(start), err)
		return nil, 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	dur := time.Since(start)
	if err != nil {
		trace.HTTPError("fdorg", dur, err)
		return nil, resp.StatusCode, resp.Header, err
	}
	trace.HTTPEnd("fdorg", resp.StatusCode, dur, len(body))
	return body, resp.StatusCode, resp.Header, nil
}
```

- [ ] **Step 4: Run fdorg tests**

Run: `cd backend && go test ./internal/fdorg/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd backend
git add internal/fdorg/client.go internal/fdorg/client_test.go
git commit -m "feat(fdorg): emit HTTP wirelog around doRequest"
```

---

## Task 10: `cmd/wcp/main.go` — rewrite fetcher closures with new signatures + snippet derivation

**Files:**
- Modify: `backend/cmd/wcp/main.go`

This task closes the build break left by Task 5. There are no new tests — the integration is exercised by `internal/predict/pipeline_test.go`. We verify by running `go build ./...` and the full test suite.

- [ ] **Step 1: Update the closures**

Modify the `deps := predict.Deps{...}` block in `runPredict` (around [backend/cmd/wcp/main.go:148-171](backend/cmd/wcp/main.go#L148-L171)) to match the new `(data, err, snippet)` signatures with snippet derivation per the spec table:

```go
deps := predict.Deps{
	FetchOdds: func(ctx context.Context, h, a, k string) (any, error, string) {
		if oddsClient == nil {
			return nil, errors.New("odds client not configured (ODDS_API_KEY missing)"), ""
		}
		o, err := oddsClient.GetForMatch(ctx, h, a, k)
		if err != nil {
			return nil, err, ""
		}
		snip := fmt.Sprintf("bookmaker=%s home=%.2f away=%.2f draw=%.2f",
			o.Bookmaker, o.HomeOdds, o.AwayOdds, o.DrawOdds)
		return o, nil, snip
	},
	FetchNews: func(ctx context.Context, d any, h, a string) (fetchers.NewsResult, error, string) {
		r, err := fetchers.FetchNews(ctx, driver, h, a)
		if err != nil {
			return r, err, ""
		}
		return r, nil, firstLine(r.HomeSummary) + " / " + firstLine(r.AwaySummary)
	},
	FetchLineup: func(ctx context.Context, d any, h, a string) (fetchers.LineupResult, error, string) {
		r, err := fetchers.FetchLineup(ctx, driver, h, a)
		if err != nil {
			return r, err, ""
		}
		return r, nil, fmt.Sprintf("confirmed=%v notes=%s", r.Confirmed, truncate(r.Notes, 200))
	},
	FetchContext: func(s *store.Store, h, a string) (fetchers.ContextResult, error, string) {
		r, err := fetchers.FetchContext(s, h, a)
		if err != nil {
			return r, err, ""
		}
		return r, nil, truncate(r.TournamentContext, 200) + " / " + truncate(r.TrackRecord, 200)
	},
}
```

Add small helpers at file scope (bottom of `main.go`):

```go
// firstLine returns s up to the first newline, with leading/trailing whitespace stripped.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

// truncate returns s clipped to n bytes (UTF-8 safe — never splits a rune).
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := n
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}
```

Add imports to `main.go`: `"errors"` (if not already imported), `"strings"`, `"unicode/utf8"`.

- [ ] **Step 2: Build**

Run: `cd backend && go build ./...`
Expected: clean — no compile errors.

- [ ] **Step 3: Run full test suite**

Run: `cd backend && go test ./...`
Expected: ALL PASS.

- [ ] **Step 4: Run vet**

Run: `cd backend && go vet ./...`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
cd backend
git add cmd/wcp/main.go
git commit -m "feat(cmd): wire fetcher closures to (data, err, snippet) with per-kind snippets"
```

---

## Task 11: `store/export.go` — include `trace` in `predictions.json`

**Files:**
- Modify: `backend/internal/store/export.go`
- Modify: `backend/internal/store/store_test.go` (or create a new `export_test.go`)

- [ ] **Step 1: Write the failing test**

Add to `backend/internal/store/store_test.go`:

```go
func TestExportJSONIncludesTrace(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()
	require.NoError(t, s.UpsertTeam(Team{Code: "ARG", Name: "Argentina"}))
	require.NoError(t, s.UpsertTeam(Team{Code: "SAU", Name: "Saudi Arabia"}))
	require.NoError(t, s.UpsertMatch(Match{
		ID: "m1", HomeTeamCode: "ARG", AwayTeamCode: "SAU",
		KickoffUTC: "2026-06-25T11:00:00Z", Stage: "group",
	}))

	traceArr := `[{"kind":"odds","ok":true,"started_at":"2026-06-25T10:30:00.000Z","duration_ms":380,"error":"","snippet":"bookmaker=x"}]`
	_, err := s.InsertPrediction(Prediction{
		MatchID: "m1", CreatedAt: "x", Trigger: "on_demand", Confidence: "high",
		PredictedWinner: "ARG", PredictedScore: "2-0", WinProbability: 0.7,
		Reasoning: "r", InputsJSON: "{}", RenderedPrompt: "", ModelID: "m",
		PromptVersion: "v", TraceJSON: traceArr,
	})
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "out.json")
	require.NoError(t, s.ExportJSON(path))

	raw, _ := os.ReadFile(path)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))

	matches := payload["matches"].([]any)
	preds := matches[0].(map[string]any)["predictions"].([]any)
	pred := preds[0].(map[string]any)
	require.NotNil(t, pred["trace"])
	trace := pred["trace"].([]any)
	require.Len(t, trace, 1)
	require.Equal(t, "odds", trace[0].(map[string]any)["kind"])
}

func TestExportJSONEmitsNullTraceWhenAbsent(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()
	require.NoError(t, s.UpsertTeam(Team{Code: "ARG", Name: "Argentina"}))
	require.NoError(t, s.UpsertTeam(Team{Code: "SAU", Name: "Saudi Arabia"}))
	require.NoError(t, s.UpsertMatch(Match{
		ID: "m1", HomeTeamCode: "ARG", AwayTeamCode: "SAU",
		KickoffUTC: "2026-06-25T11:00:00Z", Stage: "group",
	}))
	_, err := s.InsertPrediction(Prediction{
		MatchID: "m1", CreatedAt: "x", Trigger: "on_demand", Confidence: "low",
		PredictedWinner: "ARG", PredictedScore: "1-0", WinProbability: 0.5,
		Reasoning: "", InputsJSON: "{}", RenderedPrompt: "", ModelID: "m",
		PromptVersion: "v", // TraceJSON empty
	})
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "out.json")
	require.NoError(t, s.ExportJSON(path))

	raw, _ := os.ReadFile(path)
	// Confirm `"trace": null` appears — we want an explicit null, not omitted.
	require.Contains(t, string(raw), `"trace": null`)
}
```

Add imports: `"encoding/json"`, `"os"`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/store/... -run TestExportJSONIncludesTrace`
Expected: FAIL.

- [ ] **Step 3: Update `ExportPrediction`**

Modify `backend/internal/store/export.go`:

```go
import "encoding/json"

type ExportPrediction struct {
	ID              int64            `json:"id"`
	CreatedAt       string           `json:"created_at"`
	Trigger         string           `json:"trigger"`
	Confidence      string           `json:"confidence"`
	PredictedWinner string           `json:"predicted_winner"`
	PredictedScore  string           `json:"predicted_score"`
	WinProbability  float64          `json:"win_probability"`
	Reasoning       string           `json:"reasoning"`
	ModelID         string           `json:"model_id"`
	Variant         string           `json:"variant"`
	Trace           *json.RawMessage `json:"trace"` // pointer so encoding/json emits null for nil
}
```

Update the loop in `ExportJSON` that builds `ExportPrediction`s:

```go
for _, p := range preds {
	ep := ExportPrediction{
		ID: p.ID, CreatedAt: p.CreatedAt, Trigger: p.Trigger,
		Confidence: p.Confidence, PredictedWinner: p.PredictedWinner,
		PredictedScore: p.PredictedScore, WinProbability: p.WinProbability,
		Reasoning: p.Reasoning, ModelID: p.ModelID, Variant: p.Variant,
	}
	if p.TraceJSON != "" {
		raw := json.RawMessage(p.TraceJSON)
		ep.Trace = &raw
	}
	em.Predictions = append(em.Predictions, ep)
}
```

This preserves the array verbatim (no re-encoding) and emits `null` when absent.

- [ ] **Step 4: Run tests**

Run: `cd backend && go test ./internal/store/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd backend
git add internal/store/export.go internal/store/store_test.go
git commit -m "feat(export): emit trace array (or null) in predictions.json"
```

---

## Task 12: Frontend types — `TraceEntry` and `trace` field on `Prediction`

**Files:**
- Modify: `frontend/src/types/api.ts`

No tests for type-only changes. The compile/typecheck step verifies.

- [ ] **Step 1: Add `TraceEntry` and extend `Prediction`**

Modify `frontend/src/types/api.ts`:

```typescript
export interface TraceEntry {
  kind: "odds" | "news" | "lineup" | "context" | "predict";
  started_at: string; // ISO 8601 with milliseconds, UTC
  duration_ms: number;
  ok: boolean;
  error: string; // empty when ok
  snippet: string; // empty when ok=false and no payload was returned
}

export interface Prediction {
  id: number;
  created_at: string;
  trigger: Trigger;
  confidence: Confidence;
  predicted_winner: string;
  predicted_score: string;
  win_probability: number;
  reasoning: string;
  model_id: string;
  variant: string;
  // Legacy predictions written before the trace column existed return null;
  // newer ones return a 5-element array in fixed order: odds, news, lineup,
  // context, predict.
  trace: TraceEntry[] | null;
}
```

- [ ] **Step 2: Typecheck**

Run: `cd frontend && npx tsc --noEmit`
Expected: clean — or, if existing code references `Prediction` in places that need a default, fix as you go. The new field is required (no `?`) since the backend always emits it (array or null). If TypeScript flags any existing test fixture missing this field, add `trace: null` to satisfy the type.

- [ ] **Step 3: Run frontend tests to confirm nothing else regressed**

Run: `cd frontend && npm test`
Expected: PASS (or PASS after adding `trace: null` to fixtures flagged by the typecheck).

- [ ] **Step 4: Commit**

```bash
cd frontend
git add src/types/api.ts
# plus any test fixture files updated to add trace: null
git commit -m "feat(types): add TraceEntry and Prediction.trace field"
```

---

## Task 13: `lib/traceFormat.ts` — pure helpers + tests

**Files:**
- Create: `frontend/src/lib/traceFormat.ts`
- Create: `frontend/tests/traceFormat.test.ts`

- [ ] **Step 1: Write the failing test**

Create `frontend/tests/traceFormat.test.ts`:

```typescript
import { describe, expect, it } from "vitest";
import { formatDuration, okCount, pillTone } from "../src/lib/traceFormat";
import type { TraceEntry } from "../src/types/api";

function entry(kind: TraceEntry["kind"], ok: boolean): TraceEntry {
  return {
    kind,
    started_at: "2026-06-25T17:30:00.000Z",
    duration_ms: 0,
    ok,
    error: ok ? "" : "x",
    snippet: "",
  };
}

describe("formatDuration", () => {
  it("renders sub-second values in ms", () => {
    expect(formatDuration(0)).toBe("0ms");
    expect(formatDuration(380)).toBe("380ms");
    expect(formatDuration(999)).toBe("999ms");
  });
  it("renders values >= 1000ms in seconds with one decimal", () => {
    expect(formatDuration(1000)).toBe("1.0s");
    expect(formatDuration(3614)).toBe("3.6s");
    expect(formatDuration(91240)).toBe("91.2s");
  });
});

describe("okCount", () => {
  it("counts ok entries", () => {
    expect(okCount([entry("odds", true), entry("news", false), entry("lineup", true), entry("context", true), entry("predict", true)])).toBe(4);
  });
  it("returns 0 for empty array", () => {
    expect(okCount([])).toBe(0);
  });
});

describe("pillTone", () => {
  it("returns 'ok' when all entries are healthy", () => {
    expect(pillTone(5, 5)).toBe("ok");
  });
  it("returns 'degraded' when any entry failed", () => {
    expect(pillTone(4, 5)).toBe("degraded");
    expect(pillTone(0, 5)).toBe("degraded");
  });
  it("returns 'degraded' for an empty or short trace as a defensive default", () => {
    expect(pillTone(0, 0)).toBe("degraded");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npm test -- traceFormat`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement helpers**

Create `frontend/src/lib/traceFormat.ts`:

```typescript
import type { TraceEntry } from "../types/api";

export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

export function okCount(entries: TraceEntry[]): number {
  let n = 0;
  for (const e of entries) if (e.ok) n++;
  return n;
}

export type PillTone = "ok" | "degraded";

// pillTone returns 'ok' only when every entry succeeded AND the trace is the
// expected length (5). Anything shorter is treated as degraded so the UI
// doesn't display a green "0/0" pill on a malformed trace.
export function pillTone(okN: number, total: number): PillTone {
  if (total === 0) return "degraded";
  return okN === total ? "ok" : "degraded";
}
```

- [ ] **Step 4: Run tests**

Run: `cd frontend && npm test -- traceFormat`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd frontend
git add src/lib/traceFormat.ts tests/traceFormat.test.ts
git commit -m "feat(frontend): add traceFormat helpers (formatDuration, okCount, pillTone)"
```

---

## Task 14: `<PredictionTrace>` component

**Files:**
- Create: `frontend/src/components/PredictionTrace.tsx`

Pure presentational component. No internal state. No component-level test (matching the existing convention where `frontend/tests/*` only covers `lib/` modules — see the click-to-expand plan's architecture note for reference).

- [ ] **Step 1: Implement the component**

Create `frontend/src/components/PredictionTrace.tsx`:

```tsx
import type { TraceEntry } from "../types/api";
import { formatDuration, okCount } from "../lib/traceFormat";

interface Props {
  trace: TraceEntry[];
  open: boolean;
  onToggle: () => void;
}

export function PredictionTrace({ trace, open, onToggle }: Props) {
  const ok = okCount(trace);
  if (!open) return null;
  return (
    <div className="mt-4 overflow-hidden rounded-md border bg-surface-sunk">
      <div className="flex items-center justify-between border-b bg-black/[0.03] px-4 py-2">
        <h5 className="m-0 text-2xs font-semibold uppercase tracking-label text-ink">
          Input trace · {ok}/{trace.length} ok
        </h5>
        <button
          type="button"
          onClick={onToggle}
          className="text-2xs font-semibold uppercase tracking-label text-ink-3 hover:text-ink focus:outline-none focus-visible:shadow-focus"
        >
          ▾ Collapse
        </button>
      </div>
      <ul className="m-0 list-none p-0">
        {trace.map((e) => (
          <li
            key={e.kind}
            className="border-b border-black/[0.06] px-4 py-3 last:border-b-0"
          >
            <div className="flex items-center justify-between">
              <span className="flex items-center gap-2 text-2xs font-semibold uppercase tracking-label text-ink">
                <span
                  className={`inline-block h-2 w-2 rounded-full ${
                    e.ok ? "bg-emerald-600" : "bg-red-600"
                  }`}
                  aria-hidden
                />
                {e.kind}
              </span>
              <span className="text-2xs tabular-nums text-ink-3">
                {e.ok ? (
                  <span className="font-semibold text-emerald-700">✓ ok</span>
                ) : (
                  <span className="font-semibold text-red-700">✗ failed</span>
                )}
                {" · "}
                {formatDuration(e.duration_ms)}
              </span>
            </div>
            {e.error && (
              <div className="mt-1 pl-4 text-xs text-red-700">{e.error}</div>
            )}
            {e.snippet && (
              <pre className="mt-1 ml-4 overflow-x-auto rounded bg-black/[0.04] px-2 py-1 text-[10.5px] leading-snug text-ink-2">
                {e.snippet}
              </pre>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}
```

- [ ] **Step 2: Verify it typechecks**

Run: `cd frontend && npx tsc --noEmit`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
cd frontend
git add src/components/PredictionTrace.tsx
git commit -m "feat(frontend): add PredictionTrace drawer component"
```

---

## Task 15: Wire triggers into `PredictionStats` + `PredictionBody`

**Files:**
- Modify: `frontend/src/components/PredictionStats.tsx`
- Modify: `frontend/src/components/PredictionBody.tsx`

- [ ] **Step 1: Extend `PredictionStats` with the `(i)` icon**

Modify `frontend/src/components/PredictionStats.tsx`. Add two optional props and render the icon when both `onTraceClick` and the prediction has a trace:

```tsx
import type { Prediction } from "../types/api";
import { okCount, pillTone } from "../lib/traceFormat";

interface Props {
  prediction: Prediction;
  teamName: (code: string) => string;
  onTraceClick?: () => void;
}

export function PredictionStats({ prediction, teamName, onTraceClick }: Props) {
  const traceAvailable = prediction.trace !== null && onTraceClick !== undefined;
  const tone =
    prediction.trace !== null
      ? pillTone(okCount(prediction.trace), prediction.trace.length)
      : "degraded";

  return (
    <div className="rounded-md bg-surface-sunk p-5 sm:p-6">
      <div className="mb-2 text-2xs font-semibold uppercase tracking-label text-ink-3">
        Predicted winner
      </div>
      <div className="font-display text-display-lg font-extrabold uppercase leading-none tracking-display text-primary">
        {teamName(prediction.predicted_winner)}
      </div>

      <div className="mt-5 grid grid-cols-2 gap-5">
        <div>
          <div className="mb-1.5 text-2xs font-semibold uppercase tracking-label text-ink-3">
            Score
          </div>
          <div className="font-display text-3xl font-extrabold leading-none text-ink">
            {prediction.predicted_score}
          </div>
        </div>
        <div>
          <div className="mb-1.5 flex items-center gap-2 text-2xs font-semibold uppercase tracking-label text-ink-3">
            Win probability
            {traceAvailable && (
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  onTraceClick!();
                }}
                aria-label="View input trace"
                title="View input trace"
                className={
                  "inline-flex h-5 w-5 items-center justify-center rounded-full border text-[10px] font-bold leading-none transition-colors " +
                  (tone === "ok"
                    ? "border-black/15 bg-black/[0.04] text-ink-3 hover:bg-black/10"
                    : "border-primary/25 bg-primary/[0.08] text-primary hover:bg-primary/15")
                }
              >
                i
              </button>
            )}
          </div>
          <div className="inline-block border-b-[3px] border-secondary pb-0.5 font-display text-3xl font-extrabold leading-none text-ink">
            {Math.round(prediction.win_probability * 100)}%
          </div>
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Add the header pill + drawer into `PredictionBody`**

Modify `frontend/src/components/PredictionBody.tsx`. Read it first to confirm the current shape — the changes are:

1. Add `useState` import and `traceOpen` state.
2. Render a pill in the header row when `pred?.trace !== null`.
3. Pass `onTraceClick={() => setTraceOpen(o => !o)}` into `<PredictionStats>`.
4. Render `<PredictionTrace>` between the stats/reasoning grid and the bottom action bar.

```tsx
import { useState } from "react";
import type { Match } from "../types/api";
import { latestPrediction } from "../lib/trackRecord";
import { flagFor } from "../data/flags";
import { formatKickoff, formatCountdown } from "../lib/format";
import { confidenceBadge } from "../lib/confidence";
import { okCount, pillTone } from "../lib/traceFormat";
import { Badge } from "./Badge";
import { Button } from "./Button";
import { Refresh, Zap } from "./icons";
import { PredictionStats } from "./PredictionStats";
import { PredictionReasoning } from "./PredictionReasoning";
import { PredictionTrace } from "./PredictionTrace";
import { ThinkingIndicator } from "./ThinkingIndicator";

interface Props {
  match: Match;
  teamName: (code: string) => string;
  variant: "dashboard" | "upcoming";
  groupLabel?: string;
  onPredict?: (matchID: string) => void;
  predictDisabled?: boolean;
  activeMatchId?: string | null;
  elapsedMs?: number;
  onCollapse?: () => void;
}

export function PredictionBody({
  match,
  teamName,
  variant,
  groupLabel,
  onPredict,
  predictDisabled,
  activeMatchId,
  elapsedMs,
  onCollapse,
}: Props) {
  const isPredictingThis = activeMatchId === match.id;
  const pred = latestPrediction(match);
  const ko = new Date(match.kickoff_utc);
  const homeName = teamName(match.home_team_code);
  const awayName = teamName(match.away_team_code);
  const [traceOpen, setTraceOpen] = useState(false);

  const traceAvailable = pred?.trace != null;
  const okN = traceAvailable ? okCount(pred!.trace!) : 0;
  const totalN = traceAvailable ? pred!.trace!.length : 0;
  const tone = traceAvailable ? pillTone(okN, totalN) : "degraded";

  return (
    <div className="wcp-reveal">
      <header className="mb-4 flex flex-wrap items-baseline justify-between gap-2">
        <div className="text-xs font-semibold uppercase tracking-label-mid text-ink-3">
          {formatKickoff(match.kickoff_utc)} · {formatCountdown(ko)}
        </div>
        <div className="flex items-center gap-3">
          {groupLabel && (
            <span className="text-xs font-semibold uppercase tracking-label text-primary">
              {groupLabel}
            </span>
          )}
          {pred && (
            <Badge tone={confidenceBadge(pred.confidence).tone}>
              {confidenceBadge(pred.confidence).label} confidence
            </Badge>
          )}
          {traceAvailable && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                setTraceOpen((o) => !o);
              }}
              className={
                "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-[3px] text-[10px] font-bold uppercase tracking-label transition-colors " +
                (tone === "ok"
                  ? "border-black/12 bg-black/[0.04] text-ink-3 hover:bg-black/10"
                  : "border-primary/22 bg-primary/[0.08] text-primary hover:bg-primary/15")
              }
              aria-expanded={traceOpen}
              aria-label={`Input trace ${okN} of ${totalN} ok`}
            >
              <span
                className={
                  "inline-block h-[5px] w-[5px] rounded-full " +
                  (tone === "ok" ? "bg-emerald-600" : "bg-primary")
                }
                aria-hidden
              />
              {okN}/{totalN} inputs
            </button>
          )}
        </div>
      </header>

      <div className="font-display text-display-lg font-extrabold uppercase leading-none tracking-display text-ink">
        {flagFor(match.home_team_code)} {homeName}
        <span className="mx-3 text-[0.55em] font-bold text-ink-4">vs</span>
        {flagFor(match.away_team_code)} {awayName}
      </div>
      {match.venue && (
        <div className="mt-2 text-sm text-ink-2">{match.venue}</div>
      )}

      {pred ? (
        <>
          <div className="mt-6 grid grid-cols-1 gap-6 md:grid-cols-2 md:gap-8">
            <PredictionStats
              prediction={pred}
              teamName={teamName}
              onTraceClick={
                traceAvailable ? () => setTraceOpen((o) => !o) : undefined
              }
            />
            <PredictionReasoning reasoning={pred.reasoning} />
          </div>
          {traceAvailable && (
            <PredictionTrace
              trace={pred.trace!}
              open={traceOpen}
              onToggle={() => setTraceOpen((o) => !o)}
            />
          )}
        </>
      ) : (
        <div className="mt-6 rounded-md border border-dashed bg-surface-sunk px-5 py-4 text-sm text-ink-2">
          No prediction yet. The scheduled launchd agent will fire at T-30, or
          you can predict now manually.
        </div>
      )}

      {(onPredict || (variant === "dashboard" && onCollapse)) && (
        <div className="mt-6 flex items-center justify-between border-t pt-4">
          <div className="flex gap-2.5">
            {onPredict &&
              (isPredictingThis ? (
                <span className="px-4 py-2 text-sm font-semibold text-ink-2">
                  <ThinkingIndicator elapsedMs={elapsedMs} />
                </span>
              ) : (
                <Button
                  variant={pred ? "ghost" : "primary"}
                  disabled={predictDisabled}
                  onClick={(e) => {
                    e.stopPropagation();
                    onPredict(match.id);
                  }}
                >
                  {pred ? (
                    <>
                      <Refresh /> Re-predict
                    </>
                  ) : (
                    <>
                      <Zap /> Predict now
                    </>
                  )}
                </Button>
              ))}
          </div>
          {variant === "dashboard" && onCollapse && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                onCollapse();
              }}
              className="text-2xs font-semibold uppercase tracking-label text-ink-3 hover:text-ink focus:outline-none focus-visible:shadow-focus"
            >
              ▴ Collapse
            </button>
          )}
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 3: Typecheck and run frontend tests**

Run: `cd frontend && npx tsc --noEmit && npm test`
Expected: clean typecheck, all tests pass.

- [ ] **Step 4: Manual smoke test in dev**

Run: `cd frontend && npm run dev` (in one terminal). In another, ensure `predictions.json` has at least one prediction with a non-null `trace` field — easiest way: run `cd backend && go run ./cmd/wcp predict --match <some-match-id>` after Tasks 1–11 are merged.

Visit the dashboard:
- Find a prediction card. Confirm the pill renders in the header next to the confidence badge.
- For a prediction with 5/5 ok inputs, the pill is grey; for one with failures, it's orange.
- Click the pill — the accordion expands below the stats/reasoning grid.
- Click the `(i)` icon next to "Win probability" — same accordion toggles.
- Confirm row status dots, durations, error messages, and snippet blocks render as designed.
- Find a legacy prediction without a `trace` field (or set `trace: null` in `predictions.json` manually for one row) — confirm neither the pill nor the icon renders.

- [ ] **Step 5: Commit**

```bash
cd frontend
git add src/components/PredictionStats.tsx src/components/PredictionBody.tsx
git commit -m "feat(frontend): add trace pill + (i) icon triggers and inline drawer"
```

---

## Task 16: Update `CLAUDE.md` "Adding a new fetcher" section

**Files:**
- Modify: `CLAUDE.md`

- [ ] **Step 1: Find and update the section**

Open `CLAUDE.md` and locate the section `## Adding a new fetcher`. Replace the existing checklist with:

```markdown
## Adding a new fetcher

1. Create `backend/internal/<name>/<name>.go` with the client struct and method (e.g. `GetForMatch(ctx, ...)`).
2. Add a fixture under `backend/testdata/` if it has a wire format.
3. Add tests in `<name>_test.go` using `httptest.NewServer` against the fixture.
4. If the fetcher captures rate-limit info, call `ratelimit.Record<Source>(...)` after each request.
5. The HTTP or subprocess call site must emit wire-level logs via `internal/trace`. For HTTP: wrap `httpc.Do` with `trace.HTTPStart` / `trace.HTTPEnd` / `trace.HTTPError`. For `claude -p`: use `trace.SubprocessStart` / `trace.SubprocessEnd` / `trace.SubprocessError`. Pass a short namespace string that matches the fetcher's trace `kind` (e.g. `"odds"`, `"news"`).
6. Wire it into `backend/internal/predict/pipeline.go::Deps` as a new field with signature `(data, err, snippet)`. The closure in `cmd/wcp/main.go::runPredict` must produce a short, human-readable snippet on success (≤400 chars — truncation is handled by `trace.Recorder`); return `""` on failure and let the error string carry the diagnostic. See `internal/trace/recorder.go` for the snippet conventions used by the existing fetchers.
7. If the fetcher's failure should affect the confidence flag, extend `predict.Inputs` and `predict.Confidence`.
8. Extend `internal/trace.kinds` to include the new fetcher's kind so the trace array picks up a slot for it. If the fetcher is conditional (only runs in some flows), decide whether its absence reads as `ok: false` with a specific `error` string, or whether you skip the `Start` call entirely (which surfaces as `error: "not run"` in the trace). Adjust the frontend's `N/total` pill expectations accordingly.
9. Inject a fake into `pipeline_test.go` with the new `(data, err, snippet)` signature.
```

- [ ] **Step 2: Verify the file renders cleanly**

Run: `cat CLAUDE.md | head -200 | tail -50` (or open in the IDE) and confirm the formatting matches the surrounding sections.

- [ ] **Step 3: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: update Adding a new fetcher checklist with tracing steps"
```

---

## Final verification

After all 16 tasks land:

- [ ] **Run full backend test suite**

Run: `cd backend && go test ./...`
Expected: ALL PASS.

- [ ] **Run vet**

Run: `cd backend && go vet ./...`
Expected: clean.

- [ ] **Run full frontend test suite**

Run: `cd frontend && npm test && npx tsc --noEmit`
Expected: PASS.

- [ ] **Manual end-to-end check**

```bash
cd backend
go build ./... && ./bin/wcp predict --match <upcoming-match-id> 2>&1 | tee /tmp/predict.log
```

Then:
- Inspect `/tmp/predict.log` and confirm the `[wcp:odds]`, `[wcp:news]`, `[wcp:lineup]`, `[wcp:predict]`, and `[wcp:trace]` lines appear in plausible order.
- Open the SQLite DB: `sqlite3 backend/wcp.db 'SELECT trace_json FROM predictions ORDER BY id DESC LIMIT 1;'` — confirm a 5-element JSON array.
- Check `backend/predictions.json` — confirm the newest prediction has a `trace` array (not null).
- Open the dashboard, find that prediction, verify the pill, the `(i)` icon, and the drawer all render and toggle correctly.
