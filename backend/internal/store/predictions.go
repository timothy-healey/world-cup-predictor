package store

import "database/sql"

type Prediction struct {
	ID              int64   `json:"id"`
	MatchID         string  `json:"match_id"`
	CreatedAt       string  `json:"created_at"`
	Trigger         string  `json:"trigger"`          // "scheduled" | "on_demand"
	Confidence      string  `json:"confidence"`       // "high" | "medium" | "low"
	PredictedWinner string  `json:"predicted_winner"` // team code or "draw"
	PredictedScore  string  `json:"predicted_score"`  // e.g. "2-1"
	WinProbability  float64 `json:"win_probability"`
	Reasoning       string  `json:"reasoning"`
	InputsJSON      string  `json:"inputs_json"`
	RenderedPrompt  string  `json:"rendered_prompt"`
	ModelID         string  `json:"model_id"`
	PromptVersion   string  `json:"prompt_version"`
	Variant         string  `json:"variant"` // "full" for production runs; named subset (e.g. "no-odds") for ablation experiments
}

func (s *Store) InsertPrediction(p Prediction) (int64, error) {
	variant := p.Variant
	if variant == "" {
		variant = "full"
	}
	res, err := s.db.Exec(
		`INSERT INTO predictions (match_id, created_at, trigger, confidence, predicted_winner, predicted_score, win_probability, reasoning, inputs_json, rendered_prompt, model_id, prompt_version, variant)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.MatchID, p.CreatedAt, p.Trigger, p.Confidence, p.PredictedWinner, p.PredictedScore,
		p.WinProbability, p.Reasoning, p.InputsJSON, p.RenderedPrompt, p.ModelID, p.PromptVersion, variant,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ListPredictionsByMatch(matchID string) ([]Prediction, error) {
	rows, err := s.db.Query(
		`SELECT id, match_id, created_at, trigger, confidence, predicted_winner, predicted_score, win_probability, reasoning, inputs_json, rendered_prompt, model_id, prompt_version, variant
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
			&p.InputsJSON, &p.RenderedPrompt, &p.ModelID, &p.PromptVersion, &p.Variant); err != nil {
			return nil, err
		}
		p.WinProbability = prob.Float64
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) ListLatestPredictions(limit int) ([]Prediction, error) {
	rows, err := s.db.Query(
		`SELECT id, match_id, created_at, trigger, confidence, predicted_winner, predicted_score, win_probability, reasoning, inputs_json, rendered_prompt, model_id, prompt_version, variant
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
			&p.InputsJSON, &p.RenderedPrompt, &p.ModelID, &p.PromptVersion, &p.Variant); err != nil {
			return nil, err
		}
		p.WinProbability = prob.Float64
		out = append(out, p)
	}
	return out, rows.Err()
}
