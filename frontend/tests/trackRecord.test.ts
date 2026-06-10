import { describe, expect, it } from "vitest";
import { trackRecord } from "../src/lib/trackRecord";
import type { Match, Prediction } from "../src/types/api";

const p = (over: Partial<Prediction>): Prediction => ({
  id: 1,
  created_at: "2026-06-11T15:00:00Z",
  trigger: "scheduled",
  confidence: "medium",
  predicted_winner: "MEX",
  predicted_score: "2-1",
  win_probability: 0.6,
  reasoning: "",
  model_id: "claude",
  variant: "full",
  trace: null,
  ...over,
});

const m = (over: Partial<Match>): Match => ({
  id: "2026-06-11-MEX-vs-CAN",
  home_team_code: "MEX",
  away_team_code: "CAN",
  kickoff_utc: "2026-06-11T16:00:00Z",
  stage: "group",
  venue: "",
  home_score: null,
  away_score: null,
  predictions: [],
  ...over,
});

describe("trackRecord", () => {
  it("returns zeros when no matches have results", () => {
    expect(trackRecord([m({ predictions: [p({})] })])).toEqual({
      total: 1,
      completed: 0,
      winnerCorrect: 0,
      exactCorrect: 0,
      winnerAccuracy: null,
      exactAccuracy: null,
      averageConfidence: 2,
    });
  });

  it("counts a correct winner with exact score", () => {
    const match = m({
      home_score: 2,
      away_score: 1,
      predictions: [p({ predicted_winner: "MEX", predicted_score: "2-1" })],
    });
    expect(trackRecord([match])).toMatchObject({
      completed: 1,
      winnerCorrect: 1,
      exactCorrect: 1,
      winnerAccuracy: 1,
      exactAccuracy: 1,
    });
  });

  it("counts a correct winner with wrong score", () => {
    const match = m({
      home_score: 3,
      away_score: 0,
      predictions: [p({ predicted_winner: "MEX", predicted_score: "2-1" })],
    });
    expect(trackRecord([match])).toMatchObject({
      completed: 1,
      winnerCorrect: 1,
      exactCorrect: 0,
    });
  });

  it("handles draw predictions vs draw outcomes", () => {
    const match = m({
      home_score: 1,
      away_score: 1,
      predictions: [p({ predicted_winner: "draw", predicted_score: "1-1" })],
    });
    expect(trackRecord([match])).toMatchObject({
      winnerCorrect: 1,
      exactCorrect: 1,
    });
  });

  it("uses the latest prediction per match (by created_at)", () => {
    const match = m({
      home_score: 0,
      away_score: 0,
      predictions: [
        p({ id: 1, created_at: "2026-06-10T10:00:00Z", predicted_winner: "MEX" }),
        p({ id: 2, created_at: "2026-06-11T10:00:00Z", predicted_winner: "draw" }),
      ],
    });
    expect(trackRecord([match]).winnerCorrect).toBe(1);
  });

  it("ignores matches with no predictions", () => {
    const match = m({ home_score: 1, away_score: 0, predictions: [] });
    expect(trackRecord([match])).toMatchObject({ total: 0, completed: 0 });
  });
});
