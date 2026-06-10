import type { Match, Stage } from "../types/api";

export function isGroupStageComplete(matches: Match[]): boolean {
  const group = matches.filter((m) => m.stage === "group");
  if (group.length === 0) return false;
  return group.every((m) => m.home_score !== null && m.away_score !== null);
}

export function isKnockoutMatch(stage: Match["stage"]): boolean {
  return stage !== "group";
}

const STAGE_LABEL: Record<Stage, string> = {
  group: "Group stage",
  "round-of-32": "Round of 32",
  "round-of-16": "Round of 16",
  qf: "Quarter-final",
  sf: "Semi-final",
  final: "Final",
  "third-place": "Third-place playoff",
};

export function stageLabel(stage: Stage): string {
  return STAGE_LABEL[stage];
}
