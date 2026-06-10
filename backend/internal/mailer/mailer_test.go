package mailer

import (
	"strings"
	"testing"

	"github.com/timhealey/world-cup-predictor/backend/internal/store"

	"github.com/stretchr/testify/require"
)

func TestRenderEmail(t *testing.T) {
	m := store.Match{
		ID: "m1", HomeTeamCode: "ARG", AwayTeamCode: "SAU",
		KickoffUTC: "2026-06-25T11:00:00Z", Stage: "group",
	}
	p := store.Prediction{
		PredictedWinner: "ARG", PredictedScore: "2-0",
		WinProbability: 0.71, Confidence: "medium",
		Reasoning: "Argentina dominant\n- Messi in form\n- Saudi missing keeper",
		ModelID:   "claude-opus-4-7", CreatedAt: "2026-06-25T10:30:00Z",
	}
	subject, body := RenderEmail(m, p)
	require.Contains(t, subject, "ARG vs SAU")
	require.Contains(t, subject, "[partial]") // medium confidence triggers partial tag
	require.Contains(t, body, "Predicted winner: ARG")
	require.Contains(t, body, "Confidence: medium")
	require.Contains(t, body, "Messi in form")
}

func TestRenderEmailHighConfidence(t *testing.T) {
	m := store.Match{ID: "m1", HomeTeamCode: "ARG", AwayTeamCode: "SAU", KickoffUTC: "2026-06-25T11:00:00Z"}
	p := store.Prediction{
		PredictedWinner: "ARG", PredictedScore: "2-0",
		WinProbability: 0.71, Confidence: "high",
		Reasoning: "r", ModelID: "m",
	}
	subject, _ := RenderEmail(m, p)
	require.False(t, strings.Contains(subject, "[partial]"))
}
