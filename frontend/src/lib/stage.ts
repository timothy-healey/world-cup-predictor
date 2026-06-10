import type { Match } from "../types/api";

export function isGroupStageComplete(matches: Match[]): boolean {
  const group = matches.filter((m) => m.stage === "group");
  if (group.length === 0) return false;
  return group.every((m) => m.home_score !== null && m.away_score !== null);
}

export function isKnockoutMatch(stage: Match["stage"]): boolean {
  return stage !== "group";
}
