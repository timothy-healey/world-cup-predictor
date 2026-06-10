# Dashboard card click-to-expand — Design

**Date:** 2026-06-11
**Status:** Draft (pending user review)

## Purpose

Clicking an upcoming-match card on the Dashboard reveals that match's full prediction (winner, score, win probability, confidence, reasoning) inline — without switching tabs or opening a modal. As part of the same change, the Upcoming-tab prediction card is rebalanced to use full team names and a roomier two-column body, and the Past-tab card gets a visual refresh so all three surfaces share one prediction-body layout.

## Goals

- Surface predictions from the Dashboard with one click, without losing Dashboard context (no tab switch, no modal overlay).
- Use one consistent prediction-body layout across Dashboard inline-expanded, Upcoming-tab card, and Past-tab card.
- Use full team names (from `Team.name`) in the expanded body and on the Upcoming/Past tabs. Compact dashboard cards (collapsed state) keep showing 3-letter codes.
- Quick, restrained motion (per DESIGN.md "never animate CSS layout properties" rule).

## Non-goals

- No change to the Past-tab interaction model — those cards remain always-expanded (currently they show actual result + every prediction + reasoning inline). The change there is purely visual.
- No backend changes. The data already contains everything needed.
- No deep-linking (e.g., `?match=<id>`) for the expanded state. Local UI state only.
- No "multiple cards expanded at once" — only one expanded card at a time on the Dashboard.

## Interaction model

### Dashboard (upcoming section)

- **Collapsed state** — three `MatchCard`s in a 1.6fr / 1fr / 1fr grid, as today. Codes (e.g., "BRA"), kickoff line, "Predicted" / "T-30" badge, predict button.
- **Click** anywhere on a card body (except action buttons) toggles its expanded state.
- **Expanded state** — the clicked card grows to span all 3 columns (`grid-column: 1 / -1`). The other two sibling cards reflow to a second row, staying as `MatchCard`s in compact form.
- Only one card can be expanded at a time. Clicking another card collapses the current one and expands the new one.
- A "▴ Collapse" affordance in the expanded card's footer collapses it. **Esc** key also collapses.
- The "Predict now" / "Re-predict" button keeps its existing behavior; clicks stop propagation so they don't toggle expand.

### Upcoming tab

- Cards are already full-width. They just adopt the new prediction body (full names, two-column split, larger stats panel). No expand/collapse — they're always expanded.

### Past tab

- Cards stay always-expanded as today. Visual refresh: full team names, the same `<PredictionStats>` block used elsewhere, restyled prediction-list rows. The 1fr/2fr "result vs predictions" split stays.

## Layout

### The expanded prediction body (shared)

```
┌─────────────────────────────────────────────────────────────────────────┐
│  ● Sat 13 Jun · 8 PM ET · in 2h 14m       Group B · MD2  [High conf]    │
│                                                                          │
│  🇧🇷 Brazil  vs  🇪🇸 Spain                                              │
│  MetLife Stadium · East Rutherford, NJ                                  │
│                                                                          │
│  ┌──────────────────────────┐  ┌────────────────────────────────────┐  │
│  │ Predicted winner         │  │ Reasoning                          │  │
│  │ Brazil   ← primary color │  │ • Brazil unbeaten last 5…          │  │
│  │                          │  │ • Rodri ruled out 24h ago…         │  │
│  │ Score        Win prob    │  │ • Bookmaker consensus ~1.85…       │  │
│  │ 2–1          58% (under) │  │ • H2H: Brazil 4 of last 6…         │  │
│  └──────────────────────────┘  └────────────────────────────────────┘  │
│                                                                          │
│  ─────────────────────────────────────────────────────────────────────  │
│  [↻ Re-predict]                                              ▴ Collapse │
└─────────────────────────────────────────────────────────────────────────┘
```

- **Top row:** kickoff/countdown on the left; group/matchday + confidence badge on the right.
- **Title:** full team names (e.g., "Brazil vs Spain") with flag emoji, display weight 900, ~38px.
- **Venue:** small body text under the title.
- **Split body (the key layout decision):**
  - Left half: stats panel, tinted background (`bg-surface-sunk`-like). 3 stats arranged with **winner as the headline** (full row width inside the panel), then **score and win probability** in a 2-up row below.
  - Right half: reasoning bullets, `max-w-[62ch]` for readability.
