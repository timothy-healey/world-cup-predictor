# Prediction input tracing — Design

**Date:** 2026-06-11
**Status:** Draft (pending user review)

## Purpose

Make it easy to debug *why* a given prediction landed with degraded inputs. Today, when a fetcher fails, the failure is collapsed to a boolean (`ok=false`) in `cmd/wcp/main.go` and the reason is lost — leading to predictions like *"No odds, news, or confirmed XI available — confidence is materially reduced"* with no trail back to whether the odds API 429'd, the news prompt timed out, or claude returned malformed JSON.

This spec adds two synchronized debug surfaces:
- **Persistent per-fetcher trace** on each prediction (stored in SQLite, exported to `predictions.json`, surfaced in the dashboard).
- **Low-level stderr logs** for every HTTP request and `claude -p` subprocess, captured by the existing launchd log file.

## Goals

- For any past prediction, the user can see — without leaving the dashboard — which of the five fetchers (`odds`, `news`, `lineup`, `context`, `predict`) succeeded, how long each took, and the error / truncated response for each.
- For any active or recent run, stderr in `~/Library/Logs/wcp/<match-id>.log` contains a wire-level account of every external call (HTTP method/URL/status/latency, claude prompt size + duration).
- The dashboard's prediction view stays quiet when all five inputs are healthy and surfaces a single, restrained signal when they aren't.
- Adding the trace does not change the `(data, ok)` contract of the existing fetcher functions.

## Non-goals

- No new `prediction_traces` table. A single `trace_json` column is enough until cross-prediction queries are needed.
- No persistence of bootstrap-time or results-time HTTP calls in SQLite (they get stderr-only treatment).
- No retry counts, no rate-limit headers, and no full request bodies in the trace. The error string and a truncated response snippet carry the cause.
- No CLI subcommand to render past traces. `sqlite3 wcp.db "select trace_json from predictions where id = ..."` is enough.
- No HTTP-level call nesting under fetchers in the DB. The stderr log carries that detail; the DB stays at fetcher granularity.

## Architecture

### Data model

Add column to `predictions`:

```sql
ALTER TABLE predictions ADD COLUMN trace_json TEXT;
```

Applied idempotently inside `store.Open` (same pattern used for `predictions.variant`). NULL for any prediction written before this change.

The column holds a JSON array of exactly five entries, in fixed order: `odds`, `news`, `lineup`, `context`, `predict`. Fixed order means there is no concept of a "missing" entry — every prediction has all five rows.

Each entry:

```json
{
  "kind": "odds",
  "started_at": "2026-06-25T17:30:00.000Z",
  "duration_ms": 380,
  "ok": false,
  "error": "no odds found for Argentina vs Saudi Arabia",
  "snippet": ""
}
```

- `kind` — one of `odds`, `news`, `lineup`, `context`, `predict`.
- `started_at` — RFC3339 with milliseconds, UTC.
- `duration_ms` — integer wall-clock milliseconds from start to finish.
- `ok` — true iff the underlying call returned a usable result.
- `error` — non-empty only when `ok` is false; the raw error string from the fetcher (already includes URL/status/stderr context for HTTP and subprocess errors).
- `snippet` — a human-readable preview of the response, truncated to **400 characters** (UTF-8 safe). Empty when the call failed before producing any output.

The `predict` entry covers the main `claude.Predict` call that produces the prediction itself (not a fetcher in the existing terminology, but conceptually the fifth external call worth tracing). When `predict.ok` is false the prediction row itself would not exist (the pipeline errors out earlier), so in practice `predict.ok` is always true when a trace row is written — it is kept in the array for symmetry and to capture timing.

### Backend instrumentation

New package `internal/trace`:

```go
type Recorder struct { /* unexported state */ }

func New() *Recorder
func (r *Recorder) Start(kind string)
func (r *Recorder) Finish(kind string, err error, snippet string)
func (r *Recorder) JSON() ([]byte, error)   // serialize to trace_json
```

- `Start` records `started_at` for the given kind.
- `Finish` records `duration_ms`, sets `ok = err == nil`, fills `error` from `err.Error()`, and truncates `snippet` to 400 chars.
- Calling `Finish` for a kind that was never started panics — fetcher closures must always call both. Calling `Finish` twice for the same kind also panics.
- `JSON()` returns the five entries in fixed order. If any kind was never started, its entry is emitted with `ok: false, error: "not run"` and zero duration — a defensive default; in normal flow all five are always started.

