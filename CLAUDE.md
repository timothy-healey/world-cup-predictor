# Claude implementation guide

Companion to [PRODUCT.md](PRODUCT.md) (strategy) and [DESIGN.md](DESIGN.md) (visual system). This document captures the **implementation conventions** that aren't obvious from reading the code alone — what to do when adding features, where things live, what the non-obvious constraints are.

If you're an AI assistant returning to this repo in a new session, read this first.

## Mental model

The World Cup Predictor is a small Go CLI (`wcp`) plus a forthcoming React dashboard. The CLI:

1. Loads its config from `backend/.env` and the env.
2. Fetches fixtures + team data from football-data.org and persists to SQLite (`backend/wcp.db`).
3. Writes one macOS `launchd` LaunchAgent per match at `~/Library/LaunchAgents/com.wcp.<match-id>.plist`, scheduled to fire 30 minutes before kick-off.
4. When `launchd` fires a job (`wcp predict --match <id> --email`), the pipeline runs four fetchers in parallel (odds, news, lineup, context), assembles a prompt, calls `claude -p` as a subprocess, parses JSON output, persists the prediction, exports `predictions.json`, and emails the report.
5. The dashboard (separate plan) reads `predictions.json` and shows track record, upcoming matches, group standings / knockout bracket, and past results.

The Claude Max subscription drives the LLM via the `claude` CLI — there is no Anthropic API key in this codebase.

## Where things live

### `backend/cmd/wcp/main.go`
Subcommand dispatcher. Each subcommand (`bootstrap`, `predict`, `results`, `doctor`, `serve`) has a `run*` handler that constructs the dependencies it needs (`*store.Store`, `*fdorg.Client`, `*claudec.Driver`, etc.). The dispatcher loads `*config.Config` once at startup and threads it to every handler.

### `backend/internal/*` packages
Each package owns one bounded responsibility. The packages do not import each other except for these direct dependencies:

- `predict` → `claudec`, `fetchers`, `store`
- `fetchers` → `claudec` (via the `claudeBin` interface), `store`
- `bootstrap` → `fdorg`, `store`, `plist`
- `doctor` → `config`, `store`, `ratelimit`
- `mailer` → `store` (just for the type)
- everything else → no internal deps

The HTTP clients (`fdorg`, `odds`) and the subprocess driver (`claudec`) all record into `internal/ratelimit` so the doctor can read observed limits from a single source.

### `backend/internal/predict/predict.md`
The system prompt for predictions. **It is embedded into the binary via `//go:embed`** — do not move it or expect the path to work at runtime. The `loadSystemPrompt()` in `pipeline.go` reads the embedded bytes.

### `backend/testdata/`
Recorded JSON responses from real API calls, used for fixture-based unit tests. Tests stand up an `httptest.NewServer` that serves these bytes. Updating a fixture means re-recording (manually for now).

## Build and test

```bash
cd backend
make build         # builds bin/wcp
make test          # go test ./...
make install       # copies bin/wcp -> ~/bin/wcp
go vet ./...       # always passes; CI will block on this
```

Tests are fixture-driven. They never hit live APIs. The `claude` CLI is mocked by writing a small `#!/bin/sh` script to `t.TempDir()` and pointing `claudec.NewDriver` at it.

## Conventions

### TDD
Every task in the implementation plan followed the test-first cycle: write a failing test, run it, implement the minimum to pass, commit. Subsequent tasks build on the previous packages without modifying them. If you're adding a new feature, follow the same cycle — `internal/predict/confidence_test.go` is a good example of a tight table-driven test that doubles as documentation.

### Commits
Conventional commits — `feat(<package>): ...`, `fix(<package>): ...`, `docs: ...`. Each commit should leave the tree green. Co-authored-by trailer is fine but not required.

