# Reject post-kickoff predictions — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refuse to produce or persist a prediction once a match's `KickoffUTC` has elapsed. Applies universally — manual `wcp predict --match X` and scheduled launchd jobs alike.

**Architecture:** Two strict checks inside `predict.Pipeline.Run` — one immediately after `GetMatch` (rejects before any fetcher or Claude call) and one immediately before `store.InsertPrediction` (rejects after a pipeline that overran kickoff). Both use an injectable `nowFn` so tests can pin the clock. A package sentinel `ErrPredictionPastKickoff` makes the rejection identifiable via `errors.Is`.

**Tech Stack:** Go, SQLite (via existing `store` package), testify/require for assertions, `text/template` and `time` from stdlib.

**Spec:** [docs/superpowers/specs/2026-06-12-reject-post-kickoff-predictions-design.md](../specs/2026-06-12-reject-post-kickoff-predictions-design.md)

---

## File Structure

- **Modify** `backend/internal/predict/pipeline.go` — add `nowFn` field, `ErrPredictionPastKickoff` sentinel, two boundary checks. ~30 lines added.
- **Create** `backend/internal/predict/export_test.go` — test-only setter for `nowFn`. ~5 lines. Compiled only under `go test`.
- **Modify** `backend/internal/predict/pipeline_test.go` — pin the clock in the two existing tests; add three new tests covering the rejection paths.

No other packages touched. No DB schema change. No frontend change.

---

## Task 1: Add `nowFn` clock injection scaffolding (no behavior change)

**Files:**
- Modify: `backend/internal/predict/pipeline.go`
- Create: `backend/internal/predict/export_test.go`
- Modify: `backend/internal/predict/pipeline_test.go`

This task introduces the clock plumbing **with no behavior change**. After it, all existing tests still pass exactly as before; the only difference is that tests now pin the clock instead of relying on real `time.Now`.

- [ ] **Step 1: Add `nowFn` field to `Pipeline` and default in `New`**

Edit `backend/internal/predict/pipeline.go`. The `Pipeline` struct currently looks like:

```go
type Pipeline struct {
	store         *store.Store
	claude        *claudec.Driver
	deps          Deps
	systemPrompt  string
	promptVersion string
}

func New(s *store.Store, d *claudec.Driver, deps Deps) *Pipeline {
	sysPrompt, version := loadSystemPrompt()
	return &Pipeline{store: s, claude: d, deps: deps, systemPrompt: sysPrompt, promptVersion: version}
}
```

Change it to:

```go
type Pipeline struct {
	store         *store.Store
	claude        *claudec.Driver
	deps          Deps
	systemPrompt  string
	promptVersion string
	nowFn         func() time.Time
}

func New(s *store.Store, d *claudec.Driver, deps Deps) *Pipeline {
	sysPrompt, version := loadSystemPrompt()
	return &Pipeline{store: s, claude: d, deps: deps, systemPrompt: sysPrompt, promptVersion: version, nowFn: time.Now}
}
```

`time` is already imported.

- [ ] **Step 2: Create `export_test.go` with the test-only setter**

Create `backend/internal/predict/export_test.go` with this exact content:

```go
package predict

import "time"

// SetNowFn replaces the pipeline's clock for tests. Defined in *_test.go
// so it is compiled only under `go test` — production callers cannot see it.
func (p *Pipeline) SetNowFn(fn func() time.Time) { p.nowFn = fn }
```

- [ ] **Step 3: Pin the clock in existing tests**

In `backend/internal/predict/pipeline_test.go`, both `TestRunHappyPath` and `TestRunPartialFailureLowersConfidenceAndRecordsErrors` use a hardcoded `KickoffUTC: "2026-06-25T11:00:00Z"`. With the guard landing in Task 2, these tests must pin "now" to a time before that kickoff so they remain green forever (not just until 2026-06-25 passes).

For **both** tests, immediately after `pipeline := New(...)` and before `pipeline.Run(...)`, add:

```go
pipeline.SetNowFn(func() time.Time {
	return time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC) // 1h before kickoff
})
```