Wiring (in `predict.Pipeline.Run`):

The existing `Deps` fetcher signatures return `(data, ok)`. Change them to `(data, err, snippet)` — `err == nil` is the new `ok`, and `snippet` is a short human-readable preview the closure produces from whatever the fetcher returned. The pipeline computes status from the error and feeds the trio to the recorder:

```go
rec := trace.New()

rec.Start("odds")
oddsData, oddsErr, oddsSnippet := p.deps.FetchOdds(ctx, ...)
rec.Finish("odds", oddsErr, oddsSnippet)
// ... same shape for news, lineup, context ...

rec.Start("predict")
res, err := p.claude.Predict(ctx, prompt)
rec.Finish("predict", err, predictSnippetFrom(res))
if err != nil { return ... }

traceBytes, _ := rec.JSON()
pred.TraceJSON = string(traceBytes)
```

Snippet derivation per fetcher (computed in the `main.go` closure):

| Kind    | Snippet content                                                     |
|---------|---------------------------------------------------------------------|
| odds    | `"bookmaker=<name> home=<price> away=<price> draw=<price>"`         |
| news    | `<home_summary>` first line + `" / "` + `<away_summary>` first line |
| lineup  | `"confirmed=<bool> notes=<notes-truncated>"`                        |
| context | `<tournament_context>` first 200 chars + `<track_record>` first 200 |
| predict | JSON-marshalled `{winner, predicted_score, win_probability}`        |

All snippets are then truncated to 400 chars (UTF-8 safe) inside `Recorder.Finish`. On error, the closure returns `snippet = ""` and lets the error string carry the story.

`store.Prediction` gets a `TraceJSON string` field. `store.InsertPrediction` writes it to the new column. The HTTP and subprocess implementations inside `internal/fetchers/*`, `internal/odds`, `internal/fdorg`, and `internal/claudec` are not touched by the trace recorder — they are touched separately by the stderr wirelog (next section). The recorder lives entirely in the `predict.Pipeline.Run` + `main.go` closure layer.

### Stderr logging

A small helper in `internal/trace` (or a sibling `internal/wirelog` package — choose whichever keeps the file short) provides:

```go
func HTTPStart(ns, method, url string)
func HTTPEnd(ns string, status int, duration time.Duration, bytes int)
func HTTPError(ns string, duration time.Duration, err error)

func SubprocessStart(ns string, promptBytes int)
func SubprocessEnd(ns string, duration time.Duration, outBytes int)
func SubprocessError(ns string, duration time.Duration, err error)
```

Output format mirrors the frontend's `[wcp:<ns>]` shape from `frontend/src/lib/trace.ts`:

```
[wcp:odds] → GET https://api.the-odds-api.com/v4/sports/...
[wcp:odds] ✓ 200 OK (412ms, 14KB)
[wcp:news] → claude -p (prompt: 287 chars)
[wcp:news] ✗ failed after 91240ms: context deadline exceeded
[wcp:trace] odds ✗ failed in 380ms — no odds found for Argentina vs Saudi Arabia
[wcp:trace] news ✗ failed in 91240ms — claude invoke: context deadline exceeded
```

Call sites:
- `internal/odds/client.go` — wrap `c.httpc.Do` to emit `HTTPStart` / `HTTPEnd` / `HTTPError`.
- `internal/fdorg/client.go` and siblings (`fixtures.go`, `results.go`, `teams.go`) — same wrapping. These calls happen in `bootstrap` and `results`, not in `predict`, but get the same stderr treatment.
- `internal/claudec/driver.go` and `internal/fetchers/shared.go` — wrap the `exec.CommandContext(...).Run()` calls with `SubprocessStart` / `SubprocessEnd` / `SubprocessError`.
- `internal/trace.Recorder.Finish` emits the per-fetcher `[wcp:trace]` summary line.

Namespaces in use after this change: `odds`, `fdorg`, `news`, `lineup`, `context`, `predict`, `trace`. The `news` / `lineup` / `predict` subprocess wrappers pick the namespace from a string the caller passes, so the fetchers don't need to know about wirelog directly — the call site does.

All output goes to `os.Stderr`. The existing launchd plist already captures stderr to `~/Library/Logs/wcp/<match-id>.log`, so no plist change is needed.

### JSON export