### Error handling
- HTTP clients return wrapped errors with the URL/status/body for debugging.
- Subprocess invocations of `claude` capture stderr and include it in the wrapped error.
- Optional features (odds, email) degrade silently with a one-time startup warning when their env vars are missing; predictions still run and store, with `confidence` adjusted accordingly.
- The `bootstrap` command continues on per-match failures (e.g. unresolvable TLAs from football-data.org's `/matches` endpoint) with a `[warn]` log line.

### Logging
`fmt.Fprintf(os.Stderr, "[warn] ...")` for warnings. There is no structured logger. Avoid `log` package (default format is verbose).

### File size discipline
Each `internal/*` package is small (one or two files, plus a `_test.go`). If a file you're modifying has grown past ~300 lines, that's a signal to consider a split — but follow plan guidance rather than splitting unilaterally.

## Architectural pinning

### The launchd cwd trap
This is a real production-blocker bug we already hit and fixed. When `launchd` fires `wcp predict ...`, the working directory is whatever `WorkingDirectory` the plist specifies, NOT where the user installed the binary. The bootstrap command captures `os.Getwd()` at bootstrap time and writes that into every plist as the absolute `WorkingDirectory`, plus injects `WCP_DB_PATH` as an `EnvironmentVariables` entry. **Always run `wcp bootstrap` from inside `backend/`** — `.env` and `wcp.db` live there.

### launchctl re-load
`launchctl load -w` fails if a label is already loaded. The `bootstrap` command calls `plist.UnloadAgent(path)` (ignoring its error) before `plist.LoadAgent(path)`, so re-running bootstrap correctly picks up any plist content changes. Never call `LoadAgent` without unloading first.

### Football-data.org TLA inconsistency
The `/teams` endpoint and the `/matches` endpoint occasionally return different TLAs for the same team (we hit Curaçao as `CUW` vs `CUR`). Bootstrap resolves the canonical team code by:
1. Looking up by `match.HomeTLA` in `teams.code`
2. If miss, looking up by `match.HomeID` in `teams.fixture_src_id`
3. If still miss, logging a warning and skipping the match

Don't rely on TLA equality. Use `bootstrap.resolveTeamCode()` if you need to bridge from match data to team data.

### Knockout fixtures show up as TBD
Until the group stage concludes, knockout-round matches in football-data.org's `/matches` feed often have empty TLAs and empty IDs. They're skipped by bootstrap and need a re-run after group stage to be loaded. Don't try to "fix" this; it's correct degraded behaviour.

### Claude timeout
`claudec.Driver` has a 5-minute default timeout per invocation. Web-search prompts can take 60–180s comfortably. Don't reduce this; the four fetchers + main prediction call already total ~3–5 minutes in the happy path.

### Rate-limit state is in-process
`internal/ratelimit` is process-global. The doctor command's "Rate limits (last observed)" section only shows observations from the current CLI invocation. This is by design — warnings fire at the moment of the breach (during bootstrap, predict, results), which is when they matter. Persisting between invocations would be a small addition (JSON file under the work dir) if it becomes useful.

### `predictions.variant` is a schema down payment
The `predictions` table has a `variant TEXT NOT NULL DEFAULT 'full'` column added ahead of a planned post-hoc ablation experiment harness. The production pipeline only ever writes `"full"`. Reserved values are `"no-odds"`, `"no-news"`, `"no-lineup"`, `"no-context"` — each replays a finished match against the stored `inputs_json` with that block masked. Until the experiment subcommand lands, **filter to `variant = 'full'` in any new accuracy / track-record query** so non-production rows don't pollute stats. `store.Open` runs an idempotent `ALTER TABLE` to add the column to pre-existing DBs.

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

## Adding a new CLI subcommand

1. Add an entry to the `commands` slice in `cmd/wcp/main.go` with a `run<Name>` handler.
2. The handler signature is `func(ctx context.Context, cfg *config.Config, args []string) error`.
3. If the subcommand reads from the store, open it once at the top: `s, err := store.Open(cfg.DBPath); defer s.Close()`.
4. If the subcommand modifies external state (filesystem, network), prefer printing to stdout what was done; reserve stderr for `[warn]`/`[error]` lines.

## Common gotchas

- **`go vet` is strict on shadowed `err` in test loops.** Use a different var name if you're scanning rows in a loop.
- **`sql.NullString` everywhere.** Most text columns in SQLite are nullable; scanning into a bare `string` will panic on NULL. Look at how `store/teams.go::GetTeam` handles this and follow the pattern.
- **`json:"-"` on `fdorg.Match` fields.** The fdorg domain type uses `-` tags on raw fields populated from the nested wire types — these are deliberate, not bugs. Don't add encoding tags here without checking the consumer.
- **macOS-only.** `internal/plist/launchctl.go` no-ops on non-Darwin so tests run on Linux CI, but the real `wcp bootstrap` invocation requires macOS.

## Frontend (forthcoming)

The dashboard will be a Vite + React + TypeScript app that reads `predictions.json` and posts to a small local HTTP server (`wcp serve`, currently stubbed) for on-demand prediction triggers. The visual system is locked in [DESIGN.md](DESIGN.md). The implementation plan does not exist yet — it'll be a sibling to `docs/superpowers/plans/2026-06-10-backend.md` once written.

When the frontend lands:
- `Prediction`, `Match`, `Team` types in TypeScript will be generated from JSON Schema (source of truth: `schemas/prediction.json`) so backend struct changes propagate at build time.
- The frontend hits `127.0.0.1:8765` (configurable via `WCP_SERVE_PORT`) for the predict-now button. The `serve` subcommand will need to be implemented before this works.
