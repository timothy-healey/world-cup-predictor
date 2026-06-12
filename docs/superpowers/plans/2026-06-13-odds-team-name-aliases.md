# Odds team-name aliases — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Translate five DB-style team names through an alias map before matching events from the-odds-api, so predictions for fixtures involving United States, Bosnia-Herzegovina, Czechia, Cape Verde Islands, and Congo DR stop dropping odds.

**Architecture:** New file `backend/internal/odds/alias.go` exports a package-internal `oddsAPIName(dbName string) string` backed by a 5-entry `map[string]string`. The existing comparison in `client.go::GetForMatch` is updated to call `oddsAPIName` on the home/away args before the equality check. No public signature change, no plumbing changes upstream.

**Tech Stack:** Go 1.21+ standard library, `github.com/stretchr/testify/require`, `net/http/httptest`. All test infrastructure already exists in `backend/internal/odds/client_test.go` — reuse the same patterns.

**Spec:** `docs/superpowers/specs/2026-06-13-odds-team-name-aliases-design.md`.

**Pre-conditions:** The captured live response is already on disk at `backend/testdata/odds-wc2026.json` (untracked). The working directory at task start should be the repo root, with that file present.

---

## File structure

- **Create:** `backend/internal/odds/alias.go` — 5-entry alias map + `oddsAPIName` lookup function.
- **Create:** `backend/internal/odds/alias_test.go` — table-driven test for `oddsAPIName`.
- **Modify:** `backend/internal/odds/client.go:137-142` — call `oddsAPIName` on the two name args before the event-list comparison.
- **Modify:** `backend/internal/odds/client_test.go` — append two new wire-path tests (`TestGetForMatch_AliasedTeams`, `TestGetForMatch_NonAliasedTeams`) that use the captured fixture.
- **Add (already on disk):** `backend/testdata/odds-wc2026.json` — captured live response, 939 KB, 70 events.

---

## Task 1: Commit the captured fixture

**Files:**
- Add: `backend/testdata/odds-wc2026.json` (already present, untracked)

- [ ] **Step 1: Verify the file is present and unmodified**

Run:
```bash
ls -l backend/testdata/odds-wc2026.json
```
Expected: a single file ~939 KB. If absent, the brainstorming step that captured it was skipped — stop and recover before continuing.

- [ ] **Step 2: Stage and commit only the fixture**

```bash
git add backend/testdata/odds-wc2026.json
git commit -m "$(cat <<'EOF'
test(odds): record live /v4/sports/soccer_fifa_world_cup/odds/ response

Captured 2026-06-13 with regions=uk,us,eu,au, markets=h2h. 70 events
covering all 48 WC teams. Used as a fixture for upcoming alias-aware
matching tests in internal/odds.
EOF
)"
```
Expected: one new file committed; `git status` shows no other staged changes.

---

## Task 2: Add `oddsAPIName` with TDD

**Files:**
- Create: `backend/internal/odds/alias_test.go`
- Create: `backend/internal/odds/alias.go`

- [ ] **Step 1: Write the failing unit test**

Create `backend/internal/odds/alias_test.go`:

```go
package odds

import "testing"

func TestOddsAPIName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Bosnia-Herzegovina", "Bosnia & Herzegovina"},
		{"Cape Verde Islands", "Cape Verde"},
		{"Congo DR", "DR Congo"},
		{"Czechia", "Czech Republic"},
		{"United States", "USA"},
		{"Mexico", "Mexico"}, // identity path
		{"", ""},             // empty stays empty
	}
	for _, c := range cases {
		if got := oddsAPIName(c.in); got != c.want {
			t.Errorf("oddsAPIName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it fails (compile error)**

Run:
```bash
cd backend && go test ./internal/odds/ -run TestOddsAPIName -v
```
Expected: build failure containing `undefined: oddsAPIName`.

- [ ] **Step 3: Create `alias.go` with the map and lookup**

Create `backend/internal/odds/alias.go`:

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

- [ ] **Step 4: Run the test to verify it passes**

Run:
```bash
cd backend && go test ./internal/odds/ -run TestOddsAPIName -v
```
Expected: `--- PASS: TestOddsAPIName` and `ok  github.com/timhealey/world-cup-predictor/backend/internal/odds`.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/odds/alias.go backend/internal/odds/alias_test.go
git commit -m "$(cat <<'EOF'
feat(odds): add team-name alias map for the-odds-api

Five teams use names from football-data.org that don't match what
the-odds-api returns. oddsAPIName translates DB names to Odds API names
with identity fallback for the other 43 WC teams.
EOF
)"
```

---

## Task 3: Wire the alias into `GetForMatch`

**Files:**
- Modify: `backend/internal/odds/client_test.go` (append two tests)
- Modify: `backend/internal/odds/client.go:137-142`

- [ ] **Step 1: Append two failing wire-path tests**

Append to `backend/internal/odds/client_test.go`:

