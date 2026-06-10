# World Cup Predictor

A local-first tool that predicts every match of the 2026 FIFA World Cup using Claude, betting odds, team news, lineup data, and the predictor's own track record. Predictions fire automatically 30 minutes before kick-off via macOS `launchd` and arrive as a formatted email.

The dashboard (separate plan) reads the same data to surface track record, upcoming matches, group standings, and a knockout bracket once the group stage concludes.

## Why local-first

- Tomorrow morning prediction emails reach Tim in Australia at ~5am local. The laptop is asleep, so launchd queues missed jobs and runs them on wake.
- The repo is shareable. A friend can clone it, drop their own keys in `.env`, and run their own predictor without sharing any state.
- Zero hosting cost. The Claude Max subscription drives the LLM via the `claude` CLI as a subprocess — no API key needed for the model.

## Setup

```bash
# 1. Clone and fill in credentials
git clone <repo>
cd world-cup-predictor
cp backend/.env.example backend/.env
# Edit backend/.env (see "Configuration" below)

# 2. Build + install the CLI
cd backend && make install
# wcp is now at ~/bin/wcp

# 3. Bootstrap the tournament from the backend directory
#    (the working directory at bootstrap time is captured into every launchd plist,
#    so the binary can find .env and wcp.db when launchd fires it later)
wcp bootstrap

# 4. Verify
wcp doctor
wcp doctor --dry-run-next   # full predict pipeline against the next match, no email
```

After step 3 there are 72 group-stage launchd agents scheduled at T-30 for each match. Re-run `wcp bootstrap` after the group stage ends to load the knockout fixtures once teams have advanced.

## Configuration

All credentials live in `backend/.env`, gitignored. `backend/.env.example` is the committed template.

| Variable | Required? | Notes |
|---|---|---|
| `FOOTBALL_DATA_API_KEY` | **yes** | [football-data.org](https://www.football-data.org/client/register) free tier. Used for fixtures, teams, finished results. |
| `THE_ODDS_API_KEY` | optional | [the-odds-api.com](https://the-odds-api.com/) free tier (500 reqs/month). Without it, predictions still run at lower confidence. |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASSWORD` / `NOTIFICATION_EMAIL_TO` | optional | Gmail SMTP. `SMTP_PASSWORD` must be a [Gmail app password](https://myaccount.google.com/apppasswords), not your account password. Without these, predictions are stored but no email is sent. |
| `WCP_DB_PATH` | optional | Defaults to `./wcp.db`. The launchd plist sets this to the absolute path captured at bootstrap time. |
| `WCP_SERVE_PORT` | optional | Defaults to `8765`. Used by the (forthcoming) dashboard's "predict now" button. |
| `WCP_CLAUDE_BIN` | optional | Defaults to `claude`. Path to the Claude Code CLI. |

The `claude` CLI must be on PATH and authenticated (`claude login`) under your own account before predictions will work — the repo doesn't manage Claude auth.

## Commands

```
wcp bootstrap                       # Fetch fixtures, write & load per-match launchd plists
wcp bootstrap --no-agents           # Fetch fixtures only; skip launchd setup (manual predictions only)
wcp predict --match <id>            # Predict a specific match. No email by default.
wcp predict                         # Predict the next upcoming unpredicted match
wcp predict --match <id> --email    # Force-send the prediction email (launchd uses this)
wcp results fetch                   # Pull recent finished match results
wcp doctor                          # Self-audit: config, claude binary, store, agents, rate limits
wcp doctor --dry-run-next           # Run a real prediction against the next match, no email
```

Use `--no-agents` if you don't want predictions to fire automatically — useful on non-macOS clones (no `launchd`), or if you'd rather drive every prediction yourself via `wcp predict` or the dashboard's "Predict now" button.

## Project structure

```
world-cup-predictor/
├── PRODUCT.md                      # Strategic brief (audience, anti-references, design principles)
├── DESIGN.md                       # Visual system tokens (Floodlight palette, type, spacing)
├── CLAUDE.md                       # Implementation guide for Claude / future contributors
├── README.md                       # This file
├── backend/
│   ├── cmd/wcp/                    # CLI entry: subcommand dispatch
│   ├── internal/
│   │   ├── bootstrap/              # Fetch fixtures + load launchd agents
│   │   ├── claudec/                # Subprocess wrapper around `claude -p`, embedded system prompt
│   │   ├── config/                 # Env-driven config with required/optional tiers
│   │   ├── doctor/                 # Self-audit command
│   │   ├── fdorg/                  # football-data.org HTTP client + rate-limit headers
│   │   ├── fetchers/               # odds wrapper + lineup/news (Claude web search) + context (SQLite)
│   │   ├── mailer/                 # Gmail SMTP + email template
│   │   ├── odds/                   # The Odds API client
│   │   ├── plist/                  # launchd .plist writer + launchctl load/unload
│   │   ├── predict/                # Pipeline orchestration + confidence flag + embedded prompt
│   │   ├── ratelimit/              # Shared in-process state for rate-limit observations
│   │   └── store/                  # SQLite store: teams, matches, predictions + predictions.json export
│   ├── prompts/                    # (Reserved — embedded under internal/predict for now)
│   ├── testdata/                   # Recorded API responses for fixture-based tests
│   └── Makefile                    # build, test, install
├── docs/
│   ├── design/
│   │   ├── README.md               # Visual reference index
│   │   ├── mockups/                # Self-contained HTML mockups (dashboard, bracket, card)
│   │   └── screenshots/            # PNG renders for at-a-glance review
│   └── superpowers/
│       ├── specs/                  # Approved design spec
│       └── plans/                  # Implementation plans
└── frontend/                       # (Forthcoming — separate plan)
```

## Development

```bash
cd backend
make build           # produces bin/wcp
make test            # go test ./...
make test-verbose    # go test -v ./...
make install         # copy bin/wcp to ~/bin/wcp
```

Tests use fixture-based recordings under `backend/testdata/` for HTTP clients and fake shell scripts in `t.TempDir()` for `claude` subprocess tests — no live network or real Claude calls.

## Status

- Backend: complete and verified end-to-end against the live 2026 World Cup opener.
- Frontend: design system locked in ([DESIGN.md](DESIGN.md), [docs/design/mockups](docs/design/mockups)). Implementation pending — separate plan to be written next.

## Documents

- [PRODUCT.md](PRODUCT.md) — audience, personality, anti-references, design principles
- [DESIGN.md](DESIGN.md) — design tokens (colour, type, spacing, components)
- [CLAUDE.md](CLAUDE.md) — implementation guide for Claude / future contributors
- [docs/design/README.md](docs/design/README.md) — visual mockups
- [docs/superpowers/specs/](docs/superpowers/specs/) — approved design specifications
- [docs/superpowers/plans/](docs/superpowers/plans/) — TDD implementation plans
