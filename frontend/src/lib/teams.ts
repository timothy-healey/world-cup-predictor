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