- **Footer row** with a top border: actions on the left (Re-predict, plus optional "View full report →" when we later add a per-match deep-view), collapse affordance on the right (Dashboard only).

### Past-tab card

- Header line uses full names: "Brazil vs Spain" instead of "BRA vs ESP".
- The compact actual-score block on the left (today: kickoff/stage, names, final score, venue) stays in place but adopts full team names.
- The "Predictions (n)" list on the right keeps today's compact one-line row structure (timestamp · winner+score · confidence badge · trigger · verdict). The only change to those rows is that the winner is rendered as a full name instead of a code (e.g., "Brazil 2–1" not "BRA 2–1"). These list rows are intentionally **not** restyled to use `<PredictionStats>` — that component is for the full prediction body only.
- The "Reasoning (latest)" section stays as-is in markup; its surrounding spacing is touched up to match the new prediction-body's rhythm.

## Component design

### New shared components

- **`<PredictionStats match={...} teamName={...} />`** — renders the stats panel: predicted-winner headline (primary color, full name via lookup), then score + win prob in a 2-up sub-grid. Used by Dashboard expanded card, Upcoming card, and Past card list rows (a compact one-line variant).
- **`<PredictionReasoning prediction={...} />`** — renders the reasoning bullets list. Trivial component; mostly here so the bullet/label styling lives in one place.
- **`<PredictionBody match={...} variant="dashboard"|"upcoming" teamName={...} onPredict={...} onCollapse={...} />`** — composes the header row, title, venue, split body, and footer. Used by the Dashboard expanded card and the Upcoming-tab card. The `variant` prop toggles the "Collapse" affordance and any minor differences.

### Modified components

- **`MatchCard.tsx`**
  - New props: `expanded?: boolean`, `onToggle?: () => void`, `teamName?: (code: string) => string`.
  - When `expanded`: render `<PredictionBody variant="dashboard" />` instead of the compact body.
  - When not expanded: render today's compact body, but wrap the card so it's clickable (role="button", `aria-expanded`, keyboard handlers).
  - The internal action button stops propagation so it still works without toggling expand.
  - Variants `"compact"` and `"next"` stay, plus the orthogonal `expanded` flag.

- **`PredictionCard.tsx`**
  - Reimplemented as a thin wrapper around `<PredictionBody variant="upcoming" />`. The current ad-hoc layout is replaced.

- **`PastMatchCard.tsx`**
  - Uses `teamName` lookup for the title.
  - Replaces the inline `<li>` row contents with a compact `<PredictionStats variant="row">` (or equivalent inline form).
  - Keeps the actual-result block and the verdict badge logic as-is.

### State management

- Expanded-card state lives in **`Dashboard.tsx`**: `const [expandedId, setExpandedId] = useState<string | null>(null)`.
- Reasoning: only Dashboard knows the grid context where one card spans columns. `MatchCard` shouldn't own state that affects sibling layout.
- Clicking a card calls `setExpandedId(id === expandedId ? null : id)`.
- Esc handling: `useEffect` on `Dashboard` listens for keydown when `expandedId !== null`.

### Team-name lookup

- `Dashboard`, `Upcoming`, and `Past` already build a per-team code → group lookup. We extend the pattern with a `code → name` lookup, derived inline from `data.teams`.
- Passed as a `(code: string) => string` function. If a code isn't in the map (rare — happens for unresolved TLAs), fall back to the code itself.
- Could later be lifted into `useData.ts` if more surfaces need it; not necessary now.

## Animation

Per `DESIGN.md`: "Never animate CSS layout properties." The grid column-span change is therefore **instant** — no animated transition on the grid layout itself.

- **On expand:**
  - State updates → grid layout shifts immediately (clicked card spans the row, siblings drop to row 2).
  - The expanded body content fades + slides in: `opacity 0 → 1`, `transform: translateY(-4px) → 0`, duration **120 ms**, `ease-out`.
- **On collapse:**
  - Same content animates out: `opacity 1 → 0`, duration **80 ms**. Once the animation completes (or in the same frame, since 80 ms is acceptable jank-free), state updates and the grid reverts.
  - Simpler alternative: revert layout immediately, no out-animation. Trade-off: feels more abrupt on collapse but skips one piece of timing logic.