You will need to add `"time"` to the imports of `pipeline_test.go` if it is not already imported.

- [ ] **Step 4: Run existing tests to verify behavior is unchanged**

Run: `cd backend && go test ./internal/predict/...`
Expected: `ok  github.com/timhealey/world-cup-predictor/backend/internal/predict  ...` (PASS for both existing tests)

- [ ] **Step 5: Commit**

```bash
git add backend/internal/predict/pipeline.go backend/internal/predict/export_test.go backend/internal/predict/pipeline_test.go
git commit -m "$(cat <<'EOF'
refactor(predict): inject clock via nowFn for testability

No behavior change. Adds Pipeline.nowFn (defaulting to time.Now) and a
test-only SetNowFn setter. Existing tests pin the clock to a time before
their hardcoded kickoff so they stay green regardless of wall-clock date.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Reject at pipeline start when kickoff has elapsed (TDD)

**Files:**
- Modify: `backend/internal/predict/pipeline.go`
- Modify: `backend/internal/predict/pipeline_test.go`

- [ ] **Step 1: Write the failing test**

Append to `backend/internal/predict/pipeline_test.go`:

```go
func TestRun_RejectsAtStartWhenPastKickoff(t *testing.T) {
	s, _ := store.Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()
	require.NoError(t, s.UpsertTeam(store.Team{Code: "ARG", Name: "Argentina"}))
	require.NoError(t, s.UpsertTeam(store.Team{Code: "SAU", Name: "Saudi Arabia"}))
	require.NoError(t, s.UpsertMatch(store.Match{
		ID: "m1", HomeTeamCode: "ARG", AwayTeamCode: "SAU",
		KickoffUTC: "2026-06-25T11:00:00Z", Stage: "group",
	}))

	tmp := t.TempDir()
	// Claude script that records each invocation so we can assert it was NOT called.
	callLog := filepath.Join(tmp, "calls.log")
	fake := filepath.Join(tmp, "claude")
	require.NoError(t, os.WriteFile(fake, []byte("#!/bin/sh\necho called >> "+callLog+"\n"), 0o755))

	d := claudec.NewDriver(fake, "test-model")
	pipeline := New(s, d, Deps{
		FetchOdds: func(ctx context.Context, h, a, k string) (any, error, string) {
			return nil, nil, ""
		},
		FetchNews: func(ctx context.Context, d any, h, a string) (fetchers.NewsResult, error, string) {
			return fetchers.NewsResult{}, nil, ""
		},
		FetchLineup: func(ctx context.Context, d any, h, a string) (fetchers.LineupResult, error, string) {
			return fetchers.LineupResult{}, nil, ""
		},
		FetchContext: func(s *store.Store, h, a string) (fetchers.ContextResult, error, string) {
			return fetchers.ContextResult{}, nil, ""
		},
	})
	// Pin "now" to 1 hour after kickoff.
	pipeline.SetNowFn(func() time.Time {
		return time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	})

	_, err := pipeline.Run(context.Background(), "m1", "on_demand")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrPredictionPastKickoff)

	// Claude must not have been invoked.
	_, statErr := os.Stat(callLog)
	require.True(t, os.IsNotExist(statErr), "claude was called but should not have been")

	// No prediction row should exist for the match.
	preds, err := s.ListPredictionsByMatch("m1")
	require.NoError(t, err)
	require.Len(t, preds, 0)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd backend && go test ./internal/predict/ -run TestRun_RejectsAtStartWhenPastKickoff -v`
Expected: FAIL — most likely with a compile error `undefined: ErrPredictionPastKickoff`, or (if you stub the symbol) a runtime failure where `err` is `nil` because no guard exists yet.

- [ ] **Step 3: Add the sentinel and the start-of-pipeline check**

Edit `backend/internal/predict/pipeline.go`.

First, add `"errors"` to the import block if not already imported.

Second, add the sentinel near the top of the file, just below the `systemPromptBytes` `//go:embed` block:

