package store

import "database/sql"

type Prediction struct {
	ID              int64
	MatchID         string
	CreatedAt       string
	Trigger         string // "scheduled" | "on_demand"
	Confidence      string // "high" | "medium" | "low"
	PredictedWinner string // team code or "draw"
	PredictedScore  string // e.g. "2-1"
	WinProbability  float64
	Reasoning       string
	InputsJSON      string
	RenderedPrompt  string
	ModelID         string
	PromptVersion   string
}

func (s *Store) InsertPrediction(p Prediction) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO predictions (match_id, created_at, trigger, confidence, predicted_winner, predicted_score, win_probability, reasoning, inputs_json, rendered_prompt, model_id, prompt_version)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.MatchID, p.CreatedAt, p.Trigger, p.Confidence, p.PredictedWinner, p.PredictedScore,
		p.WinProbability, p.Reasoning, p.InputsJSON, p.RenderedPrompt, p.ModelID, p.PromptVersion,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ListPredictionsByMatch(matchID string) ([]Prediction, error) {
	rows, err := s.db.Query(
		`SELECT id, match_id, created_at, trigger, confidence, predicted_winner, predicted_score, win_probability, reasoning, inputs_json, rendered_prompt, model_id, prompt_version
		 FROM predictions WHERE match_id = ? ORDER BY created_at DESC`,
		matchID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Prediction
	for rows.Next() {
		var p Prediction
		var prob sql.NullFloat64
		if err := rows.Scan(&p.ID, &p.MatchID, &p.CreatedAt, &p.Trigger, &p.Confidence,
			&p.PredictedWinner, &p.PredictedScore, &prob, &p.Reasoning,
			&p.InputsJSON, &p.RenderedPrompt, &p.ModelID, &p.PromptVersion); err != nil {
			return nil, err
		}
		p.WinProbability = prob.Float64
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) ListLatestPredictions(limit int) ([]Prediction, error) {
	rows, err := s.db.Query(
		`SELECT id, match_id, created_at, trigger, confidence, predicted_winner, predicted_score, win_probability, reasoning, inputs_json, rendered_prompt, model_id, prompt_version
		 FROM predictions ORDER BY created_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Prediction
	for rows.Next() {
		var p Prediction
		var prob sql.NullFloat64
		if err := rows.Scan(&p.ID, &p.MatchID, &p.CreatedAt, &p.Trigger, &p.Confidence,
			&p.PredictedWinner, &p.PredictedScore, &prob, &p.Reasoning,
			&p.InputsJSON, &p.RenderedPrompt, &p.ModelID, &p.PromptVersion); err != nil {
			return nil, err
		}
		p.WinProbability = prob.Float64
		out = append(out, p)
	}
	return out, rows.Err()
}
