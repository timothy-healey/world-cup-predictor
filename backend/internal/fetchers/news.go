package fetchers

import (
	"context"
	"fmt"
)

type NewsResult struct {
	HomeSummary string `json:"home_summary"`
	AwaySummary string `json:"away_summary"`
}

func FetchNews(ctx context.Context, d claudeBin, home, away string) (NewsResult, error) {
	prompt := fmt.Sprintf(`Summarize the most relevant football news for %s and %s
from the last 14 days. Focus on: injuries, suspensions, form, off-field issues
that could affect the match. Keep each summary under 80 words.

Reply with ONLY this JSON, no prose:
{
  "home_summary": "...",
  "away_summary": "..."
}`, home, away)
	return runJSON[NewsResult](ctx, d, "news", prompt)
}
