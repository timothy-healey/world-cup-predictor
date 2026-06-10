package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/timhealey/world-cup-predictor/backend/internal/bootstrap"
	"github.com/timhealey/world-cup-predictor/backend/internal/config"
	"github.com/timhealey/world-cup-predictor/backend/internal/fdorg"
	"github.com/timhealey/world-cup-predictor/backend/internal/store"
)

type command struct {
	name string
	run  func(ctx context.Context, cfg *config.Config, args []string) error
	help string
}

var commands = []command{
	{name: "bootstrap", run: runBootstrap, help: "Fetch fixtures, write & load launchd plists"},
	{name: "predict", run: stubRun, help: "Run a prediction for a specific match or the next one"},
	{name: "results", run: stubRun, help: "Pull recent finished match results"},
	{name: "serve", run: stubRun, help: "Local HTTP server for the dashboard"},
	{name: "doctor", run: stubRun, help: "Self-audit and config check"},
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
	return bootstrap.Run(ctx, s, c, agentsDir)
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
