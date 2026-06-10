package predict

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/timhealey/world-cup-predictor/backend/internal/claudec"
	"github.com/timhealey/world-cup-predictor/backend/internal/fetchers"
	"github.com/timhealey/world-cup-predictor/backend/internal/store"
)

// Deps holds injectable fetcher functions for testing.
type Deps struct {
	FetchOdds    func(ctx context.Context, homeName, awayName, kickoff string) (any, bool)
	FetchNews    func(ctx context.Context, d any, home, away string) (fetchers.NewsResult, bool)
	FetchLineup  func(ctx context.Context, d any, home, away string) (fetchers.LineupResult, bool)
	FetchContext func(s *store.Store, homeCode, awayCode string) (fetchers.ContextResult, bool)
}

type Pipeline struct {
	store         *store.Store
	claude        *claudec.Driver
	deps          Deps
	systemPrompt  string
	promptVersion string
}

func New(s *store.Store, d *claudec.Driver, deps Deps) *Pipeline {
	sysPrompt, version := loadSystemPrompt()
	return &Pipeline{store: s, claude: d, deps: deps, systemPrompt: sysPrompt, promptVersion: version}
}

func loadSystemPrompt() (string, string) {
	body, err := os.ReadFile("./prompts/predict.md")
	if err != nil {
		return "Predict the winner and scoreline as JSON.", "fallback"
	}
	// Use a content-hash as the version (cheap, change-detectable).
	return string(body), fmt.Sprintf("len-%d", len(body))
}

func (p *Pipeline) Run(ctx context.Context, matchID, trigger string) (store.Prediction, error) {
	m, err := p.store.GetMatch(matchID)
	if err != nil {
		return store.Prediction{}, fmt.Errorf("get match: %w", err)
	}
	home, _ := p.store.GetTeam(m.HomeTeamCode)
	away, _ := p.store.GetTeam(m.AwayTeamCode)

	// Run fetchers in parallel via goroutines.
	type oddsR struct {
		data any
		ok   bool
	}
	type newsR struct {
		data fetchers.NewsResult
		ok   bool
	}
	type lineupR struct {
		data fetchers.LineupResult
		ok   bool
	}
	type ctxR struct {
		data fetchers.ContextResult
		ok   bool
	}

	oCh := make(chan oddsR, 1)
	nCh := make(chan newsR, 1)
	lCh := make(chan lineupR, 1)
	cCh := make(chan ctxR, 1)

	go func() {
		d, ok := p.deps.FetchOdds(ctx, home.Name, away.Name, m.KickoffUTC)
		oCh <- oddsR{d, ok}
	}()
	go func() {
		d, ok := p.deps.FetchNews(ctx, p.claude, home.Name, away.Name)
		nCh <- newsR{d, ok}
	}()
	go func() {
		d, ok := p.deps.FetchLineup(ctx, p.claude, home.Name, away.Name)
		lCh <- lineupR{d, ok}
	}()
	go func() {
		d, ok := p.deps.FetchContext(p.store, m.HomeTeamCode, m.AwayTeamCode)
		cCh <- ctxR{d, ok}
	}()

	odds := <-oCh
	news := <-nCh
	lineup := <-lCh
	context_ := <-cCh

	conf := Confidence(Inputs{
		LineupOK:        lineup.ok,
		LineupConfirmed: lineup.ok && lineup.data.Confirmed,
		OddsOK:          odds.ok,
		NewsOK:          news.ok,
		ContextOK:       context_.ok,
	})

	inputsRaw, _ := json.Marshal(map[string]any{
		"odds":    odds.data,
		"news":    news.data,
		"lineup":  lineup.data,
		"context": context_.data,
	})

	prompt := claudec.BuildPrompt(claudec.PromptInputs{
		SystemPrompt: p.systemPrompt,
		HomeName:     home.Name,
		AwayName:     away.Name,
		KickoffUTC:   m.KickoffUTC,
		Stage:        m.Stage,
		OddsBlock:    blockify(odds.data),
		NewsBlock:    fmt.Sprintf("Home: %s\nAway: %s", news.data.HomeSummary, news.data.AwaySummary),
		LineupBlock:  fmt.Sprintf("Confirmed: %v\nNotes: %s", lineup.data.Confirmed, lineup.data.Notes),
		ContextBlock: context_.data.TournamentContext + "\n\n" + context_.data.TrackRecord,
	})

	res, err := p.claude.Predict(ctx, prompt)
	if err != nil {
		return store.Prediction{}, fmt.Errorf("claude predict: %w", err)
	}

	pred := store.Prediction{
		MatchID:         matchID,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
		Trigger:         trigger,
		Confidence:      conf,
		PredictedWinner: res.Winner,
		PredictedScore:  res.PredictedScore,
		WinProbability:  res.WinProbability,
		Reasoning:       strings.Join(res.Reasoning, "\n- "),
		InputsJSON:      string(inputsRaw),
		RenderedPrompt:  prompt,
		ModelID:         p.claude.ModelID(),
		PromptVersion:   p.promptVersion,
	}
	id, err := p.store.InsertPrediction(pred)
	if err != nil {
		return store.Prediction{}, fmt.Errorf("insert prediction: %w", err)
	}
	pred.ID = id
	return pred, nil
}

func blockify(v any) string {
	if v == nil {
		return ""
	}
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
