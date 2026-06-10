import type { Confidence } from "../types/api";

export type BadgeTone = "correct" | "secondary" | "wrong" | "pending";

export function confidenceBadge(c: Confidence): { label: string; tone: BadgeTone } {
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
