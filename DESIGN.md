# Design

Visual system for the World Cup Predictor dashboard. Single light theme, "Floodlight" palette (mustard + magenta on cream), Big Shoulders Display + Inter type pairing. Treats the dashboard like a printed tournament programme rather than a sports-media broadcast.

## Visual Theme

**Name:** Floodlight
**Mood:** A printed tournament programme on a cream paper stock. Bold display type does the heavy lifting; restrained card chrome stays out of the way. The cream background and warm dark ink give a tactile, slightly tactile-printed feel that sets this apart from default-white SaaS dashboards. Saturated identity colors (mustard, magenta) carry energy in 1-2 places per surface; everything else is tinted neutral. Calm composition, loud type, occasional bright moments.

**Anti-references in effect** (from PRODUCT.md): no SaaS-cream tile soup, no ESPN clutter, no AI-bro neon, no Bloomberg density.

## Color

**Strategy: Committed.** Magenta is the identity hue — used for stage labels, winner highlights, and key data emphasis. Mustard is the secondary, used for the success-state contrast on dark buttons and for high-energy data points (probabilities, big scores). Everything else is tinted-warm neutral. Backgrounds are cream, never white. Ink is warm-dark, never `#000`.

### Tokens (light theme)

```
/* Brand */
--color-primary           #DD2680   oklch(56% 0.22 0)        Magenta. Stage labels, predicted winner, key data emphasis, focus rings.
--color-primary-soft      #FBE4F0   oklch(94% 0.04 0)        Primary backgrounds for selected states, focus highlights.
--color-secondary         #F0BA2C   oklch(80% 0.15 85)       Mustard. Text on dark CTAs, high-energy data points, score emphasis.
--color-secondary-soft    #FBE9C8   oklch(92% 0.05 85)       Backgrounds for "medium confidence" badge fills.

/* Surfaces — cream stock, warm */
--color-bg                #FAF5EB   oklch(96.5% 0.012 80)    Page background.
--color-surface           #FFFCF5   oklch(98.5% 0.008 80)    Card / panel background.
--color-surface-sunk      #F3EBD8   oklch(93% 0.018 80)      Embedded "well" surfaces (prediction blocks inside cards).

/* Ink — warm dark, tinted toward primary */
--color-ink               #1B0E12   oklch(13% 0.015 10)      Primary text, dark CTAs.
--color-ink-2             #4A3940   oklch(34% 0.018 10)      Secondary text.
--color-ink-3             #847078   oklch(58% 0.014 10)      Tertiary / meta text.
--color-ink-4             #BCB0B5   oklch(76% 0.010 10)      Disabled / placeholder.

/* Outcomes — semantic, paired with icons (✓ / ✗) */
--color-correct           #1B7E55   oklch(50% 0.13 155)      ✓ winner correct, advancing in group, completed match.
--color-correct-soft      #D9F1E3   oklch(92% 0.045 155)
--color-wrong             #C73848   oklch(55% 0.16 22)       ✗ prediction wrong.
--color-wrong-soft        #FBE0E2   oklch(92% 0.03 22)
--color-pending           #4A6BD1   oklch(50% 0.15 268)      Scheduled / awaiting kickoff. Indigo, not magenta — distinct from "prediction made".
--color-pending-soft      #DFE4FA   oklch(92% 0.04 268)

/* Borders — hairline, ink-tinted */
--color-border            rgba(27,14,18,0.10)
--color-border-strong     rgba(27,14,18,0.20)
--color-border-dashed     rgba(27,14,18,0.18)
```

### Usage rules

- **Magenta is the identity color.** Use on stage labels, predicted-winner emphasis, primary focus rings, the page-title bar accent. Don't use it for body text.
- **Mustard is the secondary accent.** Use as text on dark CTAs and for one high-energy data point per card (typically the win probability). Don't use mustard as a background on light cream — too low contrast.
- **Semantic colors are reserved.** Green only for correct / advancing. Red only for wrong / eliminated. Indigo only for "scheduled, not yet predicted". Don't repurpose them.
- **Outcomes pair color with an icon** (✓ / ✗), never color alone.
- **Borders are hairline.** Use the standard `--color-border` token; reach for `border-strong` only when grouping is genuinely ambiguous.

## Typography