`store/export.go` extracts the prediction rows for `predictions.json` (read by the dashboard). After this change, each prediction object grows a sibling field:

```json
{
  "id": 17,
  "match_id": "2026-06-25-ARG-vs-SAU",
  "predicted_winner": "ARG",
  "win_probability": 0.65,
  "reasoning": "...",
  "trace": [
    { "kind": "odds", "started_at": "...", "duration_ms": 380, "ok": false, "error": "no odds found...", "snippet": "" },
    { "kind": "news", "started_at": "...", "duration_ms": 91240, "ok": false, "error": "...", "snippet": "" },
    { "kind": "lineup", "started_at": "...", "duration_ms": 612, "ok": false, "error": "malformed json", "snippet": "I don't have access to confirmed lineup..." },
    { "kind": "context", "started_at": "...", "duration_ms": 12, "ok": true, "error": "", "snippet": "Tournament: Group C standings — ..." },
    { "kind": "predict", "started_at": "...", "duration_ms": 3614, "ok": true, "error": "", "snippet": "{\"winner\":\"ARG\",\"predicted_score\":\"2-0\",..." }
  ]
}
```

When `trace_json` is NULL (legacy predictions), `trace` is emitted as `null` (not omitted — explicit null is easier to type-check in the frontend).

The JSON Schema in `schemas/prediction.json` gains the `trace` field as a nullable array of `TraceEntry` objects with the obvious shape. TypeScript types regenerate from it.

### Frontend UI

#### Trigger placement

Two triggers, both opening the **same** inline accordion drawer:

1. **Header pill** — next to the existing confidence badge in the prediction-card header row.
2. **(i) icon** — small circular button (20px) inline with the "Win probability" label inside `PredictionStats`.

Both share appearance rules driven by `okCount = trace.filter(t => t.ok).length`:

- `okCount === 5` → neutral grey: `background: rgba(0,0,0,0.04)`, text `var(--ink-3)`, border `rgba(0,0,0,0.12)`.
- `okCount < 5` → primary tone: `background: rgba(193,68,14,0.08)`, text `var(--primary)`, border `rgba(193,68,14,0.22)`.

Pill content: `<dot> N/5 INPUTS` (uppercase label). Icon content: `i`.

If `prediction.trace === null` (legacy row), neither trigger renders. If a future case ever produces `trace.length !== 5`, both still render with the label `INPUTS` and N reflecting the actual array — i.e., the UI is forgiving but the backend guarantees 5.

#### The drawer (inline accordion)

A new component, `<PredictionTrace>`, owned by the prediction body. State is local (`useState<boolean>(false)`). Both triggers call the same `setOpen(v => !v)`.

Placement: between the existing stats/reasoning two-column grid and the bottom action bar. Full width. Collapsed by default.

Open state:

```
┌─────────────────────────────────────────────────┐
│  INPUT TRACE · 2/5 OK            ▾ COLLAPSE     │
├─────────────────────────────────────────────────┤
│  ● ODDS                  ✗ failed · 380ms       │
│    no odds found for Argentina vs Saudi Arabia  │
├─────────────────────────────────────────────────┤
│  ● NEWS                  ✗ failed · 91.2s       │
│    claude invoke: context deadline exceeded     │
├─────────────────────────────────────────────────┤
│  ● LINEUP                ✗ failed · 612ms       │
│    malformed json                               │
│    ┌───────────────────────────────────────┐   │
│    │ I don't have access to confirmed line │   │
│    └───────────────────────────────────────┘   │
├─────────────────────────────────────────────────┤
│  ● CONTEXT               ✓ ok · 12ms            │
├─────────────────────────────────────────────────┤
│  ● PREDICT               ✓ ok · 3.6s            │
└─────────────────────────────────────────────────┘
```

Each row shows:
- Status dot (`--ok` green when `ok: true`, `--fail` red when `ok: false`).
- Kind name in uppercase tracking-label style.
- Status word + duration on the right. Durations under 1000ms render as `Nms`; ≥ 1000ms as `N.Xs` to one decimal.
- If `error` is non-empty: rendered in `--fail` color, indented under the row head.
- If `snippet` is non-empty: rendered in a mono-font block (`var(--surface-sunk)` background, 10.5px) below the error.

The collapse affordance is a text button on the right of the drawer header (`▾ COLLAPSE`), matching the tone of the existing `▴ Collapse` on the dashboard expand-card pattern.

