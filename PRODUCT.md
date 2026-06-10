# Product

## Register

product

## Users

A single user (Tim) running the tool on his own Mac in Australia for the 2026 FIFA World Cup. He is a software engineer and a football fan. Most matches kick off in his small hours (4:30am local), so he often won't see the live prediction email when it fires — he checks the dashboard when he wakes up or during the day. The dashboard is also where he reviews how the predictor is performing across the tournament. Single user, local-only, no auth, no other readers. The repo is shareable: if a friend clones it, they should be able to drop their own credentials in `.env` and run it themselves.

## Product Purpose

A local-first dashboard for an LLM-driven World Cup match predictor. Surfaces three things and only three things well: (1) what's coming up in the next day, (2) the current state of the tournament (group standings during the group stage, knockout bracket once it begins), and (3) a track record of past predictions vs. actual results. Reasoning behind each prediction is rendered as plain-English bullet points, not raw LLM prose. Success looks like the user opening the dashboard on a quiet Tuesday afternoon and immediately being able to (a) see what's on tonight, (b) trust how the predictor is doing, and (c) skim the reasoning behind a past prediction without scrolling.

## Brand Personality

**Bold, kinetic, considered.** Gameday energy without broadcast clutter — confident color and type, but composition stays calm. A bit of fun and a bit of fizz; not stadium-loud, more pub-on-a-good-day. Treats predictions seriously enough to be useful but never solemn — this is a hobby tool, not a financial terminal.

## Anti-references

**All four common reflexes are off-limits.** This is the unusually strong constraint of this project.

- **No generic SaaS-cream dashboard.** No beige cards, no soft pastels, no friendly-rounded stat tiles. The Loom/Linear-imitator look is the default failure mode for product dashboards and it is explicitly excluded.
- **No ESPN-style sports-media clutter.** No ticker bars, no ad-shaped sponsor panels, no oversized team crests crowding match cards, no broadcast-graphic chrome.
- **No crypto/AI-bro neon-on-black.** No glowing gradients, no electric accents, no drenched dark mode with magenta highlights.
- **No Bloomberg terminal density.** No monospace-everywhere, no zero-padding tables, no eye-strain utility. There has to be breathing room.

The design must clearly avoid all four. Anyone glancing at the dashboard should not be able to slot it into "sports app", "SaaS dashboard", "AI tool", or "trading terminal" by reflex.

## Design Principles

1. **Predictions are the headline.** The prediction itself (winner / score / confidence) is the headline of every card. Reasoning supports it. Don't hide the call under chrome.
2. **Show what's at stake.** Kickoff countdowns, group context, bracket progress, track-record stats — every surface should feel like a tournament is actually underway. The dashboard is never neutral.
3. **Bold type, calm composition.** Hierarchy comes from a strong type scale and confident weight contrast, not from boxes-in-boxes. Less reliance on cards-of-cards.
4. **Gameday color, used sparingly.** Saturated brand color carries identity in 1-2 places per screen; the rest is restrained. Color earns its keep — used for outcome semantics (correct / wrong / pending) and for moments of energy, not as wallpaper.
5. **Plain-English reasoning.** Predictions explain themselves in short bullet points, never long LLM prose blocks. If a bullet can't be skimmed in two seconds, it's too long.

## Accessibility & Inclusion

Single-user, single-machine context — no special accessibility requirements beyond sensible WCAG AA contrast. Standard system font is acceptable. Outcomes (correct / wrong) should pair color with an icon (✓ / ✗) so the signal isn't carried by hue alone — a habit, not a requirement.