Family pairing: **Big Shoulders Display** (display) + **Inter** (body). Big Shoulders is a condensed athletic-American display face that gives team matchups and big numbers a confident, programme-like presence; Inter handles the rest with quiet competence.

### Loading

Both via Google Fonts. Preconnect + display swap; preload the display weights actually used (700 / 800 / 900).

```html
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Big+Shoulders+Display:wght@700;800;900&family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
```

### Tokens

```
--font-display            "Big Shoulders Display", system-ui, sans-serif
--font-body               "Inter", system-ui, -apple-system, sans-serif
--font-mono               ui-monospace, "SF Mono", "DM Mono", monospace   /* only for raw odds */

/* Type scale — body sizes are Inter; display sizes are Big Shoulders */
--text-xs       11px / 1.4    Inter 600   Labels, meta
--text-sm       12px / 1.5    Inter 500   Filters, badges, dropdowns
--text-base     14px / 1.55   Inter 400   Body, reasoning bullets
--text-md       15px / 1.5    Inter 500   Match card titles in compact view
--text-lg       18px / 1.35   Inter 600   Section headings
--text-xl       22px / 1.2    Display 800 Stat tile numbers
--text-2xl      28px / 1.1    Display 800 Match name on prediction cards
--text-3xl      40px / 1.05   Display 900 Actual score on past-match cards
--text-hero     56px / 1.0    Display 900 Top-of-page accent (e.g. tournament title bar)

/* Letter-spacing — Big Shoulders is condensed; slight negative tracking on largest sizes */
--tracking-display       -0.01em    Default for display text
--tracking-display-tight -0.02em    Use on hero sizes
--tracking-label          0.08em    Uppercase labels (stage, section labels)
```

### Usage rules

- **Display face is for team names, scorelines, big numbers, section labels.** Always uppercase or title-case — never lowercase paragraphs. Don't run Big Shoulders past a few words; it's a display face, not body.
- **Inter is for everything else** — body copy, reasoning bullets, filters, kickoff timestamps, button labels.
- **Body line length stays under 75ch.** Reasoning bullets typically run 60-70ch.
- **Hierarchy comes from scale + weight contrast.** Big Shoulders 900 next to Inter 400 is the headline-vs-body relationship; don't try to do it within one family.
- **No lowercase Big Shoulders for matchups.** "ARGENTINA vs SAUDI ARABIA" reads as a tournament programme; "Argentina vs Saudi Arabia" loses the personality.

## Layout

### Spacing scale

4px base unit. Use the named token, not raw pixel values.

```
--space-0    0
--space-px   1px
--space-1    4px
--space-2    8px
--space-3    12px
--space-4    16px
--space-5    20px
--space-6    24px
--space-8    32px
--space-10   40px
--space-12   48px
--space-16   64px
--space-20   80px
```

### Rhythm rules

- **Vary padding by component weight.** Prediction cards: 18-20px. Compact match cards: 14-16px. Stat tiles: 14-16px. Group tables: 10-12px.
- **No nested cards.** If you feel the need for a card-inside-a-card, use a sunk surface (`--color-surface-sunk`) and a 1px border instead.
- **Group standings grid is 4 columns** (12 groups → 4×3). Match cards are 3 columns. Stat tiles are 4 columns. Don't make these all the same column count — variance is part of the rhythm.

### Radii

```
--radius-sm     4px      Tabs underline, small badges
--radius        6px      Buttons, inputs, dropdowns
--radius-md     8px      Compact match cards, badges-with-content, stat tiles
--radius-lg     12px     Full prediction cards, past-match cards
--radius-xl     16px     Hero containers (rare)
--radius-pill   999px    Status badges, filter pills
```

### Borders

```
--border-hairline   1px solid var(--color-border)         Default card / surface border
--border-strong     1px solid var(--color-border-strong)  Use for emphasis only
--border-dashed     1px dashed var(--color-border-dashed) Internal dividers within cards
```

**No side-stripe borders.** No `border-left: 4px solid <accent>` patterns anywhere.

### Elevation

Mostly flat. Borders carry structure. The dashboard is a programme on paper, not a stack of cards.