#### Component layout

- New: `frontend/src/components/PredictionTrace.tsx` — pure, takes `trace: TraceEntry[]` plus `open` / `onToggle`.
- Modified: `frontend/src/components/PredictionStats.tsx` — accept an `onTraceClick` prop (optional); when set and `prediction.trace !== null`, render the `(i)` icon next to the "Win probability" label.
- Modified: `frontend/src/components/PredictionBody.tsx` — owns the `traceOpen` state. Renders the header pill (also conditional on `prediction.trace !== null`), passes `onTraceClick` into `<PredictionStats>`, renders `<PredictionTrace>` between the grid and the action bar.
- The new pill component lives inline in `PredictionBody` (small enough not to warrant its own file).

### Documentation updates

`CLAUDE.md` has an "Adding a new fetcher" checklist that today stops at wiring the fetcher into `predict.Pipeline.Deps`. Extend it so future fetchers stay consistent with tracing:

- Add a step: the fetcher's `main.go` closure must return `(data, err, snippet)` rather than `(data, ok)`, and the snippet should follow the per-fetcher format in this spec (short, human-readable, ≤400 chars — truncation is handled by `trace.Recorder`).
- Add a step: the HTTP client or subprocess call site inside the fetcher must use `internal/trace`'s wirelog helpers (`HTTPStart`/`HTTPEnd`/`HTTPError` for HTTP; `SubprocessStart`/`SubprocessEnd`/`SubprocessError` for `claude -p`) with a namespace that matches the new fetcher's `kind`.
- Add a step: extend `internal/trace.Recorder` to accept the new `kind` (if the array is no longer the fixed five). If the new fetcher is conditional, decide whether its absence reads as `ok: false` with a specific `error` or whether the kind is dropped from the array entirely — and update the frontend's `okCount` / "N/5" label accordingly.

## Test plan

### Backend

- `internal/trace`: table-driven test for `Recorder` covering normal start/finish, error cases, snippet truncation (≤400 chars on multi-byte input), and JSON output ordering.
- `internal/trace` wirelog helpers: golden-line tests against `bytes.Buffer` (replace `os.Stderr` via a package-private writer var) verifying format for HTTP and subprocess start/end/error.
- `internal/predict`: extend `pipeline_test.go` to assert that the `Recorder` was populated with five entries in the correct order, with `ok` reflecting whether each fake fetcher errored. Assert the persisted `Prediction.TraceJSON` round-trips through `store`.
- `internal/store`: add a migration test confirming that opening a pre-existing DB without `trace_json` succeeds and adds the column, and that newly inserted predictions persist + return the JSON intact.
- `internal/store/export`: assert that `predictions.json` includes the `trace` field (array when present, `null` for legacy rows).

### Frontend

- `frontend/tests/predictionTrace.test.tsx`: component renders status dot color, formatted duration, error line for failed rows, snippet block when present, and respects `open` state.
- `frontend/tests/predictionBody.test.tsx` (or extend existing): asserts both triggers (pill, icon) toggle the same `open` state, and that neither renders when `trace === null`.
- `frontend/tests/format.test.ts` or co-located: duration formatter (Nms / N.Xs) — table-driven.

### Manual

- Run `wcp predict --match <some-future-match>` with the odds API key missing — verify the stored prediction's `trace_json` shows odds failed with the expected error, and the dashboard renders the pill in the degraded color and the drawer with the expected row.
- Tail `~/Library/Logs/wcp/<match-id>.log` during a real launchd-fired prediction and confirm all expected `[wcp:<ns>]` lines appear, in the right order, with sensible durations.

## Open considerations

- **Long claude stdouts in `snippet`.** 400 chars is enough to identify malformed JSON or refusal text but will truncate any real prediction payload. Acceptable — debug context is the goal, not full reproducibility.
- **Bootstrap stderr volume.** Adding HTTP-level stderr to fdorg means `wcp bootstrap` produces ~one line per fetched match. Currently runs ~70+ matches → ~140 stderr lines. Acceptable; bootstrap is interactive.
- **Schema evolution.** If we later want fetcher-internal HTTP detail in the DB, the migration path is: add a sibling `trace_calls` table keyed on `prediction_id`, leave `trace_json` as the fetcher-level summary. Picked deliberately so the simple thing today doesn't block the richer thing later.