```go
// ErrPredictionPastKickoff is returned by Pipeline.Run when the match's
// kickoff has already elapsed. Predictions are only meaningful before
// kickoff; this guard prevents post-hoc rows from polluting track record.
var ErrPredictionPastKickoff = errors.New("predict: kickoff already elapsed")
```

Third, add the start-of-pipeline check inside `Pipeline.Run`. The current code is:

```go
func (p *Pipeline) Run(ctx context.Context, matchID, trigger string) (store.Prediction, error) {
	m, err := p.store.GetMatch(matchID)
	if err != nil {
		return store.Prediction{}, fmt.Errorf("get match: %w", err)
	}
	home, _ := p.store.GetTeam(m.HomeTeamCode)
	away, _ := p.store.GetTeam(m.AwayTeamCode)
```

Change it to:

```go
func (p *Pipeline) Run(ctx context.Context, matchID, trigger string) (store.Prediction, error) {
	m, err := p.store.GetMatch(matchID)
	if err != nil {
		return store.Prediction{}, fmt.Errorf("get match: %w", err)
	}
	kickoff, err := time.Parse(time.RFC3339, m.KickoffUTC)
	if err != nil {
		return store.Prediction{}, fmt.Errorf("parse kickoff %q: %w", m.KickoffUTC, err)
	}
	if !p.nowFn().Before(kickoff) {
		return store.Prediction{}, fmt.Errorf("%w: match=%s kickoff=%s now=%s",
			ErrPredictionPastKickoff, matchID,
			kickoff.UTC().Format(time.RFC3339),
			p.nowFn().UTC().Format(time.RFC3339))
	}
	home, _ := p.store.GetTeam(m.HomeTeamCode)
	away, _ := p.store.GetTeam(m.AwayTeamCode)
```

Note: `!now.Before(kickoff)` is the correct expression for `now >= kickoff` (strict — exactly-at-kickoff rejects).

- [ ] **Step 4: Run the new test to verify it passes**

Run: `cd backend && go test ./internal/predict/ -run TestRun_RejectsAtStartWhenPastKickoff -v`
Expected: PASS

- [ ] **Step 5: Run the full predict test suite to verify no regressions**

Run: `cd backend && go test ./internal/predict/...`
Expected: PASS for all tests including the two pre-existing ones (`TestRunHappyPath`, `TestRunPartialFailureLowersConfidenceAndRecordsErrors`).

- [ ] **Step 6: Run `go vet` for the whole module**

Run: `cd backend && go vet ./...`
Expected: no output, exit 0.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/predict/pipeline.go backend/internal/predict/pipeline_test.go
git commit -m "$(cat <<'EOF'
feat(predict): reject pipeline start when kickoff has elapsed

Adds ErrPredictionPastKickoff sentinel and a guard immediately after
GetMatch. Universal scope — manual and scheduled invocations alike.
No fetcher or Claude call runs once kickoff has passed.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Add at-kickoff boundary regression guard (test-only, no code change)

**Files:**
- Modify: `backend/internal/predict/pipeline_test.go`

This task proves the strict-boundary semantics (`now == kickoff` rejects) and locks it down so a future change to `>` instead of `>=` won't silently regress.

- [ ] **Step 1: Write the test**

Append to `backend/internal/predict/pipeline_test.go`:

```go
func TestRun_RejectsAtStartWhenExactlyKickoff(t *testing.T) {
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
	require.NoError(t, os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0o755))

	d := claudec.NewDriver(fake, "test-model")
	pipeline := New(s, d, Deps{
		FetchOdds: func(ctx context.Context, h, a, k string) (any, error, string) {
			return nil, nil, ""
		},
		FetchNews: func(ctx context.Context, d any, h, a string) (fetchers.NewsResult, error, string) {
			return fetchers.NewsResult{}, nil, ""
		},
		FetchLineup: func(ctx context.Context, d any, h, a string) (fetchers.LineupResult, error, string) {
			return fetchers.LineupResult{}, nil, ""
		},
		FetchContext: func(s *store.Store, h, a string) (fetchers.ContextResult, error, string) {
			return fetchers.ContextResult{}, nil, ""
		},
	})
	// Pin "now" to exactly kickoff.
	pipeline.SetNowFn(func() time.Time {
		return time.Date(2026, 6, 25, 11, 0, 0, 0, time.UTC)
	})

	_, err := pipeline.Run(context.Background(), "m1", "on_demand")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrPredictionPastKickoff)
}
```