Implementation: a CSS class with a `keyframes` or `transition` declaration on `opacity` and `transform`, applied via a `key` change or a `useEffect`-driven class toggle on mount of `<PredictionBody>`.

## Accessibility

- Each clickable card gets `role="button"`, `tabIndex={0}`, `aria-expanded={isExpanded}`, and `aria-controls` referencing the expanded panel's `id` (the panel's `id` is `match-<matchId>-prediction`).
- Keyboard: **Enter** and **Space** toggle expand. **Esc** collapses when a card is expanded; focus returns to the card that was expanded.
- Action buttons inside the card have `onClick={(e) => { e.stopPropagation(); /* existing */ }}`.
- Focus ring: existing `focus-visible:shadow-focus` utility applies; ensure it's visible on the wrapping clickable element, not just on inner buttons.

## Data model — what we're working with

`Prediction` has exactly: `confidence`, `predicted_winner` (team code or `"draw"`), `predicted_score`, `win_probability`, `reasoning`. The "Goal range" stat used in some mockup iterations does not exist in the data and is **not** part of the design.

`Team.name` exists and is the source for full names.

## Empty / edge states

- **No prediction yet on a clicked card:** expanded body shows the existing "No prediction yet. The scheduled launchd agent will fire at T-30, or you can predict now manually." copy, plus the Predict-now button.
- **Predicted winner is `"draw"`:** the stats panel renders "Draw" in the primary-color slot. (Today's `PredictionCard` already handles this implicitly.)
- **No upcoming matches:** the existing empty-state message in `Dashboard.tsx` is unchanged — there's nothing to click.
- **Reasoning empty (string parses to zero bullets):** omit the Reasoning section.

## Files changed

| File | Change |
| --- | --- |
| `frontend/src/components/PredictionStats.tsx` | **NEW** — shared stats panel |
| `frontend/src/components/PredictionReasoning.tsx` | **NEW** — shared reasoning list |
| `frontend/src/components/PredictionBody.tsx` | **NEW** — composes header + title + split body + footer |
| `frontend/src/components/MatchCard.tsx` | Add expanded variant; clickable + keyboard support |
| `frontend/src/components/PredictionCard.tsx` | Replace body with `<PredictionBody variant="upcoming" />` |
| `frontend/src/components/PastMatchCard.tsx` | Full names; restyled list rows using shared atoms |
| `frontend/src/pages/Dashboard.tsx` | Owns `expandedId`; renders expanded `MatchCard` accordingly; Esc handler |
| `frontend/src/pages/Upcoming.tsx` | Pass `teamName` lookup to `PredictionCard` |
| `frontend/src/pages/Past.tsx` | Pass `teamName` lookup to `PastMatchCard` |
| `frontend/src/index.css` | A tiny keyframes/transition for the body reveal |

## Testing

- **Vitest** tests:
  - `Dashboard`: clicking a `MatchCard` expands it; clicking again collapses; clicking another collapses the first and expands the second.
  - `Dashboard`: pressing **Esc** when a card is expanded collapses it; nothing happens when none is expanded.
  - `MatchCard`: clicking the Predict button does **not** toggle expanded state (event propagation halted).
  - `MatchCard`: pressing **Enter** or **Space** on the wrapping element toggles expand.
  - `PredictionStats`: renders predicted-winner using the team-name lookup; renders "Draw" when `predicted_winner === "draw"`.
  - `PastMatchCard`: renders full team names in the title.
- **Manual smoke test** in dev server:
  - Click each of the three Dashboard cards, verify reflow.
  - Verify the animation feels right ("quick"). Tweak the duration if not.
  - Verify the Upcoming-tab card looks visually identical to the Dashboard expanded body.
  - Verify the Past-tab card looks like the new style without losing any info that's currently visible.

## Open decisions

- **Sub-layout of the 3 stats inside the stats panel.** Default proposed here is "winner as headline" (Predicted Winner spans the panel's full inner width as a large primary-color word; Score and Win probability sit below in a 2-up sub-grid). The alternative is a plain 1-column stack (winner → score → win prob). The user did not explicitly pick between these — flagging here for confirmation during spec review.

All other choices are confirmed (Dashboard inline-expand chosen over modal / tab-navigate; full names; split body with stats on the left; quick animation; scope = Dashboard + Upcoming refresh + Past visual refresh).
