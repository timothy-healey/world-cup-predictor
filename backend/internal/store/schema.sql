CREATE TABLE IF NOT EXISTS teams (
  code                  TEXT PRIMARY KEY,
  name                  TEXT NOT NULL,
  group_id              TEXT,
  flag_url              TEXT,
  fifa_ranking          INTEGER,
  manager_name          TEXT,
  pre_tournament_form   TEXT,
  fixture_src_id        TEXT
);

CREATE TABLE IF NOT EXISTS matches (
  id                 TEXT PRIMARY KEY,
  home_team_code     TEXT NOT NULL REFERENCES teams(code),
  away_team_code     TEXT NOT NULL REFERENCES teams(code),
  kickoff_utc        TEXT NOT NULL,
  stage              TEXT NOT NULL,
  venue              TEXT,
  fixture_src_id     TEXT,
  home_score         INTEGER,
  away_score         INTEGER,
  result_fetched_at  TEXT
);

CREATE INDEX IF NOT EXISTS idx_matches_kickoff ON matches(kickoff_utc);

CREATE TABLE IF NOT EXISTS predictions (
  id                 INTEGER PRIMARY KEY AUTOINCREMENT,
  match_id           TEXT NOT NULL REFERENCES matches(id),
  created_at         TEXT NOT NULL,
  trigger            TEXT NOT NULL CHECK (trigger IN ('scheduled', 'on_demand')),
  confidence         TEXT NOT NULL CHECK (confidence IN ('high', 'medium', 'low')),
  predicted_winner   TEXT NOT NULL,
  predicted_score    TEXT NOT NULL,
  win_probability    REAL,
  reasoning          TEXT NOT NULL,
  inputs_json        TEXT NOT NULL,
  rendered_prompt    TEXT NOT NULL,
  model_id           TEXT NOT NULL,
  prompt_version     TEXT NOT NULL,
  -- 'full' = production prediction using all available inputs.
  -- Other values (e.g. 'no-odds', 'no-news', 'no-lineup', 'no-context') reserved
  -- for the planned post-hoc experiment harness that re-predicts finished
  -- matches with masked input subsets to measure per-input contribution.
  variant            TEXT NOT NULL DEFAULT 'full'
);

CREATE INDEX IF NOT EXISTS idx_predictions_match ON predictions(match_id, created_at);
