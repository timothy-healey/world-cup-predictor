# Reject post-kickoff predictions — Design

**Date:** 2026-06-12
**Status:** Draft (pending user review)

## Purpose

A prediction is only meaningful before a match starts. Today, nothing in `predict.Pipeline.Run` checks whether the match has already kicked off — so a late-firing launchd job, a manual `wcp predict --match X` invocation hours after the final whistle, or a pipeline that overruns kickoff by a few minutes can all produce a "prediction" row that taints the track record. This spec adds a strict pre-kickoff guard so no `predictions` row is ever written for a match whose `KickoffUTC` has elapsed.

## Goals

- Refuse to start the prediction pipeline once kickoff has passed (`now >= kickoff`).
- Refuse to persist a prediction whose pipeline finished at or after kickoff, even if it started earlier.
- Apply the rule universally — manual and scheduled invocations alike.
- Leave no side effects on rejection: no DB row, no `predictions.json` rewrite, no email.
- Surface the rejection clearly via the existing CLI error path (`wcp: <err>` to stderr, exit 1) so launchd's per-match log file at `~/Library/Logs/wcp/<match>.log` captures it.

## Non-goals

- No catch-up / missed-prediction recovery (out of scope — separate feature if ever needed).
- No "skipped" placeholder row or new table to track rejections. The launchd log file is the audit trail.
- No `--allow-late` escape hatch. YAGNI — the future ablation harness replays from stored `inputs_json` via a different code path and won't trip this guard.
- No change to launchd plist scheduling (still T-30, still `RunAtLoad=false`).
- No buffer (e.g., reject at T-5). The boundary is exactly kickoff.

## Architecture

### New sentinel error

`internal/predict` exports:

```go
var ErrPredictionPastKickoff = errors.New("predict: kickoff already elapsed")
```

Wrapped at call sites with `fmt.Errorf("%w: match=%s kickoff=%s now=%s", ErrPredictionPastKickoff, matchID, kickoffISO, nowISO)` so the surfaced error string carries diagnostic context while remaining `errors.Is`-able.

### Clock injection

`Pipeline` gains an unexported `nowFn func() time.Time` field, defaulting to `time.Now` in `New(...)`. Tests override it via a test-only exported setter declared in a sibling `export_test.go` file:

```go
// export_test.go
package predict
func (p *Pipeline) SetNowFn(fn func() time.Time) { p.nowFn = fn }
```

This keeps the public `New` signature unchanged and prevents non-test code from poking at the clock, since `export_test.go` is compiled only under `go test`.

### Check 1 — pipeline start

In `Pipeline.Run`, immediately after `GetMatch` returns and before any goroutines or recorder calls:

```go
kickoff, err := time.Parse(time.RFC3339, m.KickoffUTC)
if err != nil {
    return store.Prediction{}, fmt.Errorf("parse kickoff %q: %w", m.KickoffUTC, err)
}
if !p.nowFn().Before(kickoff) {
    return store.Prediction{}, fmt.Errorf("%w: match=%s kickoff=%s now=%s",
        ErrPredictionPastKickoff, matchID, kickoff.Format(time.RFC3339), p.nowFn().UTC().Format(time.RFC3339))
}
```

Note: `!now.Before(kickoff)` is the correct expression for `now >= kickoff` — strict boundary, exactly-at-kickoff rejects.

### Check 2 — persist time

Immediately before `p.store.InsertPrediction(pred)`:

```go
if !p.nowFn().Before(kickoff) {
    return store.Prediction{}, fmt.Errorf("%w: match=%s kickoff=%s now=%s ...",
        ErrPredictionPastKickoff, matchID, kickoff.Format(time.RFC3339), p.nowFn().UTC().Format(time.RFC3339))
}
```

The kickoff time parsed at Check 1 is reused. No second `time.Parse` call.

### Caller side (`cmd/wcp/main.go::runPredict`)

No code change required. The existing pattern is:

```go
rec, err := pipeline.Run(ctx, matchID, trigger)
if err != nil {
    return err
}
```

The dispatcher's `main()` prints `wcp: <err>` to stderr and exits 1. `ExportJSON` and any email logic sit *after* the early return, so they're naturally skipped on rejection.

## Behavior matrix

| Scenario | Result |
|---|---|
| Manual `wcp predict --match X`, X kicked off 1h ago | Reject at start. No Claude call. No DB write. Exit 1. |
| launchd fires at T-30 as expected, finishes T-15 | Both checks pass. Normal write. |
| launchd fires at T-30, Claude call returns at T+2 | Start check passes; persist check rejects. DB unchanged. Exit 1. Claude cost burned. |
| Laptop powered off through T-30, booted at T+1h | launchd does not replay missed `StartCalendarInterval` (no code change addresses this — by design, out of scope). |
| Match kickoff exactly equals `now` | Reject (strict `>=`). |
| `m.KickoffUTC` unparseable | Return parse error before either kickoff check. (Should never happen — bootstrap always writes RFC3339.) |

## Testing

All tests in `backend/internal/predict/pipeline_test.go`. Pattern: inject a clock via `nowFn`, use a stub `claudec.Driver` that tracks invocation count, use stub `Deps` fetchers.

1. **`TestRun_RejectsAtStartWhenPastKickoff`**
   - Setup: store has match with `KickoffUTC = nowFn() - 1h`. Stub Claude driver with a counter.
   - Action: `pipeline.Run(ctx, matchID, "on_demand")`.
   - Assert: `errors.Is(err, ErrPredictionPastKickoff)`, Claude call count == 0, no row in `predictions` for `matchID`.

2. **`TestRun_RejectsAtStartWhenExactlyKickoff`**
   - Setup: `KickoffUTC == nowFn()`.
   - Assert same as test 1 — proves strict boundary.

3. **`TestRun_RejectsAtPersistWhenPipelineOverruns`**
   - Setup: `nowFn` is a stateful closure that returns `kickoff - 1s` on the first call (start check) and `kickoff + 1s` on the second call (persist check). Stub Claude driver succeeds normally.
   - Assert: `errors.Is(err, ErrPredictionPastKickoff)`, Claude call count == 1 (it ran), no row in `predictions`.

4. **`TestRun_HappyPathStillSucceeds` (regression guard)**
   - Setup: `KickoffUTC = nowFn() + 1h`. Stub Claude driver succeeds.
   - Assert: no error, one row in `predictions`, `Variant == "full"`.

No integration test needed — the launchd plumbing and CLI dispatcher are unchanged.

## Open questions

None.

## Impact

- **Files changed:** `backend/internal/predict/pipeline.go` (checks + clock injection), `backend/internal/predict/pipeline_test.go` (new tests). No other packages affected.
- **DB schema:** unchanged.
- **Frontend:** unchanged. Rejected runs simply leave the existing `predictions` rows / `predictions.json` untouched.
- **Operational:** launchd log files at `~/Library/Logs/wcp/<match>.log` will contain a `wcp: predict: kickoff already elapsed: ...` line for any rejection. That's the audit trail.
