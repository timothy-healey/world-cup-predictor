export type Stage =
  | "group"
  | "round-of-32"
  | "round-of-16"
  | "qf"
  | "sf"
  | "final"
  | "third-place";

export type Confidence = "high" | "medium" | "low";
export type Trigger = "scheduled" | "on_demand";

export interface Team {
  code: string;
  name: string;
  group_id: string;
  flag_url: string;
  fifa_ranking: number;
  manager_name: string;
  pre_tournament_form: string;
  fixture_src_id: string;
}

export interface Prediction {
  id: number;
  created_at: string;
  trigger: Trigger;
  confidence: Confidence;
  predicted_winner: string; // team code OR "draw"
  predicted_score: string; // e.g. "2-1"
  win_probability: number;
  reasoning: string;
  model_id: string;
  // "full" = production prediction. Other values (e.g. "no-odds", "no-news")
  // are reserved for the planned post-hoc ablation experiment harness.
  // The dashboard should filter to variant === "full" everywhere unless
  // an experiment-comparison surface is built.
  variant: string;
}

export interface Match {
  id: string;
  home_team_code: string;
  away_team_code: string;
  kickoff_utc: string; // ISO 8601
  stage: Stage;
  venue: string;
  home_score: number | null;
  away_score: number | null;
  predictions: Prediction[];
}

export interface ExportPayload {
  generated_at: string;
  teams: Team[];
  matches: Match[];
}
