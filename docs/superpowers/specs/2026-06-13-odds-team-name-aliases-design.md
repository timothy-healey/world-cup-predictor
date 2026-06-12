# Odds team-name aliases — Design

**Date:** 2026-06-13
**Status:** Draft (pending user review)

## Purpose

Every recent prediction's trace shows `"error":"no odds found for <home> vs <away>"` for fixtures involving United States, Bosnia-Herzegovina, Czechia, Cape Verde Islands, or Congo DR. The Odds API returns HTTP 200 with a valid event list; the failure is an exact, case-sensitive string comparison in `internal/odds/client.go::pickFirstH2H` against names that the two upstream providers (football-data.org and the-odds-api) write differently. The user's API key, budget, and HTTP path are all healthy — the bug is purely in our matching logic.

This spec adds a small translation step so DB-style names from football-data.org reach the Odds API comparison in the form that provider actually uses.

## Goals

- Translate the five known name disagreements before matching events to fixtures.
- Leave the other 43 World Cup teams unaffected — their names match verbatim.
- Ground the alias map in a captured live response (`backend/testdata/odds-wc2026.json`) so the strings are real, not guessed.
- Keep the public surface of `odds.Client` unchanged.
- Cover all five aliases plus the identity path with table-driven tests; cover the live wire path with one `httptest.NewServer`-based test.

## Non-goals

- No fuzzy / Levenshtein / substring matching. The set of WC teams is closed and known; fuzzy matching risks picking the wrong fixture (e.g. a Germany B-side, a women's tournament event in some other sport namespace) for zero current benefit.
- No both-orderings (home/away swap) fallback. The captured live response shows football-data.org and the-odds-api agree on home/away assignment for every observed failing pair. YAGNI.
- No retry, no warn-on-miss telemetry, no "missing alias" log line. The existing trace row already records `"no odds found for X vs Y"`, which is sufficient signal if a future tournament introduces a new name disagreement.
- No DB schema change. No new env var. No new dependency.
- No keyed-by-TLA refactor. A name-keyed map is simpler and the cost of regenerating it if we ever swap fixture providers is low.

## Evidence

Comparing the 48 DB team names against the 48 distinct names in `testdata/odds-wc2026.json` (captured 2026-06-13 from a live call against the user's key, costing 1 of 500 monthly credits, leaving 416 remaining), exactly five disagree:

| DB name (football-data.org)  | The Odds API name      |
|------------------------------|------------------------|
| Bosnia-Herzegovina           | Bosnia & Herzegovina   |
| Cape Verde Islands           | Cape Verde             |
| Congo DR                     | DR Congo               |
| Czechia                      | Czech Republic         |
| United States                | USA                    |

Every "no odds found" entry in the `predictions` table can be explained by this list (either the home or the away side appears in column 1).

## Architecture

### New file: `backend/internal/odds/alias.go`

```go
package odds

// aliases maps football-data.org canonical team names to the names used by
// the-odds-api. Only the five teams where the two providers disagree need
// entries; lookup falls through to identity for the other 43 WC teams.
//
// Regenerate by diffing the DB's teams.name column against the home_team /
// away_team strings in a live /v4/sports/soccer_fifa_world_cup/odds/ response.
var aliases = map[string]string{
    "Bosnia-Herzegovina": "Bosnia & Herzegovina",
    "Cape Verde Islands": "Cape Verde",
    "Congo DR":           "DR Congo",
    "Czechia":            "Czech Republic",
    "United States":      "USA",
}

func oddsAPIName(dbName string) string {
    if v, ok := aliases[dbName]; ok {
        return v
    }
    return dbName
}
```

### Modified: `backend/internal/odds/client.go::GetForMatch`

The lookup loop currently reads:

```go
for _, e := range events {
    if e.HomeTeam == homeName && e.AwayTeam == awayName {
        return pickFirstH2H(e)
    }
}
```

It becomes:

```go
wantHome := oddsAPIName(homeName)
wantAway := oddsAPIName(awayName)
for _, e := range events {
    if e.HomeTeam == wantHome && e.AwayTeam == wantAway {
        return pickFirstH2H(e)
    }
}
```

No signature change. The `homeName` / `awayName` parameters continue to accept DB-style names from the caller (`predict.Pipeline.Run` passes `home.Name` and `away.Name` from the `teams` table).

### New fixture: `backend/testdata/odds-wc2026.json`

The captured 939 KB live response covering 70 upcoming WC events. Used by the new wire-path test and available as a reference for any future fetcher that needs to know what the-odds-api actually returns.

## Test plan

Three new tests, all under `backend/internal/odds/`:

### `TestOddsAPIName` (table-driven, no I/O)

```go
cases := []struct{ in, want string }{
    {"Bosnia-Herzegovina", "Bosnia & Herzegovina"},
    {"Cape Verde Islands", "Cape Verde"},
    {"Congo DR",           "DR Congo"},
    {"Czechia",            "Czech Republic"},
    {"United States",      "USA"},
    {"Mexico",             "Mexico"},          // identity path
    {"",                   ""},                // empty stays empty
}
```

### `TestGetForMatch_AliasedTeams`

Stand up `httptest.NewServer` serving `testdata/odds-wc2026.json` verbatim, point a `*Client` at it, and assert that calling `GetForMatch` with DB-style names returns a non-zero `Odds.HomeOdds` for each of the five aliased teams. One pair per alias, drawn from the recorded fixture:

| DB-style call args (home, away)       | Live event matched         |
|---------------------------------------|----------------------------|
| `("United States", "Paraguay")`       | USA vs Paraguay            |
| `("Canada", "Bosnia-Herzegovina")`    | Canada vs Bosnia & Herzegovina |
| `("Spain", "Cape Verde Islands")`     | Spain vs Cape Verde        |
| `("Portugal", "Congo DR")`            | Portugal vs DR Congo       |
| `("Czechia", "South Africa")`         | Czech Republic vs South Africa |

Kickoff arg can be any RFC3339 string — the current code ignores it during matching, and we don't change that.

### `TestGetForMatch_NonAliasedTeams`

Same `httptest` setup, assert `GetForMatch(ctx, "Mexico", "South Africa", ...)` still returns non-zero `HomeOdds` — proves the identity path is intact.

All tests use the existing `httptest` pattern from `client_test.go`; no new test infrastructure.

## Order of work

1. Commit captured fixture `backend/testdata/odds-wc2026.json`.
2. Add failing `TestOddsAPIName` (function doesn't exist yet) — confirm `go test ./internal/odds/...` fails with a compile error.
3. Add `alias.go` with the map and `oddsAPIName`. `TestOddsAPIName` now passes.
4. Add failing `TestGetForMatch_AliasedTeams` (the existing matching logic doesn't translate, so the test returns "no odds found").
5. Patch the comparison in `client.go` to use `oddsAPIName`. Both new wire-path tests pass.
6. Run `go vet ./... && go test ./...` to confirm nothing else regressed.
7. Single commit: `fix(odds): translate team names to the-odds-api conventions`.

## Failure modes after the fix

- **A new team joins a future tournament with a name we haven't aliased**: `GetForMatch` returns `no odds found`. The trace records it. Confidence drops. Prediction continues. This is unchanged from today's behavior and graceful enough that no telemetry is added.
- **The Odds API renames an existing team**: same failure mode as above; the alias map needs a one-line update.
- **Fixture provider switch (replacing football-data.org)**: the map is keyed by football-data.org names, so the keys would need regenerating. We accept this re-work because a provider switch would touch many other places already.
