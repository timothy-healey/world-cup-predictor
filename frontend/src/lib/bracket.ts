import type { Match, Stage } from "../types/api";

const BRACKET_STAGES: Stage[] = [
  "round-of-32",
  "round-of-16",
  "qf",
  "sf",
  "final",
];

export interface BracketColumn {
  stage: Stage;
  label: string;
  matches: Match[];
}

export interface Bracket {
  columns: BracketColumn[];
  thirdPlace: Match | null;
}

const LABELS: Record<Stage, string> = {
  group: "Group",
  "round-of-32": "Round of 32",
  "round-of-16": "Round of 16",
  qf: "Quarter-final",
  sf: "Semi-final",
  final: "Final",
  "third-place": "Third-place",
};

export function buildBracket(matches: Match[]): Bracket {
  const columns: BracketColumn[] = BRACKET_STAGES.map((stage) => ({
    stage,
    label: LABELS[stage],
    matches: [],
  }));
  let thirdPlace: Match | null = null;

  for (const match of matches) {
    if (match.stage === "third-place") {
      thirdPlace = match;
      continue;
    }
    const col = columns.find((c) => c.stage === match.stage);
    if (!col) continue;
    col.matches.push(match);
  }

  for (const col of columns) {
    col.matches.sort((x, y) => (x.kickoff_utc < y.kickoff_utc ? -1 : 1));
  }

  return { columns, thirdPlace };
}
