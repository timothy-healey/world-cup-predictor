# World Cup Predictor — Design

**Date:** 2026-06-10
**Status:** Approved (pending user review of written spec)

## Purpose

A personal tool that predicts the outcome of each 2026 FIFA World Cup match. For each fixture, it aggregates betting odds, recent news, squad/lineup information, and the predictor's own running track record, then asks Claude to produce a predicted winner, predicted scoreline, and reasoning. A formatted prediction report is emailed 30 minutes before kick-off, and all predictions are browsable on a local web dashboard.

## Goals

- One prediction per match, ideally 30 minutes before kick-off.
- Predictions remain useful even when the user's laptop was asleep during the scheduled time (the user is in a timezone where many matches kick off at 4:30am local).
- Track predictions vs. actual results so the user can see whether the tool is any good.
- Zero monthly hosting cost. Runs locally on the user's Mac.

## Non-goals

- No model fine-tuning or weight-level training. The predictor *does* improve over time by feeding past results and its own track record into each prompt as context (in-context learning), but Claude's weights are never modified.
- No multi-user support, authentication, or hosting beyond the user's machine.
- No live in-match updates — one prediction per match, locked in at kick-off.
- No betting integrations (read-only odds, never place a bet).

## Stack

| Layer | Tool |
| --- | --- |
| Backend CLI (`wcp`) | Go — single static binary invoked by `launchd` and manually |
| Backend HTTP API (local-only) | `net/http` on `127.0.0.1` — used by the dashboard's "predict now" button |
| Storage | SQLite via `modernc.org/sqlite` (pure Go, no CGO) |
| Email | `net/smtp` to Gmail |
| Tests | stdlib `testing` + `testify` |
| Frontend | Vite + React + TypeScript |
| Type sharing | JSON Schema (`schemas/prediction.json`) → generated Go structs + TS types |
| Scheduler | macOS `launchd` LaunchAgents, one `.plist` per match |
| LLM | Headless Claude Code (`claude -p`) using the user's Claude Max subscription |

## Data sources

