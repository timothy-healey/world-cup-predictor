package mailer

import (
	"fmt"
	"strings"

	"github.com/timhealey/world-cup-predictor/backend/internal/store"
)

func RenderEmail(m store.Match, p store.Prediction) (subject, body string) {
	tag := ""
	if p.Confidence != "high" {
		tag = "[partial] "
	}
	subject = fmt.Sprintf("%s%s vs %s — predicted %s %s",
		tag, m.HomeTeamCode, m.AwayTeamCode, p.PredictedWinner, p.PredictedScore)

	var b strings.Builder
	fmt.Fprintf(&b, "Match: %s vs %s\n", m.HomeTeamCode, m.AwayTeamCode)
	fmt.Fprintf(&b, "Stage: %s\n", m.Stage)
	fmt.Fprintf(&b, "Kickoff (UTC): %s\n\n", m.KickoffUTC)
	fmt.Fprintf(&b, "Predicted winner: %s\n", p.PredictedWinner)
	fmt.Fprintf(&b, "Predicted score: %s\n", p.PredictedScore)
	fmt.Fprintf(&b, "Win probability: %.0f%%\n", p.WinProbability*100)
	fmt.Fprintf(&b, "Confidence: %s\n\n", p.Confidence)
	fmt.Fprintf(&b, "Reasoning:\n%s\n\n", p.Reasoning)
	fmt.Fprintf(&b, "Model: %s\nGenerated: %s\n", p.ModelID, p.CreatedAt)
	return subject, b.String()
}
