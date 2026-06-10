import type { Confidence, Match, Prediction } from "../types/api";
import { averageConfidence } from "./confidence";
import { actualWinnerCode } from "./outcome";

export interface TrackRecord {
  total: number;
  completed: number;
  winnerCorrect: number;
  exactCorrect: number;
  winnerAccuracy: number | null;
  exactAccuracy: number | null;
  averageConfidence: number | null;
}

export function latestPrediction(m: Match): Prediction | null {
  if (m.predictions.length === 0) return null;
  return [...m.predictions].sort((a, b) => (a.created_at < b.created_at ? 1 : -1))[0];
}

export function trackRecord(matches: Match[]): TrackRecord {
  let total = 0;
  let completed = 0;
  let winnerCorrect = 0;
  let exactCorrect = 0;
  const confidences: Confidence[] = [];

  for (const m of matches) {
    const pred = latestPrediction(m);
    if (!pred) continue;
    total += 1;
    confidences.push(pred.confidence);
    const actual = actualWinnerCode(m);
    if (actual === null) continue;
    completed += 1;
    if (pred.predicted_winner === actual) winnerCorrect += 1;
    if (pred.predicted_score === `${m.home_score}-${m.away_score}`) exactCorrect += 1;
  }

  return {
    total,
    completed,
    winnerCorrect,
    exactCorrect,
    winnerAccuracy: completed === 0 ? null : winnerCorrect / completed,
    exactAccuracy: completed === 0 ? null : exactCorrect / completed,
    averageConfidence: averageConfidence(confidences),
  };
}