| Input | Source | Why |
| --- | --- | --- |
| Fixtures + kick-off times | [football-data.org](https://football-data.org) free tier | Reliable World Cup schedule |
| Pre-tournament squads | football-data.org | Submitted FIFA rosters |
| Confirmed starting XI | Claude web search (~1 hr before kick-off) | Lineups typically announced ~60 min before; LLM finds official posts |
| Betting odds | [The Odds API](https://the-odds-api.com/) free tier (500 reqs/month) | Structured decimal odds across bookmakers; 104 matches × ~3 polls ≈ 312 calls, fits free tier |
| News | Claude web search | Fresher than news API free tiers; LLM filters relevance natively |
| Tournament context + track record | Local SQLite store | Derived from already-stored matches and predictions |

## Architecture

### Core operation

`predict(match_id)` — fetch available data, run Claude, save the result. One operation, invoked from multiple triggers. The operation does not care why it was called.

### Triggers

| Trigger | When | How |
| --- | --- | --- |
| **Per-match scheduled** | Exactly at T-30 for each match | One `launchd` LaunchAgent per match, configured via `StartCalendarInterval` for that match's T-30 time. If the laptop is awake → fires on time. If asleep → `launchd` runs it on wake (built-in catch-up). |
| **On-demand CLI / dashboard** | `wcp predict next` or `wcp predict --match <id>`, or a button on the dashboard that POSTs to the local HTTP server | Manual invocation, no email send. |

A `bootstrap` command runs once at tournament start: fetches the team list, fetches all 104 fixtures, writes a `.plist` per match into `~/Library/LaunchAgents/com.wcp.<match-id>.plist`, loads them with `launchctl`. Idempotent: re-running updates moved fixtures and refreshes team metadata.

### Confidence levels

Every prediction is tagged with a confidence flag based on what data was available at runtime:

- **`high`** — confirmed starting XI found.
- **`medium`** — pre-tournament 26-player squad available, no confirmed XI yet (the asleep-laptop case).
- **`low`** — squad data missing entirely, fell back to news + odds only, or 2+ fetchers failed.

The dashboard surfaces this prominently so the user knows how much to trust each prediction.

### Components

```
┌─────────────────┐
│  bootstrap CLI  │  one-shot: fetch fixtures → write 104 launchd .plists
└────────┬────────┘
         │
         ▼
┌─────────────────┐    triggers      ┌──────────────────┐
│  launchd agents │ ───────────────► │  predict CLI     │
│  (one per match)│   at T-30 each   │  (core operation)│◄──── on-demand
└─────────────────┘                  └────────┬─────────┘
                                              │
                ┌─────────────────────────────┼─────────────────┐
                ▼                ▼            ▼                 ▼
         ┌────────────┐   ┌────────────┐   ┌────────────┐  ┌────────────┐
         │  fetchers  │   │  claude    │   │  context   │  │   store    │
         │ (odds/news │   │  driver    │   │  (reads    │  │ (SQLite +  │
         │  /lineup)  │   │ (headless) │   │   store)   │  │ JSON dump) │
         └────────────┘   └────────────┘   └────────────┘  └─────┬──────┘
                                                                 │
                                              ┌──────────────────┼───────────┐
                                              ▼                              ▼
                                       ┌────────────┐                 ┌────────────┐
                                       │   mailer   │                 │ dashboard  │
                                       │   (SMTP)   │                 │  (Vite +   │
                                       │            │                 │   React)   │
                                       └────────────┘                 └────────────┘
```

Six small units, each with one job:

1. **`bootstrap`** — fetches teams, then fixtures; generates and loads `launchd` `.plist` files. Idempotent.
2. **`fetchers/`** — four independent modules:
   - `odds.go` — calls The Odds API for a match; returns normalized decimal odds + implied probabilities.
   - `news.go` — invokes headless Claude with web search; returns a summary of relevant news for both teams over the last 14 days.
   - `lineup.go` — invokes headless Claude with web search; finds confirmed XI; falls back to "squad-only" mode if not found and sets confidence accordingly.
   - `context.go` — reads the local SQLite store; returns tournament context (results so far, standings if applicable) plus the predictor's own track record vs. completed predictions.
3. **`claude_driver`** — thin wrapper around `claude -p` headless invocation. Takes a structured prompt, returns parsed JSON prediction. Handles retries and rate limiting.
4. **`store`** — SQLite database plus an auto-exported `predictions.json` consumed by the dashboard.
5. **`mailer`** — sends formatted prediction email via Gmail SMTP.
6. **`dashboard`** — static React app served by Vite; reads `predictions.json`; can POST to a local HTTP server for on-demand predictions.

Boundaries: fetchers know nothing about Claude. The Claude driver knows nothing about email or storage. Mailer and dashboard both depend only on `store` outputs.

### Repo layout

```
world-cup-predictor/
├── backend/
│   ├── cmd/wcp/                  # CLI entry: predict, bootstrap, doctor, serve
│   ├── internal/
│   │   ├── fetchers/             # odds, news, lineup, context
│   │   ├── claude/               # subprocess wrapper around `claude -p`
│   │   ├── store/                # SQLite + predictions.json export
│   │   ├── mailer/
│   │   ├── bootstrap/            # fixture fetch + plist generation
│   │   └── server/               # local HTTP for "predict now"
│   ├── testdata/                 # API response fixtures
│   └── go.mod
├── frontend/
│   ├── src/                      # React components
│   ├── public/predictions.json   # backend writes here
│   ├── package.json
│   └── vite.config.ts
├── schemas/
│   └── prediction.json           # JSON Schema → Go struct + TS type
├── prompts/
│   └── predict.md                # system prompt, version-controlled
├── docs/superpowers/specs/
├── .env.example                  # documented required env vars; committed
├── .env                          # actual secrets; gitignored
├── .gitignore
├── README.md
└── Makefile                      # build, test, install (load launchd agents)
```

### Configuration and secrets

All API keys and per-user settings live in a single gitignored `.env` file at the repo root. A committed `.env.example` documents every required variable. The Go binary loads `.env` at startup via `godotenv` (or equivalent) — anyone cloning the repo copies `.env.example` to `.env`, fills in their own credentials, and the tool runs against their accounts.

Environment variables fall into three tiers:

**Required — tool refuses to start without these:**

```
FOOTBALL_DATA_API_KEY=...        # No fixtures or teams without this; bootstrap fails
```

**Optional — feature is silently disabled with a startup warning if absent:**

```
THE_ODDS_API_KEY=...             # Missing → odds fetcher skipped on every run, confidence drops one level

# Email (Gmail SMTP) — missing any of these disables email entirely
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=you@example.com
SMTP_PASSWORD=...                # Gmail app password (16-char), not your account password
NOTIFICATION_EMAIL_TO=you@example.com
```

**Defaults — overrides only:**

```
WCP_DB_PATH=./wcp.db
WCP_SERVE_PORT=8765
WCP_CLAUDE_BIN=claude            # path to the claude CLI, default uses $PATH
```

**Graceful degradation rule.** The tool checks env vars once at startup and logs one warning per missing optional feature, e.g.:

```
[warn] THE_ODDS_API_KEY not set — odds fetcher will be skipped
[warn] SMTP_USER not set — email notifications disabled
```

At runtime, the affected fetcher or mailer becomes a no-op that returns `ok=false, error="not configured"`. This flows through the existing error-handling path: prediction still runs, confidence flag drops accordingly, dashboard still shows the prediction. No silent failures.

**`wcp doctor`** surfaces config state explicitly — lists which optional features are enabled, which are degraded, and which required variables are missing. Useful first-run command after `cp .env.example .env`.

**Claude Code authentication is not managed by this repo.** The tool invokes the `claude` CLI as a subprocess, which uses whatever auth state is in `~/.claude/`. A new user who clones the repo must have Claude Code installed and have completed `claude` interactive login (or `claude login`) under their own account before `wcp predict` will work. `wcp doctor` checks for the `claude` binary on PATH and reports if it's missing.

**The Claude subscription itself is per-user.** The cost summary below assumes the user is on Claude Max; a new clone might run against an API key instead via a future config flag (see Open considerations).

## Data model

Three tables: `teams`, `matches`, `predictions`. Teams are normalized so matches and predictions reference them by FIFA code, and team metadata (group, ranking, flag) has one home. Match results live as nullable columns on `matches` rather than a separate table — for 104 matches and one result each, a join is unjustified.

Small-cardinality string columns whose values are defined in code (`confidence`, `trigger`) use `CHECK` constraints. Open-cardinality references (teams) use foreign keys to a real table.

```sql
CREATE TABLE teams (
  code                  TEXT PRIMARY KEY,    -- FIFA 3-letter code: "ARG", "USA", "MEX"
  name                  TEXT NOT NULL,       -- "Argentina"
  group_id              TEXT,                -- "A".."L" during group stage, NULL after
  flag_url              TEXT,                -- for dashboard
  fifa_ranking          INTEGER,             -- snapshot at tournament start
  manager_name          TEXT,                -- head coach at tournament start; insurance against last-minute changes Claude wouldn't know
  pre_tournament_form   TEXT,                -- JSON: last 5 matches before tournament. [{date, opponent_code, venue, score_for, score_against, competition}]
  fixture_src_id        TEXT                 -- football-data.org's team ID
);

CREATE TABLE matches (
  id                 TEXT PRIMARY KEY,                       -- e.g. "2026-06-11-MEX-vs-CAN"
  home_team_code     TEXT NOT NULL REFERENCES teams(code),
  away_team_code     TEXT NOT NULL REFERENCES teams(code),
  kickoff_utc        TEXT NOT NULL,                          -- ISO 8601
  stage              TEXT NOT NULL,                          -- "group", "round-of-32", "round-of-16", "qf", "sf", "final", "third-place"
  venue              TEXT,
  fixture_src_id     TEXT,                                   -- football-data.org's ID
  -- result columns, NULL until match completes
  home_score         INTEGER,
  away_score         INTEGER,
  result_fetched_at  TEXT
);

CREATE TABLE predictions (
  id                 INTEGER PRIMARY KEY AUTOINCREMENT,
  match_id           TEXT NOT NULL REFERENCES matches(id),
  created_at         TEXT NOT NULL,
  trigger            TEXT NOT NULL CHECK (trigger IN ('scheduled', 'on_demand')),
  confidence         TEXT NOT NULL CHECK (confidence IN ('high', 'medium', 'low')),
  predicted_winner   TEXT NOT NULL,                          -- a teams.code value, or the literal "draw"
  predicted_score    TEXT NOT NULL,                          -- e.g. "2-1"
  win_probability    REAL,                                   -- 0.0-1.0
  reasoning          TEXT NOT NULL,                          -- markdown
  inputs_json        TEXT NOT NULL,                          -- raw fetcher outputs, for audit
  rendered_prompt    TEXT NOT NULL,                          -- full prompt sent to Claude
  model_id           TEXT NOT NULL,                          -- e.g. "claude-opus-4-7"
  prompt_version     TEXT NOT NULL                           -- git SHA of prompts/predict.md at run time
);
```

Notes:

- **`predicted_winner` accepts a team code OR the literal string `"draw"`.** `"draw"` is an out-of-band sentinel — no FIFA team code is `draw`. Application code validates this on insert.
- **`teams.group_id` is nullable** so the same row can describe a team in both the group stage (group set) and knockout stage (group cleared to NULL after group concludes). Tournament context queries that need "what group was Argentina in?" can recover it from the match-stage history if ever needed.
- **Bootstrap populates `teams`** before `matches`. football-data.org returns teams as part of the competition feed; the bootstrap fetches teams once, then fixtures. For each team, bootstrap also fetches the last 5 completed matches (any competition, prior to tournament start) to populate `pre_tournament_form`.
- **`pre_tournament_form` is naturally superseded** by in-tournament form once matches accumulate. The context fetcher includes both, but the prompt template should weight pre-tournament form more heavily in matchdays 1–2 and less thereafter.
- **`predictions.predicted_winner`** is not a hard FK because of the `"draw"` sentinel, but the application validates that non-draw values exist in `teams`.

Multiple predictions per match are allowed: if a `medium`-confidence prediction was made overnight and the user re-runs after waking, a second `high`-confidence row gets written. The dashboard shows the latest, but history is preserved.

Match results arrive via a separate `fetch_results` job that runs daily (a single `launchd` agent) and upserts final scores into the `matches` table from football-data.org's `/matches?status=FINISHED&competition=WC` endpoint.

## Prediction flow (single match)

```
1. Load match from store. Fail loudly if not in matches table
   (means bootstrap missed it — surface via `wcp doctor`).

2. Run fetchers in parallel:
   ├─ odds.go     ─► The Odds API call (~1s)
   ├─ news.go     ─► headless Claude w/ web search (~30-60s)
   ├─ lineup.go   ─► headless Claude w/ web search (~30-60s)
   └─ context.go  ─► SQLite query (instant)

   Each fetcher returns { ok bool, data ..., error string }.
   Any failure is captured; prediction still proceeds.

3. Determine confidence:
   - Start at "high" if confirmed XI was found.
   - Start at "medium" if only the 26-player squad was available.
   - Start at "low" if the lineup fetcher crashed entirely.
   - For each additional fetcher (odds, news, context) that failed,
     drop one level. Minimum is "low".

4. Build prompt from fetcher outputs + system prompt (prompts/predict.md).

5. Invoke claude_driver:
   - calls `claude -p` with the prompt
   - expects structured JSON output (winner, score, win_probability, reasoning)
   - on parse failure: one retry with a corrective follow-up
   - on rate limit: exponential backoff up to 3 attempts

6. Persist prediction row to SQLite. Includes inputs_json + rendered_prompt
   for full reproducibility.

7. Re-export predictions.json for the dashboard.

8. If triggered at scheduled T-30 (not on-demand): send email via SMTP.
   On-demand runs don't email.

9. Exit. launchd unloads the agent.
```

Fetchers run in parallel to keep the pipeline under ~90s end-to-end. The two Claude-driven fetchers are the long pole; running them serial would push the run to ~2 minutes.

Structured Claude output uses strict JSON keys (`winner`, `predicted_score`, `win_probability`, `reasoning`). Parsing failures get one corrective retry before being treated as errors.

## Context fetcher

Runs for every prediction; gracefully returns sparse content early in the tournament and richer content as data accumulates. Two blocks:

**Tournament context** — derived from `matches` with results filled in:

- Group standings (computed live for matches in group stage).
- Recent results for both teams in this tournament (form: W/D/L sequence).
- Notable highlights from prior matches involving either team (top scorers, blowouts, missed key players).

**Predictor track record** — derived from `predictions` joined against `matches` results:

- Total predictions completed (i.e. results in).
- Winner accuracy (e.g. "32/48, 67%").
- Exact score accuracy.
- Calibration notes ("model has predicted draw 5 times, actual draws: 1 — tends to over-weight defensive matchups").
- Sample-size disclaimer included so Claude doesn't over-weight noisy early signals.

No new tables. All values are derivable from the existing data.

## Error handling

Pipeline never silently fails. Each failure mode and its behaviour:

| Failure | Behaviour |
| --- | --- |
| Odds API down / over quota | Proceed without odds. Confidence drops one level. Reasoning prompt notes "no betting odds available". Email subject prefixed `[partial]`. |
| News fetch fails | Same — proceed without; drop confidence; note in reasoning. |
| Lineup fetch finds no confirmed XI | Expected for early-morning matches. Fall back to 26-player squad + recent first-XI history. Confidence = `medium`. Not treated as an error. |
| Lineup fetch crashes entirely | Treated as error. Confidence = `low`. |
| Claude returns malformed JSON | One retry with corrective prompt. If still bad, no prediction row is written. Log the raw response. Email subject `[failed]` with the parse error. |
| Claude rate-limited | Exponential backoff up to 3 attempts. If still failing, no prediction row is written. Email instructs manual retry via `wcp predict --match <id>`. |
| SMTP send fails | Prediction is already stored. Log SMTP error. Dashboard still shows it. Email failures don't roll back predictions. |
| SMTP env vars missing | One-time startup warning. Mailer is a no-op for the session. Predictions still write to DB and surface on dashboard. |
| `THE_ODDS_API_KEY` missing | One-time startup warning. Odds fetcher returns `ok=false, error="not configured"` on every run. Confidence drops one level (same path as API failure). |
| `FOOTBALL_DATA_API_KEY` missing | Tool refuses to start. Error message points to `.env.example`. |
| `launchd` agent missing for a match | `wcp doctor` audits this: lists matches without a loaded agent. Bootstrap re-run fixes it. |
| Fixture moved | `wcp bootstrap --refresh` re-pulls fixtures and rewrites any `.plist` whose kickoff changed. Idempotent. |

Two non-obvious choices:

- **Partial predictions are still predictions.** Missing one input doesn't kill the run — the prediction gets a lower confidence flag and the reasoning explicitly says what was missing.
- **`wcp doctor` subcommand** catches silent-failure modes where bootstrap missed a match or a `.plist` got deleted. Reports: matches scheduled, agents loaded, matches in next 7 days missing agents.

## CLI surface

```
wcp bootstrap                       # Fetch fixtures, write & load launchd plists
wcp bootstrap --refresh             # Re-pull fixtures, update changed plists

wcp predict --match <id>            # Predict a specific match. No email by default.
wcp predict next                    # Predict the next upcoming unpredicted match
wcp predict --match <id> --email    # Force-send the prediction email
wcp predict --match <id> --dry-run  # Print prompt; no Claude call; no DB write

wcp results fetch                   # Pull recent finished match results

wcp serve [--port 8765]             # Local HTTP server for the dashboard

wcp doctor                          # Self-audit: matches scheduled, agents loaded
wcp doctor --dry-run-next           # Smoke test: real-API prediction on next match
```

Email semantics: `--email` flag controls whether the prediction email is sent. The `launchd` `.plist` for each match invokes `wcp predict --match <id> --email`, so scheduled runs always email. Manual CLI runs never email unless `--email` is passed.

## Testing

Test layers and what each verifies:

| Layer | What's tested | How |
| --- | --- | --- |
| Fetcher units | Each fetcher parses a recorded response correctly | Fixture-based: commit real API responses to `testdata/`; parse offline; assert normalized output |
| Confidence logic | All combinations of fetcher success/failure → correct flag | Table-driven Go test |
| Prompt assembly | Given fetcher outputs, prompt string matches expected | Snapshot test |
| Claude driver parsing | Malformed JSON, partial JSON, retries, exhausted retries | Mock the subprocess layer; feed canned responses |
| Store | Predictions written/read correctly; JSON export shape matches dashboard expectations | Real SQLite in a `t.TempDir()` per test |
| plist generation | Bootstrap emits valid `launchd` XML for a given match | Snapshot test |
| Bootstrap idempotency | Re-running with same fixtures → no duplicate agents; updated kickoff times overwrite | Integration test against a temp `LaunchAgents` dir (not the real one) |
| End-to-end (stubbed) | Full pipeline with all external calls mocked | Golden-path; degraded-data; all-fetchers-fail |

Deliberately not unit-tested: real Claude calls (slow, eats subscription quota), real API calls (rate limits, network), actual `launchctl load`, email delivery.

Development helpers:

- `wcp predict --match <id> --dry-run` — full pipeline; prints assembled prompt; no Claude call; no DB write. For prompt iteration.
- `wcp predict --match <id> --no-email` — real run, real Claude, no email. For mid-day iteration without spamming yourself.
- `wcp doctor --dry-run-next` — smoke test against real APIs for the next upcoming match.

## Type sharing between Go and React

The `Prediction` record is read by both the Go backend (when it writes the row) and the React frontend (when it renders the card). Source of truth: `schemas/prediction.json`.

- Go side: `go generate` invokes a code generator (`schema-generate` or similar) to produce `internal/store/prediction_gen.go` with the matching struct.
- TS side: a Vite plugin (or a pre-build script using `json-schema-to-typescript`) generates `frontend/src/types/prediction.ts`.

Drift between the two is caught at build time.

## Cost summary

| Item | Cost |
| --- | --- |
| The Odds API | $0 (free tier) |
| football-data.org | $0 (free tier) |
| Claude LLM | $0 (Claude Max subscription) |
| Email (Gmail SMTP) | $0 (within personal Gmail limits — 500/day) |
| Hosting | $0 (local Mac) |
| **Total** | **$0 / tournament** |

## Open considerations (not blockers)

- **Email provider lock-in.** Starting with Gmail SMTP for simplicity; if the user later wants Resend or AWS SES, the `mailer` package is small enough to swap.
- **Dashboard hosting.** The static frontend can be served by `wcp serve` locally during development. If the user wants it accessible from a phone, deploy `frontend/dist` to GitHub Pages or Vercel post-tournament with no code changes — the JSON it reads can be committed.
- **Knockout bracket visualization.** A nice-to-have for v2; the dashboard initially shows a flat chronological list of matches.
