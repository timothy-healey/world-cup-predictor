package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/timhealey/world-cup-predictor/backend/internal/bootstrap"
	"github.com/timhealey/world-cup-predictor/backend/internal/claudec"
	"github.com/timhealey/world-cup-predictor/backend/internal/config"
	"github.com/timhealey/world-cup-predictor/backend/internal/doctor"
	"github.com/timhealey/world-cup-predictor/backend/internal/fdorg"
	"github.com/timhealey/world-cup-predictor/backend/internal/fetchers"
	"github.com/timhealey/world-cup-predictor/backend/internal/mailer"
	"github.com/timhealey/world-cup-predictor/backend/internal/odds"
	"github.com/timhealey/world-cup-predictor/backend/internal/predict"
	"github.com/timhealey/world-cup-predictor/backend/internal/store"
)

type command struct {
	name string
	run  func(ctx context.Context, cfg *config.Config, args []string) error
	help string
}

var commands = []command{
	{name: "bootstrap", run: runBootstrap, help: "Fetch fixtures, write & load launchd plists"},
	{name: "predict", run: runPredict, help: "Predict a specific match or the next upcoming one"},
	{name: "results", run: runResults, help: "Pull recent finished match results"},
	{name: "serve", run: stubRun, help: "Local HTTP server for the dashboard"},
	{name: "doctor", run: runDoctor, help: "Self-audit and config check"},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "wcp: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	repoRoot, _ := os.Getwd()
	cfg, err := config.Load(repoRoot)
	if err != nil {
		return err
	}
	cfg.PrintWarnings()

	if len(os.Args) < 2 {
		printUsage()
		return errors.New("no command given")
	}
	name := os.Args[1]
	if name == "-h" || name == "--help" || name == "help" {
		printUsage()
		return nil
	}
	for _, c := range commands {
		if c.name == name {
			return c.run(context.Background(), cfg, os.Args[2:])
		}
	}
	printUsage()
	return fmt.Errorf("unknown command %q", name)
}

func stubRun(ctx context.Context, cfg *config.Config, args []string) error {
	return errors.New("not implemented yet")
}

func runBootstrap(ctx context.Context, cfg *config.Config, args []string) error {
	s, err := store.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer s.Close()
	c := fdorg.NewClient("https://api.football-data.org", cfg.FootballDataAPIKey)
	home, _ := os.UserHomeDir()
	agentsDir := filepath.Join(home, "Library", "LaunchAgents")
	// Bootstrap is run from the user's shell cwd (the backend directory). Capture
	// it now and bake it into each per-match plist so launchd-fired predictions
	// can find .env and wcp.db regardless of launchd's cwd resolution.
	workDir, _ := os.Getwd()
	return bootstrap.Run(ctx, s, c, agentsDir, workDir)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: wcp <command> [flags]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "commands:")
	for _, c := range commands {
		fmt.Fprintf(os.Stderr, "  %-12s  %s\n", c.name, c.help)
	}
	_ = flag.CommandLine // silence unused if flag pkg ever needed
}

