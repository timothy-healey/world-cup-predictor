# Dashboard card click-to-expand Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Clicking an upcoming-match card on the Dashboard expands it inline to reveal the full prediction (winner / score / win prob / confidence / reasoning). The Upcoming-tab prediction card is rebuilt on a shared prediction-body layout, and the Past-tab card is refreshed to use full team names — all three surfaces become visually consistent.

**Architecture:** Extract two new presentational components (`<PredictionStats>`, `<PredictionReasoning>`) and one composite (`<PredictionBody>`) shared by the Dashboard's expanded `<MatchCard>` and the Upcoming-tab's `<PredictionCard>`. Dashboard owns the expansion state (`expandedId`). The grid layout shift is instant per `DESIGN.md` ("never animate CSS layout properties"); a quick opacity + translate animation reveals the expanded body on mount. Pure logic (team-name lookup, expand-state transitions) lives in `lib/` and is unit-tested; component composition is verified via manual smoke testing in dev mode — matching the existing frontend test pattern (`frontend/tests/*` only covers `lib/` modules).

**Tech Stack:** React 18, TypeScript, Tailwind (existing design tokens in `tailwind.config.ts`), Vitest for unit tests, Vite for build/dev server.

---

## File Structure

**New files:**

- `frontend/src/lib/teams.ts` — `buildTeamNameLookup` pure helper. Tested.
- `frontend/tests/teams.test.ts` — unit test for the lookup helper.
- `frontend/src/lib/expand.ts` — `nextExpandedId` pure helper for toggle state. Tested.
- `frontend/tests/expand.test.ts` — unit test for the expand-state helper.
- `frontend/src/components/PredictionStats.tsx` — shared stats panel (winner headline + score + win prob).
- `frontend/src/components/PredictionReasoning.tsx` — shared reasoning bullet list.
- `frontend/src/components/PredictionBody.tsx` — composes header + title + venue + split body + footer. Includes the mount-reveal animation.

**Modified files:**

- `frontend/src/components/MatchCard.tsx` — new `expanded` / `onToggle` / `teamName` props; clickable wrapper with keyboard support; renders `<PredictionBody>` when expanded.
- `frontend/src/components/PredictionCard.tsx` — reimplemented as a thin wrapper around `<PredictionBody variant="upcoming">`.
- `frontend/src/components/PastMatchCard.tsx` — uses full team names via the lookup; touched-up row styling.
- `frontend/src/pages/Dashboard.tsx` — owns `expandedId`; Esc handler; conditional grid layout when a card is expanded.
- `frontend/src/pages/Upcoming.tsx` — builds + passes `teamName` to `<PredictionCard>`.
- `frontend/src/pages/Past.tsx` — builds + passes `teamName` to `<PastMatchCard>`.
- `frontend/src/index.css` — `@keyframes` for the body reveal animation, gated by `prefers-reduced-motion`.
- `frontend/tailwind.config.ts` — optional: register the animation utility (decided per Task 5).

---

## Task 1: Team-name lookup helper

**Files:**
- Create: `frontend/src/lib/teams.ts`
- Create: `frontend/tests/teams.test.ts`

- [ ] **Step 1: Write the failing test**

Create `frontend/tests/teams.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { buildTeamNameLookup } from "../src/lib/teams";
import type { Team } from "../src/types/api";

function team(code: string, name: string): Team {
  return {
    code,
    name,
    group_id: "A",
    flag_url: "",
    fifa_ranking: 0,
    manager_name: "",
    pre_tournament_form: "",
    fixture_src_id: "",
  };
}

describe("buildTeamNameLookup", () => {
  const lookup = buildTeamNameLookup([
    team("BRA", "Brazil"),
    team("ESP", "Spain"),
  ]);

  it("returns the full name when the code matches", () => {
    expect(lookup("BRA")).toBe("Brazil");
    expect(lookup("ESP")).toBe("Spain");
  });

  it("returns the code unchanged when no team matches", () => {
    expect(lookup("XYZ")).toBe("XYZ");
  });

  it("returns 'Draw' for the special 'draw' value", () => {
    expect(lookup("draw")).toBe("Draw");
  });

  it("is case-sensitive on the code (matches the data convention)", () => {
    expect(lookup("bra")).toBe("bra");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npm test -- teams`
Expected: FAIL — module `../src/lib/teams` not found.

- [ ] **Step 3: Implement the helper**

Create `frontend/src/lib/teams.ts`:

```ts
import type { Team } from "../types/api";

// Builds a code → full name lookup from the teams array.
// Handles the special "draw" winner value by returning "Draw".
// Falls back to the input code when a team is not found (e.g. a TLA
// that did not resolve during bootstrap).
export function buildTeamNameLookup(teams: Team[]): (code: string) => string {
  const byCode = new Map<string, string>();
  for (const t of teams) byCode.set(t.code, t.name);
  return (code: string) => {
    if (code === "draw") return "Draw";
    return byCode.get(code) ?? code;
  };
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npm test -- teams`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/teams.ts frontend/tests/teams.test.ts
git -c commit.gpgsign=false commit -m "feat(frontend): add team-name lookup helper"
```

---

## Task 2: Expand-state toggle helper

**Files:**
- Create: `frontend/src/lib/expand.ts`
- Create: `frontend/tests/expand.test.ts`

- [ ] **Step 1: Write the failing test**

Create `frontend/tests/expand.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { nextExpandedId } from "../src/lib/expand";

describe("nextExpandedId", () => {
  it("opens the clicked id when nothing is expanded", () => {
    expect(nextExpandedId(null, "match-1")).toBe("match-1");
  });

  it("collapses when the clicked id is already expanded", () => {
    expect(nextExpandedId("match-1", "match-1")).toBeNull();
  });

  it("switches to the clicked id when a different one is expanded", () => {
    expect(nextExpandedId("match-1", "match-2")).toBe("match-2");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npm test -- expand`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement the helper**

Create `frontend/src/lib/expand.ts`:

```ts
// Computes the next expanded match id when a card is clicked.
// Clicking the currently-expanded card collapses it (returns null);
// clicking any other card opens that one.
export function nextExpandedId(
  current: string | null,
  clicked: string,
): string | null {
  return current === clicked ? null : clicked;
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npm test -- expand`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/expand.ts frontend/tests/expand.test.ts
git -c commit.gpgsign=false commit -m "feat(frontend): add expanded-card toggle helper"
```

---

## Task 3: `<PredictionStats>` component

The shared stats panel. Winner is the headline (full row inside the panel, primary color, large display weight). Score and win probability sit in a 2-up sub-grid below.

**Files:**
- Create: `frontend/src/components/PredictionStats.tsx`

- [ ] **Step 1: Implement the component**

Create `frontend/src/components/PredictionStats.tsx`:

```tsx
import type { Prediction } from "../types/api";

interface Props {
  prediction: Prediction;
  teamName: (code: string) => string;
}

export function PredictionStats({ prediction, teamName }: Props) {
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
          <div className="mb-1.5 text-2xs font-semibold uppercase tracking-label text-ink-3">
            Win probability
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

- [ ] **Step 2: Typecheck**

Run: `cd frontend && npx tsc -b --pretty`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/PredictionStats.tsx
git -c commit.gpgsign=false commit -m "feat(frontend): add shared PredictionStats panel"
```

---

## Task 4: `<PredictionReasoning>` component

**Files:**
- Create: `frontend/src/components/PredictionReasoning.tsx`

- [ ] **Step 1: Implement the component**

Create `frontend/src/components/PredictionReasoning.tsx`:

```tsx
import { parseReasoning } from "../lib/reasoning";

interface Props {
  reasoning: string;
}

export function PredictionReasoning({ reasoning }: Props) {
  const lines = parseReasoning(reasoning);
  if (lines.length === 0) return null;

  return (
    <div>
      <div className="mb-2.5 text-xs font-semibold uppercase tracking-label text-ink-3">
        Reasoning
      </div>
      <ul className="list-disc max-w-[62ch] pl-5 text-sm leading-relaxed text-ink">
        {lines.map((line, idx) => (
          <li key={idx} className="mb-1.5">
            {line}
          </li>
        ))}
      </ul>
    </div>
  );
}
```

- [ ] **Step 2: Typecheck**

Run: `cd frontend && npx tsc -b --pretty`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/PredictionReasoning.tsx
git -c commit.gpgsign=false commit -m "feat(frontend): add shared PredictionReasoning list"
```

---

## Task 5: Reveal animation CSS

A subtle opacity + translateY reveal for the mounted prediction body. Respects `prefers-reduced-motion`.

**Files:**
- Modify: `frontend/src/index.css`

- [ ] **Step 1: Inspect current contents**

Run: `cd frontend && head -40 src/index.css`
Expected: see the existing Tailwind directives and any existing custom CSS.

- [ ] **Step 2: Append the keyframes and utility class**

Add to the end of `frontend/src/index.css`:

```css
@keyframes wcp-reveal {
  from {
    opacity: 0;
    transform: translateY(-4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.wcp-reveal {
  animation: wcp-reveal 120ms ease-out both;
}

@media (prefers-reduced-motion: reduce) {
  .wcp-reveal {
    animation: none;
  }
}
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/index.css
git -c commit.gpgsign=false commit -m "feat(frontend): add prediction-body reveal animation"
```

---

## Task 6: `<PredictionBody>` composite

Header row (kickoff/countdown left, group + confidence right) → big title with full team names → venue → split body (stats left, reasoning right) → footer (actions left, optional collapse right). The `variant` prop toggles the collapse affordance.

**Files:**
- Create: `frontend/src/components/PredictionBody.tsx`

- [ ] **Step 1: Implement the component**

Create `frontend/src/components/PredictionBody.tsx`:

```tsx
import type { Match } from "../types/api";
import { latestPrediction } from "../lib/trackRecord";
import { flagFor } from "../data/flags";
import { formatKickoff, formatCountdown } from "../lib/format";
import { confidenceBadge } from "../lib/confidence";
import { Badge } from "./Badge";
import { Button } from "./Button";
import { Refresh, Zap } from "./icons";
import { PredictionStats } from "./PredictionStats";
import { PredictionReasoning } from "./PredictionReasoning";

interface Props {
  match: Match;
  teamName: (code: string) => string;
  variant: "dashboard" | "upcoming";
  groupLabel?: string;
  onPredict?: (matchID: string) => void;
  predictDisabled?: boolean;
  onCollapse?: () => void;
}

export function PredictionBody({
  match,
  teamName,
  variant,
  groupLabel,
  onPredict,
  predictDisabled,
  onCollapse,
}: Props) {
  const pred = latestPrediction(match);
  const ko = new Date(match.kickoff_utc);
  const homeName = teamName(match.home_team_code);
  const awayName = teamName(match.away_team_code);

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
        <div className="mt-6 grid grid-cols-1 gap-6 md:grid-cols-2 md:gap-8">
          <PredictionStats prediction={pred} teamName={teamName} />
          <PredictionReasoning reasoning={pred.reasoning} />
        </div>
      ) : (
        <div className="mt-6 rounded-md border border-dashed bg-surface-sunk px-5 py-4 text-sm text-ink-2">
          No prediction yet. The scheduled launchd agent will fire at T-30, or
          you can predict now manually.
        </div>
      )}

      {(onPredict || (variant === "dashboard" && onCollapse)) && (
        <div className="mt-6 flex items-center justify-between border-t pt-4">
          <div className="flex gap-2.5">
            {onPredict && (
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
            )}
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

- [ ] **Step 2: Typecheck**

Run: `cd frontend && npx tsc -b --pretty`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/PredictionBody.tsx
git -c commit.gpgsign=false commit -m "feat(frontend): add shared PredictionBody composite"
```

---

## Task 7: Refactor `<PredictionCard>` to use `<PredictionBody>`

Make the Upcoming-tab card a thin wrapper around the new shared body. The Upcoming page now builds and passes a `teamName` lookup.

**Files:**
- Modify: `frontend/src/components/PredictionCard.tsx`
- Modify: `frontend/src/pages/Upcoming.tsx`

- [ ] **Step 1: Replace `PredictionCard.tsx`**

Overwrite `frontend/src/components/PredictionCard.tsx` with:

```tsx
import type { Match } from "../types/api";
import { PredictionBody } from "./PredictionBody";

interface Props {
  match: Match;
  teamName: (code: string) => string;
  groupLabel?: string;
  onPredict?: (matchID: string) => void;
  predictDisabled?: boolean;
}

export function PredictionCard({
  match,
  teamName,
  groupLabel,
  onPredict,
  predictDisabled,
}: Props) {
  return (
    <article className="mb-3.5 rounded-lg border bg-surface px-6 py-5 sm:px-8 sm:py-6">
      <PredictionBody
        match={match}
        teamName={teamName}
        variant="upcoming"
        groupLabel={groupLabel}
        onPredict={onPredict}
        predictDisabled={predictDisabled}
      />
    </article>
  );
}
```

- [ ] **Step 2: Update `Upcoming.tsx` to pass `teamName`**

In `frontend/src/pages/Upcoming.tsx`, add the import and lookup:

```tsx
// at the top with other imports
import { buildTeamNameLookup } from "../lib/teams";
```

Then in the component body, add (next to the existing `teamGroup` block):

```tsx
const teamName = useMemo(() => buildTeamNameLookup(data.teams), [data.teams]);
```

(Add `useMemo` to the `react` import if it isn't already imported — it already is.)

And update each `<PredictionCard …/>` render to pass `teamName={teamName}`:

```tsx
<PredictionCard
  key={m.id}
  match={m}
  teamName={teamName}
  groupLabel={
    teamGroup[m.home_team_code]
      ? `Group ${teamGroup[m.home_team_code]} · ${stageLabel(m.stage)}`
      : stageLabel(m.stage)
  }
  onPredict={onPredict}
  predictDisabled={predictDisabled}
/>
```

- [ ] **Step 3: Typecheck**

Run: `cd frontend && npx tsc -b --pretty`
Expected: no errors.

- [ ] **Step 4: Manual smoke test in dev**

Run: `cd frontend && npm run dev` (and start the backend with sample data per `README.md`, or use the seeded `predictions.json`).
Navigate to http://localhost:5173, click the **Upcoming** tab.

Expected:
- Cards show full team names (e.g., "Brazil vs Spain"), not codes.
- Stats panel sits on the left, reasoning on the right (on viewports wider than `md`); stacked on narrower screens.
- Confidence badge shows in the top-right of the card header.
- Predict / Re-predict button still works.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/PredictionCard.tsx frontend/src/pages/Upcoming.tsx
git -c commit.gpgsign=false commit -m "refactor(frontend): rebuild PredictionCard on shared PredictionBody"
```

---

## Task 8: `<PastMatchCard>` with full names

Visual refresh only — full names in the title and in the per-row predicted winner. No expand/collapse interaction.

**Files:**
- Modify: `frontend/src/components/PastMatchCard.tsx`
- Modify: `frontend/src/pages/Past.tsx`

- [ ] **Step 1: Add `teamName` prop and use full names in the title + rows**

In `frontend/src/components/PastMatchCard.tsx`, replace the `Props` interface and the title + row rendering. The full updated file:

```tsx
import type { Match, Prediction } from "../types/api";
import { Badge } from "./Badge";
import { Check, X } from "./icons";
import { flagFor } from "../data/flags";
import { formatKickoff, formatScore, formatTimestamp } from "../lib/format";
import { confidenceBadge } from "../lib/confidence";
import { actualWinnerCode } from "../lib/outcome";
import { parseReasoning } from "../lib/reasoning";
import { stageLabel } from "../lib/stage";

interface Props {
  match: Match;
  teamName: (code: string) => string;
}

interface ActualOutcome {
  winner: string;
  score: string;
}

function verdict(p: Prediction, actual: ActualOutcome) {
  const winnerOk = p.predicted_winner === actual.winner;
  const scoreOk = p.predicted_score === actual.score;
  if (winnerOk && scoreOk) return { tone: "correct" as const, label: "Exact" };
  if (winnerOk) return { tone: "correct" as const, label: "Winner correct" };
  return { tone: "wrong" as const, label: "Wrong" };
}

export function PastMatchCard({ match, teamName }: Props) {
  const winner = actualWinnerCode(match);
  if (winner === null) return null;
  const score = formatScore(match.home_score, match.away_score);
  if (score === null) return null;
  const actual: ActualOutcome = { winner, score };
  const tintCorrect = match.predictions.some((p) => p.predicted_winner === winner);
  const surface = match.predictions.length === 0
    ? "bg-surface"
    : tintCorrect
      ? "bg-correct-soft/30"
      : "bg-wrong-soft/30";
  const sorted = [...match.predictions].sort((a, b) =>
    a.created_at < b.created_at ? 1 : -1,
  );
  const latest = sorted[0];

  return (
    <article className={`mb-3.5 rounded-lg border ${surface} px-6 py-5`}>
      <div className="grid grid-cols-[1fr_2fr] gap-6">
        <div>
          <div className="text-xs font-semibold uppercase tracking-label text-ink-3">
            {formatKickoff(match.kickoff_utc)} · {stageLabel(match.stage)}
          </div>
          <div className="mt-1.5 font-display text-2xl font-extrabold uppercase leading-none tracking-display text-ink">
            {flagFor(match.home_team_code)} {teamName(match.home_team_code)}
            <span className="mx-2 text-[0.65em] font-bold text-ink-4">vs</span>
            {flagFor(match.away_team_code)} {teamName(match.away_team_code)}
          </div>
          <div className="mt-3 font-display text-3xl font-black leading-none text-ink">
            {formatScore(match.home_score, match.away_score)}
          </div>
          {match.venue && <div className="mt-2 text-sm text-ink-2">{match.venue}</div>}
        </div>
        <div>
          <div className="mb-2 text-xs font-semibold uppercase tracking-label text-ink-3">
            Predictions ({sorted.length})
          </div>
          {sorted.length === 0 ? (
            <div className="text-sm italic text-ink-3">No prediction was made.</div>
          ) : (
            <ul className="space-y-1.5">
              {sorted.map((p) => {
                const v = verdict(p, actual);
                return (
                  <li
                    key={p.id}
                    className="flex items-center justify-between rounded-md border bg-surface px-3 py-1.5"
                  >
                    <div className="flex items-center gap-3 text-sm">
                      <span className="font-mono text-xs text-ink-3">
                        {formatTimestamp(p.created_at)}
                      </span>
                      <span className="font-display text-base font-extrabold uppercase text-ink">
                        {teamName(p.predicted_winner)} {p.predicted_score}
                      </span>
                      <Badge tone={confidenceBadge(p.confidence).tone}>
                        {confidenceBadge(p.confidence).label}
                      </Badge>
                      <span className="text-xs text-ink-3">
                        {p.trigger === "scheduled" ? "scheduled" : "on demand"}
                      </span>
                    </div>
                    {v.tone === "correct" ? (
                      <span className="flex items-center gap-1.5 text-sm font-semibold text-correct">
                        <Check size={14} /> {v.label}
                      </span>
                    ) : (
                      <span className="flex items-center gap-1.5 text-sm font-semibold text-wrong">
                        <X size={14} /> {v.label}
                      </span>
                    )}
                  </li>
                );
              })}
            </ul>
          )}
          {latest && (
            <div className="mt-4">
              <div className="mb-1.5 text-xs font-semibold uppercase tracking-label text-ink-3">
                Reasoning (latest)
              </div>
              <ul className="max-w-[62ch] list-disc pl-5 text-sm leading-relaxed">
                {parseReasoning(latest.reasoning).map((line, idx) => (
                  <li key={idx} className="mb-1">
                    {line}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      </div>
    </article>
  );
}
```

- [ ] **Step 2: Update `Past.tsx` to pass `teamName`**

In `frontend/src/pages/Past.tsx`, add to imports:

```tsx
import { buildTeamNameLookup } from "../lib/teams";
```

Then in the component body (add after the existing `useMemo` filter — or in its own `useMemo`):

```tsx
const teamName = useMemo(() => buildTeamNameLookup(data.teams), [data.teams]);
```

Make sure `useMemo` is imported (`import { useMemo, useState } from "react";` already exists at the top).

Update the `<PastMatchCard …/>` call:

```tsx
filtered.map((m) => <PastMatchCard key={m.id} match={m} teamName={teamName} />)
```

- [ ] **Step 3: Typecheck**

Run: `cd frontend && npx tsc -b --pretty`
Expected: no errors.

- [ ] **Step 4: Manual smoke test in dev**

Navigate to the **Past** tab.

Expected:
- Title shows full team names.
- Each prediction row's "winner score" segment shows the full name (e.g., "Brazil 2-1").
- All other behavior — verdict badge, reasoning, tint — unchanged.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/PastMatchCard.tsx frontend/src/pages/Past.tsx
git -c commit.gpgsign=false commit -m "refactor(frontend): use full team names on past match card"
```

---

## Task 9: `<MatchCard>` — expanded variant + clickable wrapper

Add `expanded` / `onToggle` / `teamName` props. When `expanded`, render `<PredictionBody variant="dashboard">`. When not, render today's compact body but make the whole card clickable (with proper a11y attributes and keyboard support). The internal Predict button stops event propagation.

**Files:**
- Modify: `frontend/src/components/MatchCard.tsx`

- [ ] **Step 1: Rewrite `MatchCard.tsx`**

Replace the contents of `frontend/src/components/MatchCard.tsx` with:

```tsx
import type { KeyboardEvent, MouseEvent } from "react";
import type { Match } from "../types/api";
import { latestPrediction } from "../lib/trackRecord";
import { Badge } from "./Badge";
import { Button } from "./Button";
import { Zap, Refresh } from "./icons";
import { flagFor } from "../data/flags";
import { formatKickoff, formatCountdown } from "../lib/format";
import { confidenceBadge } from "../lib/confidence";
import { PredictionBody } from "./PredictionBody";

interface Props {
  match: Match;
  variant?: "compact" | "next";
  groupLabel?: string;
  onPredict?: (matchID: string) => void;
  predictDisabled?: boolean;
  expanded?: boolean;
  onToggle?: () => void;
  teamName?: (code: string) => string;
}

export function MatchCard({
  match,
  variant = "compact",
  groupLabel,
  onPredict,
  predictDisabled,
  expanded = false,
  onToggle,
  teamName,
}: Props) {
  const ko = new Date(match.kickoff_utc);
  const now = new Date();
  const within10 = ko.getTime() - now.getTime() < 10 * 60 * 1000 && ko.getTime() > now.getTime();
  const pred = latestPrediction(match);
  const interactive = Boolean(onToggle);

  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (!onToggle) return;
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      onToggle();
    }
  };

  const handleClick = (e: MouseEvent<HTMLDivElement>) => {
    if (!onToggle) return;
    // Ignore clicks on interactive descendants (buttons handle their own events).
    if ((e.target as HTMLElement).closest("button")) return;
    onToggle();
  };

  if (expanded && teamName) {
    return (
      <div
        role="button"
        tabIndex={0}
        aria-expanded={true}
        aria-controls={`match-${match.id}-prediction`}
        id={`match-${match.id}-card`}
        onClick={handleClick}
        onKeyDown={handleKeyDown}
        className="rounded-lg border-2 border-ink bg-surface p-6 cursor-pointer focus:outline-none focus-visible:shadow-focus"
        style={{ gridColumn: "1 / -1" }}
      >
        <div id={`match-${match.id}-prediction`}>
          <PredictionBody
            match={match}
            teamName={teamName}
            variant="dashboard"
            groupLabel={groupLabel}
            onPredict={onPredict}
            predictDisabled={predictDisabled}
            onCollapse={onToggle}
          />
        </div>
      </div>
    );
  }

  const teamSize = variant === "next" ? "text-display-lg" : "text-xl";
  const wrapperRole = interactive ? "button" : undefined;
  const wrapperTabIndex = interactive ? 0 : undefined;

  return (
    <div
      role={wrapperRole}
      tabIndex={wrapperTabIndex}
      aria-expanded={interactive ? false : undefined}
      aria-controls={interactive ? `match-${match.id}-prediction` : undefined}
      onClick={handleClick}
      onKeyDown={handleKeyDown}
      className={`flex flex-col gap-2 rounded-lg border bg-surface ${
        variant === "next" ? "border-ink p-5" : "p-4"
      } ${interactive ? "cursor-pointer transition-colors hover:border-ink focus:outline-none focus-visible:shadow-focus" : ""}`}
    >
      <div className="text-xs font-semibold uppercase tracking-label-mid text-ink-3">
        <span
          className={`mr-1.5 inline-block h-1.5 w-1.5 align-middle rounded-pill bg-pending ${
            within10 ? "animate-pulse !bg-primary" : ""
          }`}
        />
        {formatKickoff(match.kickoff_utc)} · {formatCountdown(ko, now)}
      </div>
      <div
        className={`font-display font-extrabold uppercase tracking-display text-ink leading-none ${teamSize}`}
      >
        {flagFor(match.home_team_code)} {match.home_team_code}{" "}
        <span className="text-ink-4 font-bold text-[0.7em] mx-1">vs</span>{" "}
        {flagFor(match.away_team_code)} {match.away_team_code}
      </div>
      {(groupLabel || match.venue) && (
        <div className="text-xs uppercase tracking-label-mid font-medium text-ink-3">
          {[groupLabel, match.venue].filter(Boolean).join(" · ")}
        </div>
      )}
      <div className="mt-auto flex items-center justify-between pt-2">
        {pred ? (
          <Badge tone={confidenceBadge(pred.confidence).tone}>Predicted</Badge>
        ) : (
          <Badge tone="pending">T-30 scheduled</Badge>
        )}
        {onPredict && (
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
        )}
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Typecheck**

Run: `cd frontend && npx tsc -b --pretty`
Expected: no errors. (Dashboard.tsx still imports the old MatchCard props; will be fixed in Task 10.)

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/MatchCard.tsx
git -c commit.gpgsign=false commit -m "feat(frontend): add expanded variant + click handling to MatchCard"
```

---

## Task 10: Dashboard — own `expandedId`, Esc handler, adapted grid

Dashboard owns the expansion state. The grid splits into two zones when something is expanded: the expanded card on its own row (spanning all columns) plus the remaining cards in a sibling row that uses an even 2-column layout so the un-expanded cards stay balanced.

**Files:**
- Modify: `frontend/src/pages/Dashboard.tsx`

- [ ] **Step 1: Rewrite `Dashboard.tsx`**

Replace the contents of `frontend/src/pages/Dashboard.tsx` with:

```tsx
import { useEffect, useMemo, useState } from "react";
import type { ExportPayload } from "../types/api";
import { TrackRecord } from "../components/TrackRecord";
import { MatchCard } from "../components/MatchCard";
import { GroupStandings } from "../components/GroupStandings";
import { KnockoutBracket } from "../components/KnockoutBracket";
import { isGroupStageComplete } from "../lib/stage";
import { buildTeamNameLookup } from "../lib/teams";
import { nextExpandedId } from "../lib/expand";

const UPCOMING_GRID = "1.6fr 1fr 1fr";
const COMPACT_REMAINDER_GRID = "1fr 1fr";

interface Props {
  data: ExportPayload;
  onPredict: (matchID: string) => void;
  predictDisabled: boolean;
}

export function Dashboard({ data, onPredict, predictDisabled }: Props) {
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const upcoming = useMemo(() => {
    const now = new Date().toISOString();
    return data.matches
      .filter((m) => m.kickoff_utc > now)
      .sort((a, b) => (a.kickoff_utc < b.kickoff_utc ? -1 : 1))
      .slice(0, 3);
  }, [data.matches]);

  const teamName = useMemo(() => buildTeamNameLookup(data.teams), [data.teams]);

  const teamGroup: Record<string, string> = {};
  for (const t of data.teams) teamGroup[t.code] = t.group_id;

  // If the currently expanded match leaves the upcoming window, drop it.
  useEffect(() => {
    if (expandedId && !upcoming.some((m) => m.id === expandedId)) {
      setExpandedId(null);
    }
  }, [expandedId, upcoming]);

  // Esc collapses any expanded card.
  useEffect(() => {
    if (expandedId === null) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setExpandedId(null);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [expandedId]);

  const groupStageDone = isGroupStageComplete(data.matches);
  const expandedMatch = expandedId
    ? upcoming.find((m) => m.id === expandedId) ?? null
    : null;
  const compactRemainder = upcoming.filter((m) => m.id !== expandedId);

  return (
    <div className="bg-bg px-7 py-7">
      <TrackRecord matches={data.matches} />

      <section className="mb-8">
        <header className="mb-3.5 flex items-baseline justify-between">
          <h2 className="text-xs font-semibold uppercase tracking-label text-primary">
            Upcoming matches
          </h2>
          <div className="text-sm text-ink-3">soonest first</div>
        </header>

        {upcoming.length === 0 ? (
          <div className="rounded-lg border bg-surface p-6 text-center text-sm text-ink-3">
            No upcoming matches. Re-run <code className="font-mono">wcp bootstrap</code> if the tournament is in progress.
          </div>
        ) : expandedMatch ? (
          <div className="flex flex-col gap-3.5">
            <MatchCard
              key={expandedMatch.id}
              match={expandedMatch}
              expanded
              teamName={teamName}
              onToggle={() => setExpandedId(null)}
              groupLabel={
                teamGroup[expandedMatch.home_team_code]
                  ? `Group ${teamGroup[expandedMatch.home_team_code]}`
                  : undefined
              }
              onPredict={onPredict}
              predictDisabled={predictDisabled}
            />
            <div
              className="grid gap-3.5"
              style={{ gridTemplateColumns: COMPACT_REMAINDER_GRID }}
            >
              {compactRemainder.map((m) => (
                <MatchCard
                  key={m.id}
                  match={m}
                  variant="compact"
                  teamName={teamName}
                  groupLabel={
                    teamGroup[m.home_team_code]
                      ? `Group ${teamGroup[m.home_team_code]}`
                      : undefined
                  }
                  onPredict={onPredict}
                  predictDisabled={predictDisabled}
                  onToggle={() => setExpandedId(nextExpandedId(expandedId, m.id))}
                />
              ))}
            </div>
          </div>
        ) : (
          <div
            className="grid gap-3.5"
            style={{ gridTemplateColumns: UPCOMING_GRID }}
          >
            {upcoming.map((m, idx) => (
              <MatchCard
                key={m.id}
                match={m}
                variant={idx === 0 ? "next" : "compact"}
                teamName={teamName}
                groupLabel={
                  teamGroup[m.home_team_code]
                    ? `Group ${teamGroup[m.home_team_code]}`
                    : undefined
                }
                onPredict={onPredict}
                predictDisabled={predictDisabled}
                onToggle={() => setExpandedId(nextExpandedId(expandedId, m.id))}
              />
            ))}
          </div>
        )}
      </section>

      <section>
        {groupStageDone ? (
          <KnockoutBracket matches={data.matches} />
        ) : (
          <GroupStandings teams={data.teams} matches={data.matches} />
        )}
      </section>
    </div>
  );
}
```

- [ ] **Step 2: Typecheck**

Run: `cd frontend && npx tsc -b --pretty`
Expected: no errors.

- [ ] **Step 3: Manual smoke test**

Run: `cd frontend && npm run dev`. Make sure the backend is providing `predictions.json` (or use seeded data). Navigate to the Dashboard.

Expected:
- All three upcoming cards behave like buttons (cursor pointer, keyboard focusable, hover darkens the border).
- Clicking a card expands it inline — the clicked card spans the full row, the other two drop to a second row with equal widths.
- The expanded card shows the full prediction body: kickoff/countdown, group + confidence badge, full team names, venue, stats panel on the left (with "Brazil" or whoever in the primary color as the headline), reasoning on the right, footer with Re-predict + Collapse.
- Animation: the expanded body fades and slides in (~120ms).
- Click the same expanded card again → collapses back to the 3-up grid.
- Click a different (compact) card while one is expanded → it becomes the expanded one; previously-expanded collapses.
- Press **Esc** while a card is expanded → collapses.
- Click the "Predict now" / "Re-predict" button on any card → triggers the prediction request but does NOT toggle expand.
- Keyboard navigation: Tab to a card, press Enter or Space → expands. Tab to the Re-predict button inside → activates that button only.

- [ ] **Step 4: Run all tests**

Run: `cd frontend && npm test`
Expected: all tests pass — both new (`teams`, `expand`) and existing.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/pages/Dashboard.tsx
git -c commit.gpgsign=false commit -m "feat(frontend): click-to-expand upcoming match cards on dashboard"
```

---

## Task 11: Full build, sanity, ship

**Files:** none — verification only.

- [ ] **Step 1: Full typecheck + build**

Run: `cd frontend && npm run build`
Expected: TypeScript compiles cleanly and Vite emits `dist/` with no errors.

- [ ] **Step 2: All tests**

Run: `cd frontend && npm test`
Expected: every test in `frontend/tests/` passes (including new `teams.test.ts` and `expand.test.ts`).

- [ ] **Step 3: End-to-end smoke**

Run: `cd frontend && npm run dev`. With the backend providing predictions data, walk through:
1. Dashboard: cards collapsed → click → expanded → click another → switches → Esc → collapsed.
2. Upcoming tab: every card uses the new two-column body with full team names.
3. Past tab: full team names in titles and prediction rows; all other info still visible.
4. Reduced-motion check: enable "Reduce motion" in your OS accessibility settings; expansion should be instant (no reveal animation).

- [ ] **Step 4: No commit needed if everything passed.**

If any tweaks were necessary during smoke, commit them as a single follow-up:

```bash
git add -p   # stage the deltas explicitly
git -c commit.gpgsign=false commit -m "polish(frontend): card-click-to-expand smoke-test follow-ups"
```

---

## Notes for the implementer

- **Design tokens.** This feature uses existing Tailwind tokens — `bg-surface`, `bg-surface-sunk`, `text-ink`, `text-ink-2`, `text-ink-3`, `text-primary`, `text-secondary`, `tracking-display`, `tracking-label`, `tracking-label-mid`, `text-display-lg`, `text-2xs`, etc. They're all already defined in `frontend/tailwind.config.ts`. If any class produces a Tailwind warning, double-check the token name against the config rather than inventing a new utility.
- **TDD scope.** Per existing project convention, only pure logic in `lib/` is unit-tested (`frontend/tests/*`). React components are verified via the manual smoke tests in Tasks 7, 8, and 10. Don't add React component tests in this plan — that's a separate infra decision (`jsdom` + `@testing-library/react`) the project hasn't made yet.
- **Animation rule.** Per `DESIGN.md` ("Never animate CSS layout properties"), the grid reflow itself is instant. The `wcp-reveal` class only animates `opacity` and `transform`. Don't reach for `height` / `max-height` transitions.
- **Stop-propagation pattern.** The Predict / Re-predict button and the Collapse button inside the expanded card must call `e.stopPropagation()` in their `onClick` so they don't bubble up and re-toggle the card. The Task 6 (`PredictionBody`) and Task 9 (`MatchCard`) code already does this — keep it that way.
- **Confidence badge wording.** In the expanded body the badge reads "High confidence" / "Medium confidence" / "Low confidence" (label + word). The compact `MatchCard` keeps the plain "Predicted" badge tinted by confidence as today. Don't change the compact badge here — it's intentional.
- **Grid math.** When something is expanded, the remaining 0–2 cards render in a 2-column sub-grid. With one remainder the lone card naturally fills its column (the other is empty). That's acceptable; we keep the grid because it preserves the column rhythm visually.
- **Edge case — match leaves the upcoming window.** If a match's kick-off passes while it's expanded (e.g., user leaves the tab open), the `useEffect` cleanup in Dashboard collapses it automatically on the next `upcoming` recomputation. That's deliberate.
