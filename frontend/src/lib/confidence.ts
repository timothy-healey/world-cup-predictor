import type { Confidence } from "../types/api";

// The three tones confidenceBadge can actually return — a structural subset of
// the Badge component's BadgeTone union. Kept local (not exported) so the only
// named BadgeTone in the codebase lives with the Badge component.
type ConfidenceTone = "correct" | "secondary" | "wrong";

export function confidenceBadge(c: Confidence): { label: string; tone: ConfidenceTone } {
  switch (c) {
    case "high":
      return { label: "High", tone: "correct" };
    case "medium":
      return { label: "Medium", tone: "secondary" };
    case "low":
      return { label: "Low", tone: "wrong" };
  }
}

export function confidenceScore(c: Confidence): number {
  switch (c) {
    case "high":
      return 3;
    case "medium":
      return 2;
    case "low":
      return 1;
  }
}

export function averageConfidence(items: Confidence[]): number | null {
  if (items.length === 0) return null;
  const sum = items.reduce((acc, c) => acc + confidenceScore(c), 0);
  return sum / items.length;
}