func runPredict(ctx context.Context, cfg *config.Config, args []string) error {
	fs := flag.NewFlagSet("predict", flag.ExitOnError)
	var matchID, next string
	var email bool
	var dryRun bool
	fs.StringVar(&matchID, "match", "", "match ID to predict (e.g. 2026-06-25-ARG-vs-SAU)")
	fs.StringVar(&next, "next", "", "next: predict the next upcoming unpredicted match (no value, just present)")
	fs.BoolVar(&email, "email", false, "send email after prediction")
	fs.BoolVar(&dryRun, "dry-run", false, "print prompt; do not call Claude; do not write to DB")
	if err := fs.Parse(args); err != nil {
		return err
	}

	s, err := store.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer s.Close()

	if matchID == "" {
		mid, err := pickNextMatch(s)
		if err != nil {
			return err
		}
		matchID = mid
	}

	driver := claudec.NewDriver(cfg.ClaudeBin, "claude-opus-4-7")

	var oddsClient *odds.Client
	if cfg.OddsEnabled() {
		oddsClient = odds.NewClient("https://api.the-odds-api.com", cfg.OddsAPIKey)
	}

	deps := predict.Deps{
		FetchOdds: func(ctx context.Context, h, a, k string) (any, bool) {
			if oddsClient == nil {
				return nil, false
			}
			o, err := oddsClient.GetForMatch(ctx, h, a, k)
			if err != nil {
				return nil, false
			}
			return o, true
		},
		FetchNews: func(ctx context.Context, d any, h, a string) (fetchers.NewsResult, bool) {
			r, err := fetchers.FetchNews(ctx, driver, h, a)
			return r, err == nil
		},
		FetchLineup: func(ctx context.Context, d any, h, a string) (fetchers.LineupResult, bool) {
			r, err := fetchers.FetchLineup(ctx, driver, h, a)
			return r, err == nil
		},
		FetchContext: func(s *store.Store, h, a string) (fetchers.ContextResult, bool) {
			r, err := fetchers.FetchContext(s, h, a)
			return r, err == nil
		},
	}

	pipeline := predict.New(s, driver, deps)
	trigger := "on_demand"
	if email {
		trigger = "scheduled"
	}

	rec, err := pipeline.Run(ctx, matchID, trigger)
	if err != nil {
		return err
	}
	fmt.Printf("predicted %s — %s (%s confidence)\n", rec.PredictedWinner, rec.PredictedScore, rec.Confidence)

	// Export JSON next to the DB for the dashboard.
	exportPath := filepath.Join(filepath.Dir(cfg.DBPath), "predictions.json")
	if err := s.ExportJSON(exportPath); err != nil {
		fmt.Fprintf(os.Stderr, "[warn] export json: %v\n", err)
	}

	if email {
		if !cfg.EmailEnabled() {
			fmt.Fprintln(os.Stderr, "[warn] email requested but SMTP not configured")
		} else {
			m, _ := s.GetMatch(matchID)
			subject, body := mailer.RenderEmail(m, rec)
			mail := mailer.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.NotificationTo)
			if err := mail.Send(subject, body); err != nil {
				fmt.Fprintf(os.Stderr, "[warn] email send: %v\n", err)
			}
		}
	}
	return nil
}

func runDoctor(ctx context.Context, cfg *config.Config, args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	dryRunNext := fs.Bool("dry-run-next", false, "run a full prediction on the next match without sending email")
	if err := fs.Parse(args); err != nil {
		return err
	}

	s, err := store.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer s.Close()
	home, _ := os.UserHomeDir()
	agentsDir := filepath.Join(home, "Library", "LaunchAgents")
	fmt.Print(doctor.Run(cfg, s, agentsDir))

	if *dryRunNext {
		fmt.Println("\n--- dry-run-next ---")
		// Reuse the predict path; force --no-email by not setting `email`.
		return runPredict(ctx, cfg, []string{})
	}
	return nil
}

func runResults(ctx context.Context, cfg *config.Config, args []string) error {
	s, err := store.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer s.Close()

	c := fdorg.NewClient("https://api.football-data.org", cfg.FootballDataAPIKey)
	finished, err := c.GetFinishedResults(ctx)
	if err != nil {
		return err
	}
	updated := 0
	for _, m := range finished {
		if m.HomeScore == nil || m.AwayScore == nil {
			continue
		}
		t, err := time.Parse(time.RFC3339, m.UTCDate)
		if err != nil {
			continue
		}
		id := fmt.Sprintf("%s-%s-vs-%s", t.UTC().Format("2006-01-02"), m.HomeTLA, m.AwayTLA)
		if err := s.SetMatchResult(id, *m.HomeScore, *m.AwayScore, time.Now().UTC().Format(time.RFC3339)); err != nil {
			fmt.Fprintf(os.Stderr, "[warn] update %s: %v\n", id, err)
			continue
		}
		updated++
	}
	fmt.Printf("updated %d results\n", updated)

	exportPath := filepath.Join(filepath.Dir(cfg.DBPath), "predictions.json")
	_ = s.ExportJSON(exportPath)
	return nil
}

func pickNextMatch(s *store.Store) (string, error) {
	matches, err := s.ListMatches()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, m := range matches {
		if m.KickoffUTC > now {
			preds, _ := s.ListPredictionsByMatch(m.ID)
			if len(preds) == 0 {
				return m.ID, nil
			}
		}
	}
	return "", errors.New("no upcoming unpredicted matches found")
}
