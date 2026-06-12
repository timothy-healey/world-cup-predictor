package predict

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/timhealey/world-cup-predictor/backend/internal/claudec"
	"github.com/timhealey/world-cup-predictor/backend/internal/fetchers"
	"github.com/timhealey/world-cup-predictor/backend/internal/store"
	"github.com/timhealey/world-cup-predictor/backend/internal/trace"
)

// systemPromptBytes embeds the predict system prompt at build time so the
// binary is self-contained and works regardless of the cwd it runs from.
//
//go:embed predict.md
var systemPromptBytes []byte

// Deps holds injectable fetcher functions for testing. Each fetcher returns:
//   - data: the fetcher result (typed per kind)
//   - err: non-nil iff the fetcher failed; the pipeline treats nil as "ok"
//   - snippet: a human-readable preview for the trace (caller is free to
//     return "" on failure; truncation to 400 bytes happens in trace.Recorder)
type Deps struct {
	FetchOdds    func(ctx context.Context, homeName, awayName, kickoff string) (any, error, string)
	FetchNews    func(ctx context.Context, d any, home, away string) (fetchers.NewsResult, error, string)
	FetchLineup  func(ctx context.Context, d any, home, away string) (fetchers.LineupResult, error, string)
	FetchContext func(s *store.Store, homeCode, awayCode string) (fetchers.ContextResult, error, string)
}

type Pipeline struct {
	store         *store.Store
	claude        *claudec.Driver
	deps          Deps
	systemPrompt  string
	promptVersion string
	nowFn         func() time.Time
}

func New(s *store.Store, d *claudec.Driver, deps Deps) *Pipeline {
	sysPrompt, version := loadSystemPrompt()
	return &Pipeline{store: s, claude: d, deps: deps, systemPrompt: sysPrompt, promptVersion: version, nowFn: time.Now}
}

func loadSystemPrompt() (string, string) {
	return string(systemPromptBytes), fmt.Sprintf("len-%d", len(systemPromptBytes))
}

func (p *Pipeline) Run(ctx context.Context, matchID, trigger string) (store.Prediction, error) {
	m, err := p.store.GetMatch(matchID)
	if err != nil {
		return store.Prediction{}, fmt.Errorf("get match: %w", err)
	}
	home, _ := p.store.GetTeam(m.HomeTeamCode)
	away, _ := p.store.GetTeam(m.AwayTeamCode)

	rec := trace.New()

	type oddsR struct {
		data    any
		err     error
		snippet string
	}
	type newsR struct {
		data    fetchers.NewsResult
		err     error
		snippet string
	}
	type lineupR struct {
		data    fetchers.LineupResult
		err     error
		snippet string
	}
	type ctxR struct {
		data    fetchers.ContextResult
		err     error
		snippet string
	}

	oCh := make(chan oddsR, 1)
	nCh := make(chan newsR, 1)
	lCh := make(chan lineupR, 1)
	cCh := make(chan ctxR, 1)

	// Start the four trace timers up front so each kind has a started_at
	// regardless of which goroutine finishes first.
	rec.Start("odds")
	rec.Start("news")
	rec.Start("lineup")
	rec.Start("context")

	go func() {
		d, e, s := p.deps.FetchOdds(ctx, home.Name, away.Name, m.KickoffUTC)
		oCh <- oddsR{d, e, s}
	}()
	go func() {
		d, e, s := p.deps.FetchNews(ctx, p.claude, home.Name, away.Name)
		nCh <- newsR{d, e, s}
	}()
	go func() {
		d, e, s := p.deps.FetchLineup(ctx, p.claude, home.Name, away.Name)
		lCh <- lineupR{d, e, s}
	}()
	go func() {
		d, e, s := p.deps.FetchContext(p.store, m.HomeTeamCode, m.AwayTeamCode)
		cCh <- ctxR{d, e, s}
	}()

	odds := <-oCh
	news := <-nCh
	lineup := <-lCh
	context_ := <-cCh

	rec.Finish("odds", odds.err, odds.snippet)
	rec.Finish("news", news.err, news.snippet)
	rec.Finish("lineup", lineup.err, lineup.snippet)
	rec.Finish("context", context_.err, context_.snippet)

	conf := Confidence(Inputs{
		LineupOK:        lineup.err == nil,
		LineupConfirmed: lineup.err == nil && lineup.data.Confirmed,
		OddsOK:          odds.err == nil,
		NewsOK:          news.err == nil,
		ContextOK:       context_.err == nil,
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
		OddsBlock: func() string {
			if odds.err != nil {
				return ""
			}
			return blockify(odds.data)
		}(),
		NewsBlock: func() string {
			if news.err != nil {
				return ""
			}
			return fmt.Sprintf("Home: %s\nAway: %s", news.data.HomeSummary, news.data.AwaySummary)
		}(),
		LineupBlock: func() string {
			if lineup.err != nil {
				return ""
			}
			return fmt.Sprintf("Confirmed: %v\nNotes: %s", lineup.data.Confirmed, lineup.data.Notes)
		}(),
		ContextBlock: func() string {
			if context_.err != nil {
				return ""
			}
			return strings.TrimSpace(context_.data.TournamentContext + "\n\n" + context_.data.TrackRecord)
		}(),
	})

	rec.Start("predict")
	res, err := p.claude.Predict(ctx, prompt)
	rec.Finish("predict", err, predictSnippet(res, err))
	if err != nil {
		return store.Prediction{}, fmt.Errorf("claude predict: %w", err)
	}

	traceBytes, _ := rec.JSON()

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
		Variant:         "full",
		TraceJSON:       string(traceBytes),
	}
	id, err := p.store.InsertPrediction(pred)
	if err != nil {
		return store.Prediction{}, fmt.Errorf("insert prediction: %w", err)
	}
	pred.ID = id
	return pred, nil
}

// predictSnippet derives a short preview from the claude predict result.
// On error the result is the zero value, so we return an empty snippet — the
// error string already carries the diagnostic context.
func predictSnippet(res claudec.Result, err error) string {
	if err != nil {
		return ""
	}
	b, jerr := json.Marshal(map[string]any{
		"winner":          res.Winner,
		"predicted_score": res.PredictedScore,
		"win_probability": res.WinProbability,
	})
	if jerr != nil {
		return ""
	}
	return string(b)
}

func blockify(v any) string {
	if v == nil {
		return ""
	}
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
