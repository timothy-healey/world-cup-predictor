import { describe, expect, it } from "vitest";
import { confidenceBadge, confidenceScore, averageConfidence } from "../src/lib/confidence";

describe("confidenceBadge", () => {
  it("returns the tone/label for each level", () => {
    expect(confidenceBadge("high")).toEqual({ label: "High", tone: "correct" });
    expect(confidenceBadge("medium")).toEqual({ label: "Medium", tone: "secondary" });
    expect(confidenceBadge("low")).toEqual({ label: "Low", tone: "wrong" });
  });
});

describe("confidenceScore", () => {
  it("maps levels to ordinals", () => {
    expect(confidenceScore("high")).toBe(3);
    expect(confidenceScore("medium")).toBe(2);
    expect(confidenceScore("low")).toBe(1);
  });
});

describe("averageConfidence", () => {
  it("returns null when there are no inputs", () => {
    expect(averageConfidence([])).toBeNull();
  });
  it("averages confidence scores", () => {
    expect(averageConfidence(["high", "medium", "medium", "low"])).toBeCloseTo(2.0);
  });
});