- [ ] **Step 2: Run the test to verify it passes**

Run: `cd backend && go test ./internal/predict/ -run TestRun_RejectsAtStartWhenExactlyKickoff -v`
Expected: PASS (the `!Before` check from Task 2 already enforces this boundary).

- [ ] **Step 3: Commit**

```bash
git add backend/internal/predict/pipeline_test.go
git commit -m "$(cat <<'EOF'
test(predict): lock strict at-kickoff boundary

Regression guard ensuring now==kickoff rejects (not just now>kickoff).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Reject at persist time when pipeline overruns kickoff (TDD)

**Files:**
- Modify: `backend/internal/predict/pipeline.go`
- Modify: `backend/internal/predict/pipeline_test.go`

Covers the case where the pipeline started before kickoff but finished after — e.g., Claude took 6 minutes when launchd fired at T-5.

- [ ] **Step 1: Write the failing test**

Append to `backend/internal/predict/pipeline_test.go`:

```go
func TestRun_RejectsAtPersistWhenPipelineOverruns(t *testing.T) {
	s, _ := store.Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()
	require.NoError(t, s.UpsertTeam(store.Team{Code: "ARG", Name: "Argentina"}))
	require.NoError(t, s.UpsertTeam(store.Team{Code: "SAU", Name: "Saudi Arabia"}))
	require.NoError(t, s.UpsertMatch(store.Match{
		ID: "m1", HomeTeamCode: "ARG", AwayTeamCode: "SAU",
		KickoffUTC: "2026-06-25T11:00:00Z", Stage: "group",
	}))

	tmp := t.TempDir()
	callLog := filepath.Join(tmp, "calls.log")
	fake := filepath.Join(tmp, "claude")
	// Real Claude reply so the pipeline gets past the predict step and
	// reaches the persist-time check.
	require.NoError(t, os.WriteFile(fake, []byte(`#!/bin/sh
echo called >> `+callLog+`
cat <<'EOF'
{"winner":"ARG","predicted_score":"2-0","win_probability":0.71,"reasoning":["a"]}
EOF
`), 0o755))

	d := claudec.NewDriver(fake, "test-model")
	pipeline := New(s, d, Deps{
		FetchOdds: func(ctx context.Context, h, a, k string) (any, error, string) {
			return map[string]float64{"home": 1.4}, nil, "ok"
		},
		FetchNews: func(ctx context.Context, d any, h, a string) (fetchers.NewsResult, error, string) {
			return fetchers.NewsResult{HomeSummary: "h", AwaySummary: "a"}, nil, "h/a"
		},
		FetchLineup: func(ctx context.Context, d any, h, a string) (fetchers.LineupResult, error, string) {
			return fetchers.LineupResult{Confirmed: true}, nil, "ok"
		},
		FetchContext: func(s *store.Store, h, a string) (fetchers.ContextResult, error, string) {
			return fetchers.ContextResult{}, nil, "ok"
		},
	})

	// Clock that flips past kickoff after the first call. The start check
	// (first nowFn call) sees pre-kickoff; the persist check (last nowFn
	// call) sees post-kickoff.
	calls := 0
	beforeKickoff := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	afterKickoff := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	pipeline.SetNowFn(func() time.Time {
		calls++
		if calls <= 2 { // first 2 calls = start-check (uses nowFn twice: comparison + error formatting on failure path)
			return beforeKickoff
		}
		return afterKickoff
	})

	_, err := pipeline.Run(context.Background(), "m1", "on_demand")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrPredictionPastKickoff)

	// Claude WAS called on this path — the pipeline ran end-to-end.
	data, statErr := os.ReadFile(callLog)
	require.NoError(t, statErr)
	require.Contains(t, string(data), "called")

	// But no prediction row was written.
	preds, err := s.ListPredictionsByMatch("m1")
	require.NoError(t, err)
	require.Len(t, preds, 0)
}
```

Note on `calls <= 2`: the start check on the happy path calls `nowFn` exactly once (the `Before` comparison passes, so the error-formatting branch is not taken). Setting the threshold to `<= 2` leaves a one-call margin in case the implementation changes. If the check turns out to call `nowFn` more than twice before reaching persist, raise the threshold accordingly — but the simpler form `calls <= 1` works for the implementation specified in Task 2.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd backend && go test ./internal/predict/ -run TestRun_RejectsAtPersistWhenPipelineOverruns -v`
Expected: FAIL — the test returns no error because no persist-time check exists yet. A row gets written.

