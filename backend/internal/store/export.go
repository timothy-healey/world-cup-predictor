package store

import (
	"encoding/json"
	"os"
)

type ExportPrediction struct {
	ID              int64            `json:"id"`
	CreatedAt       string           `json:"created_at"`
	Trigger         string           `json:"trigger"`
	Confidence      string           `json:"confidence"`
	PredictedWinner string           `json:"predicted_winner"`
	PredictedScore  string           `json:"predicted_score"`
	WinProbability  float64          `json:"win_probability"`
	Reasoning       string           `json:"reasoning"`
	ModelID         string           `json:"model_id"`
	Variant         string           `json:"variant"`
	Trace           *json.RawMessage `json:"trace"` // pointer so encoding/json emits null for nil
}

type ExportMatch struct {
	ID           string             `json:"id"`
	HomeTeamCode string             `json:"home_team_code"`
	AwayTeamCode string             `json:"away_team_code"`
	KickoffUTC   string             `json:"kickoff_utc"`
	Stage        string             `json:"stage"`
	Venue        string             `json:"venue"`
	HomeScore    *int               `json:"home_score"`
	AwayScore    *int               `json:"away_score"`
	Predictions  []ExportPrediction `json:"predictions"`
}

type ExportPayload struct {
	GeneratedAt string        `json:"generated_at"`
	Teams       []Team        `json:"teams"`
	Matches     []ExportMatch `json:"matches"`
}

func (s *Store) ExportJSON(path string) error {
	teams, err := s.ListTeams()
	if err != nil {
		return err
	}
	matches, err := s.ListMatches()
	if err != nil {
		return err
	}
	payload := ExportPayload{
		GeneratedAt: nowUTC(),
		Teams:       teams,
		Matches:     make([]ExportMatch, 0, len(matches)),
	}
	for _, m := range matches {
		preds, _ := s.ListPredictionsByMatch(m.ID)
		em := ExportMatch{
			ID: m.ID, HomeTeamCode: m.HomeTeamCode, AwayTeamCode: m.AwayTeamCode,
			KickoffUTC: m.KickoffUTC, Stage: m.Stage, Venue: m.Venue,
			HomeScore: m.HomeScore, AwayScore: m.AwayScore,
			// Always emit an empty array (never JSON null) so the frontend can
			// treat the field as a guaranteed []Prediction without null-guarding
			// every consumer.
			Predictions: []ExportPrediction{},
		}
		for _, p := range preds {
			ep := ExportPrediction{
				ID: p.ID, CreatedAt: p.CreatedAt, Trigger: p.Trigger,
				Confidence: p.Confidence, PredictedWinner: p.PredictedWinner,
				PredictedScore: p.PredictedScore, WinProbability: p.WinProbability,
				Reasoning: p.Reasoning, ModelID: p.ModelID, Variant: p.Variant,
			}
			if p.TraceJSON != "" {
				raw := json.RawMessage(p.TraceJSON)
				ep.Trace = &raw
			}
			em.Predictions = append(em.Predictions, ep)
		}
		payload.Matches = append(payload.Matches, em)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}
