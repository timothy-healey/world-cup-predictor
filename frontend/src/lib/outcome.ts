import type { Match } from "../types/api";

// Returns the winning team code, "draw", or null if the match has no result yet.
// Used by track-record stats, the Past tab filters, and the bracket verdict footer.
export function actualWinnerCode(m: Match): string | null {
  if (m.home_score === null || m.away_score === null) return null;
  if (m.home_score > m.away_score) return m.home_team_code;
  if (m.away_score > m.home_score) return m.away_team_code;
  return "draw";
}

// Returns which side won ("home" | "away" | "draw"), or null if no result.
// Used by the bracket's team-cell styling (winner ink-strong, loser strike-through).
export function actualWinnerSide(m: Match): "home" | "away" | "draw" | null {
  if (m.home_score === null || m.away_score === null) return null;
  if (m.home_score > m.away_score) return "home";
  if (m.away_score > m.home_score) return "away";
  return "draw";
}