- [ ] **Step 3: Add the persist-time check in `pipeline.go`**

Find this block in `pipeline.go`:

```go
	pred := store.Prediction{
		MatchID:         matchID,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
		// ... other fields
		TraceJSON:       string(traceBytes),
	}
	id, err := p.store.InsertPrediction(pred)
```

Insert the check immediately before `InsertPrediction`:

```go
	pred := store.Prediction{
		MatchID:         matchID,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
		// ... other fields unchanged
		TraceJSON:       string(traceBytes),
	}
	if !p.nowFn().Before(kickoff) {
		return store.Prediction{}, fmt.Errorf("%w: match=%s kickoff=%s now=%s (pipeline overran)",
			ErrPredictionPastKickoff, matchID,
			kickoff.UTC().Format(time.RFC3339),
			p.nowFn().UTC().Format(time.RFC3339))
	}
	id, err := p.store.InsertPrediction(pred)
```

The `kickoff` variable is in scope from Task 2's start-check block (defined at the top of `Run` after `GetMatch`).

- [ ] **Step 4: Run the new test to verify it passes**

Run: `cd backend && go test ./internal/predict/ -run TestRun_RejectsAtPersistWhenPipelineOverruns -v`
Expected: PASS

- [ ] **Step 5: Run the full predict test suite for regressions**

Run: `cd backend && go test ./internal/predict/...`
Expected: PASS for all five tests (`TestRunHappyPath`, `TestRunPartialFailureLowersConfidenceAndRecordsErrors`, `TestRun_RejectsAtStartWhenPastKickoff`, `TestRun_RejectsAtStartWhenExactlyKickoff`, `TestRun_RejectsAtPersistWhenPipelineOverruns`).

- [ ] **Step 6: Run the entire backend test suite**

Run: `cd backend && go test ./...`
Expected: all packages PASS.

- [ ] **Step 7: Run `go vet`**

Run: `cd backend && go vet ./...`
Expected: no output.

- [ ] **Step 8: Commit**

```bash
git add backend/internal/predict/pipeline.go backend/internal/predict/pipeline_test.go
git commit -m "$(cat <<'EOF'
feat(predict): reject persist when pipeline overruns kickoff

Re-checks now>=kickoff immediately before InsertPrediction so a
pipeline that started before kickoff but finished after it does not
write a tainted row. The Claude call is still made on this path — we
trade the cost for a guarantee.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Final verification

After Task 4 completes:

- [ ] **Build the binary**

Run: `cd backend && make build`
Expected: `bin/wcp` produced, no errors.

- [ ] **Smoke-test the doctor command** (sanity check that the binary still works end-to-end)

Run: `cd backend && ./bin/wcp doctor`
Expected: doctor output prints normally; no panics, no missing-symbol errors.

---

## Notes for the implementer

- The spec covers exactly what's in this plan and no more. If you find yourself wanting to add a `--allow-late` flag, an "skipped" DB record, or any catch-up logic, stop — those are out of scope per the spec's non-goals.
- `!now.Before(kickoff)` is intentional (== rejects). Don't "simplify" to `now.After(kickoff)` — that flips the boundary semantics.
- The plan uses `2026-06-25T11:00:00Z` consistently in tests because that's what the existing tests use. Don't rename or change this; keeping it consistent makes the tests easier to read side-by-side.