```
--shadow-none    none                                                   Default
--shadow-hover   0 2px 8px -2px rgba(27,14,18,0.08)                     On hover / focus of interactive cards only
--shadow-focus   0 0 0 3px var(--color-primary-soft)                    Keyboard focus ring (also wraps with 2px ink outline)
```

No drop shadows on default state. No glass / blur effects.

## Components

### Buttons

**Primary (`Predict now`):** dark ink background, mustard text. The mustard-on-ink contrast is the strongest in the system — reserved for the headline call to action.

```
background: var(--color-ink)
color: var(--color-secondary)
padding: 9px 16px
border-radius: var(--radius)
font: Inter 600 13px
icon: 13px stroke, inline-flex gap 6px
```

**Ghost (`Re-predict`):** cream surface, ink border, ink text.

```
background: var(--color-surface)
color: var(--color-ink)
border: 1px solid var(--color-border-strong)
padding: 8px 14px
```

**Disabled:** opacity 0.5, cursor not-allowed. No state color change.

### Badges

Status pills carrying confidence or scheduling state. Always include a dot or icon — never color alone.

```
font: Inter 600 11px
padding: 4px 10px
border-radius: var(--radius-pill)

.high:       bg var(--color-correct-soft)    text var(--color-correct)
.medium:     bg var(--color-secondary-soft)  text #7A5910
.low:        bg var(--color-wrong-soft)      text var(--color-wrong)
.scheduled:  bg var(--color-pending-soft)    text var(--color-pending)
```

### Cards

Default: cream surface, hairline border, no shadow, `--radius-lg` for full prediction cards / `--radius-md` for compact.

```
background: var(--color-surface)
border: var(--border-hairline)
border-radius: var(--radius-lg)
padding: 18-20px
```

Inside a prediction card, the "prediction details" block uses `--color-surface-sunk` and `--radius-md` to create depth without nesting cards.

### Tabs

Underline tabs (no pill background). Active tab gets a 2px ink underline; inactive tabs are `--color-ink-3`.

```
display: flex; gap: 4px
font: Inter 500 13px
padding: 10px 16px
border-bottom: 2px solid transparent

&.active   color: var(--color-ink)    border-color: var(--color-ink)
&:hover    color: var(--color-ink)
```

### Group standings table

Compact 4-column tables in a 4×3 grid. Use display face for the team code; body face for stats. Top-2 (advancing) rows get a green dot and ink-strong color.

### Bracket cell

Match cells in the knockout bracket use `--radius-md`, `--color-surface`, hairline border. Winners are ink-strong, losers are `--color-ink-4`, TBD entries are `--color-ink-4` italic. Each cell has a dashed-divider footer showing prediction verdict.

## Motion

Minimal product motion. The dashboard should feel snappy and confident, not animated.

```
--ease-out-quart   cubic-bezier(0.25, 1, 0.5, 1)
--ease-out-expo    cubic-bezier(0.16, 1, 0.3, 1)

--duration-fast    120ms   Filter pill toggles, hover state changes
--duration         150ms   Tab switches, panel reveals
--duration-slow    240ms   Page-level transitions (rare)
```

### Rules

- **Never animate CSS layout properties** (width, height, top, etc.). Use transform/opacity.
- **No bounce or elastic curves.** Ease-out only.
- **Respect `prefers-reduced-motion: reduce`** — drop transitions to `0ms` and disable any non-essential animation.
- **One delightful moment per page.** A small pulse on the next-up match card when it's <10 minutes from kick-off is fine. Multiple competing animations are not.

## Iconography

Inline SVG only. Lucide-style stroke (2-2.4px stroke width, square caps, square joins). Country flags are emoji — the only emoji in the app.

```
--ic-stroke      2px
--ic-stroke-sm   2.4px
--ic-size        14px     Default in buttons and labels
--ic-size-sm     11px     In badges and small contexts
```

In-system icons used: `zap` (predict), `refresh` (re-predict), `check` (correct), `x` (wrong), `clock` (kickoff time). Add icons only when they add information; don't sprinkle decoratively.

## Open

- **Print stylesheet for the email** — when predictions are mailed out, the HTML email body should mirror this palette and type (with web-safe fallbacks for Big Shoulders → system condensed sans). Specified at implementation time.
- **Empty states** — designed once real surfaces are built. Should lean into the programme metaphor (a printed-but-blank fixture card waiting to be filled in).