```go
// TestGetForMatch_AliasedTeams verifies that DB-style names for the five
// teams whose names differ between football-data.org and the-odds-api are
// translated through oddsAPIName before the event lookup. Uses the captured
// live response so the strings on the wire are real, not synthesized.
func TestGetForMatch_AliasedTeams(t *testing.T) {
	body, err := os.ReadFile("../../testdata/odds-wc2026.json")
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	cases := []struct {
		home, away string
	}{
		{"United States", "Paraguay"},
		{"Canada", "Bosnia-Herzegovina"},
		{"Spain", "Cape Verde Islands"},
		{"Portugal", "Congo DR"},
		{"Czechia", "South Africa"},
	}
	for _, c := range cases {
		t.Run(c.home+" vs "+c.away, func(t *testing.T) {
			client := NewClient(srv.URL, "k")
			o, err := client.GetForMatch(t.Context(), c.home, c.away, "2026-06-15T00:00:00Z")
			require.NoError(t, err)
			require.Greater(t, o.HomeOdds, 0.0, "home odds should be populated")
		})
	}
}

// TestGetForMatch_NonAliasedTeams verifies that teams whose names already
// match across both providers still resolve via the identity path.
func TestGetForMatch_NonAliasedTeams(t *testing.T) {
	body, err := os.ReadFile("../../testdata/odds-wc2026.json")
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "k")
	o, err := client.GetForMatch(t.Context(), "Mexico", "South Africa", "2026-06-15T00:00:00Z")
	require.NoError(t, err)
	require.Greater(t, o.HomeOdds, 0.0)
}
```

- [ ] **Step 2: Run the new tests to verify they fail**

Run:
```bash
cd backend && go test ./internal/odds/ -run 'TestGetForMatch_(Aliased|NonAliased)Teams' -v
```
Expected: `TestGetForMatch_NonAliasedTeams` passes (Mexico/South Africa already match verbatim), `TestGetForMatch_AliasedTeams` FAILS on every subtest with errors like `no odds found for United States vs Paraguay`.

- [ ] **Step 3: Patch `GetForMatch` to translate names**

In `backend/internal/odds/client.go`, replace the loop at lines 137-142 (currently `for _, e := range events { if e.HomeTeam == homeName && ... }`):

```go
	wantHome := oddsAPIName(homeName)
	wantAway := oddsAPIName(awayName)
	for _, e := range events {
		if e.HomeTeam == wantHome && e.AwayTeam == wantAway {
			return pickFirstH2H(e)
		}
	}
```

The trailing `return Odds{}, fmt.Errorf("no odds found for %s vs %s @ %s", homeName, awayName, kickoffUTC)` line stays unchanged — it still reports the original DB names in the error so traces remain readable.

- [ ] **Step 4: Run both new tests to verify they pass**

Run:
```bash
cd backend && go test ./internal/odds/ -run 'TestGetForMatch_(Aliased|NonAliased)Teams' -v
```
Expected: all six subtests (`United_States_vs_Paraguay`, `Canada_vs_Bosnia-Herzegovina`, `Spain_vs_Cape_Verde_Islands`, `Portugal_vs_Congo_DR`, `Czechia_vs_South_Africa`, plus the non-aliased pass) report `--- PASS`.

- [ ] **Step 5: Run the full backend test suite + vet**

Run:
```bash
cd backend && go vet ./... && go test ./...
```
Expected: zero `vet` output, every package reports `ok`. Pay particular attention to `internal/odds` and `internal/predict` — if either regresses, stop and diagnose.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/odds/client.go backend/internal/odds/client_test.go
git commit -m "$(cat <<'EOF'
fix(odds): translate team names to the-odds-api conventions

GetForMatch was doing exact, case-sensitive equality on names from two
providers that disagree on 5 of 48 WC teams (United States/USA,
Bosnia-Herzegovina/Bosnia & Herzegovina, Cape Verde Islands/Cape Verde,
Congo DR/DR Congo, Czechia/Czech Republic). Calls now route the home/away
args through oddsAPIName before the event-list comparison; the other 43
teams are unaffected via the identity fallback.
EOF
)"
```

---

## Verification after all tasks

- [ ] **Final sanity check**

Run:
```bash
cd backend && go vet ./... && go test ./... && make build
```
Expected: vet silent, all tests pass, `bin/wcp` rebuilds without error.

- [ ] **(Optional) Re-run a real prediction**

If a fixture is still upcoming (kickoff in the future), invoke `bin/wcp predict --match <id>` for a previously-failing match (e.g. `2026-06-13-USA-vs-PAR` if still pre-kickoff at execution time) and confirm `inputs_json.odds` is populated in the resulting row. This is verification, not part of the test suite — the alias map is fully proven by the unit + fixture tests above.

---

## Self-review notes

- **Spec coverage:** Each goal in `2026-06-13-odds-team-name-aliases-design.md` maps to a step here — alias map (Task 2.3), translation at call site (Task 3.3), fixture-grounded tests (Tasks 2.1 + 3.1), unchanged public surface (Task 3.3 keeps the signature), TDD (Steps 1-2 fail, Steps 3-4 pass).
- **Type/name consistency:** `oddsAPIName` is called identically in `alias.go` and `client.go`; `aliases` is the only package-level identifier introduced; the test names follow the existing `TestClient*` style except for the new `TestGetForMatch_*` pair which is more descriptive of the behavior under test.
- **Out-of-scope confirmed absent:** no fuzzy match, no ordering fallback, no warn-on-miss, no DB schema change — matches the spec's non-goals.
