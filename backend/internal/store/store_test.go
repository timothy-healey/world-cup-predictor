package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAppliesSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wcp_test.db")
	s, err := Open(dbPath)
	require.NoError(t, err)
	defer s.Close()

	// Assert all three tables exist
	var count int
	row := s.DB().QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('teams','matches','predictions')`,
	)
	require.NoError(t, row.Scan(&count))
	require.Equal(t, 3, count)
}

func TestTeamUpsertAndGet(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()

	team := Team{
		Code: "ARG", Name: "Argentina", GroupID: "F",
		FlagURL: "https://example/arg.png", FIFARanking: 1,
		ManagerName:  "Lionel Scaloni",
		FixtureSrcID: "100",
	}
	require.NoError(t, s.UpsertTeam(team))

	got, err := s.GetTeam("ARG")
	require.NoError(t, err)
	require.Equal(t, "Argentina", got.Name)
	require.Equal(t, 1, got.FIFARanking)

	// Upsert again with a changed name; verify update
	team.Name = "Argentina (updated)"
	require.NoError(t, s.UpsertTeam(team))
	got, _ = s.GetTeam("ARG")
	require.Equal(t, "Argentina (updated)", got.Name)
}

func TestGetTeamByFixtureSrcID(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()

	// Insert a team where the upstream's /teams TLA (CUW) differs from the
	// /matches TLA (CUR) — the realistic football-data.org case for Curaçao.
	// We store the /teams TLA as the canonical code and the numeric ID
	// (here 1538) in fixture_src_id.
	require.NoError(t, s.UpsertTeam(Team{
		Code: "CUW", Name: "Curaçao", FixtureSrcID: "1538",
	}))

	got, err := s.GetTeamByFixtureSrcID("1538")
	require.NoError(t, err)
	require.Equal(t, "CUW", got.Code)
	require.Equal(t, "Curaçao", got.Name)

	// Unknown ID -> sql.ErrNoRows.
	_, err = s.GetTeamByFixtureSrcID("9999")
	require.Error(t, err)
}

func TestMatchUpsertAndList(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()

	require.NoError(t, s.UpsertTeam(Team{Code: "ARG", Name: "Argentina"}))
	require.NoError(t, s.UpsertTeam(Team{Code: "SAU", Name: "Saudi Arabia"}))

	m := Match{
		ID: "2026-06-25-ARG-vs-SAU", HomeTeamCode: "ARG", AwayTeamCode: "SAU",
		KickoffUTC: "2026-06-25T11:00:00Z", Stage: "group", Venue: "MetLife",
		FixtureSrcID: "555",
	}
	require.NoError(t, s.UpsertMatch(m))

	got, err := s.GetMatch(m.ID)
	require.NoError(t, err)
	require.Equal(t, "ARG", got.HomeTeamCode)

	matches, err := s.ListMatches()
	require.NoError(t, err)
	require.Len(t, matches, 1)
}

func TestSetMatchResult(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()
	require.NoError(t, s.UpsertTeam(Team{Code: "ARG", Name: "Argentina"}))
	require.NoError(t, s.UpsertTeam(Team{Code: "SAU", Name: "Saudi Arabia"}))
	require.NoError(t, s.UpsertMatch(Match{
		ID: "m1", HomeTeamCode: "ARG", AwayTeamCode: "SAU",
		KickoffUTC: "2026-06-25T11:00:00Z", Stage: "group",
	}))

	require.NoError(t, s.SetMatchResult("m1", 2, 0, "2026-06-25T13:00:00Z"))

	got, _ := s.GetMatch("m1")
	require.NotNil(t, got.HomeScore)
	require.Equal(t, 2, *got.HomeScore)
}

func TestPredictionInsertAndListByMatch(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()
	require.NoError(t, s.UpsertTeam(Team{Code: "ARG", Name: "Argentina"}))
	require.NoError(t, s.UpsertTeam(Team{Code: "SAU", Name: "Saudi Arabia"}))
	require.NoError(t, s.UpsertMatch(Match{
		ID: "m1", HomeTeamCode: "ARG", AwayTeamCode: "SAU",
		KickoffUTC: "2026-06-25T11:00:00Z", Stage: "group",
	}))

	p := Prediction{
		MatchID: "m1", CreatedAt: "2026-06-25T10:30:00Z",
		Trigger: "scheduled", Confidence: "medium",
		PredictedWinner: "ARG", PredictedScore: "2-0",
		WinProbability: 0.71, Reasoning: "Argentina dominant",
		InputsJSON: `{}`, RenderedPrompt: "...",
		ModelID: "claude-opus-4-7", PromptVersion: "abc123",
	}
	id, err := s.InsertPrediction(p)
	require.NoError(t, err)
	require.Greater(t, id, int64(0))

	list, err := s.ListPredictionsByMatch("m1")
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "ARG", list[0].PredictedWinner)
}

func TestPredictionVariantDefaultsToFull(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "wcp.db"))
	defer s.Close()
	require.NoError(t, s.UpsertTeam(Team{Code: "ARG", Name: "Argentina"}))
	require.NoError(t, s.UpsertTeam(Team{Code: "SAU", Name: "Saudi Arabia"}))
	require.NoError(t, s.UpsertMatch(Match{
		ID: "m1", HomeTeamCode: "ARG", AwayTeamCode: "SAU",
		KickoffUTC: "2026-06-25T11:00:00Z", Stage: "group",
	}))

	// Insert without setting Variant — should land as "full" on read.
	_, err := s.InsertPrediction(Prediction{
		MatchID: "m1", CreatedAt: "2026-06-25T10:30:00Z",
		Trigger: "scheduled", Confidence: "medium",
		PredictedWinner: "ARG", PredictedScore: "2-0",
		WinProbability: 0.71, Reasoning: "x", InputsJSON: "{}",
		RenderedPrompt: "", ModelID: "test", PromptVersion: "v",
	})
	require.NoError(t, err)

	// Insert a second prediction with an explicit non-default variant.
	_, err = s.InsertPrediction(Prediction{
		MatchID: "m1", CreatedAt: "2026-06-25T10:35:00Z",
		Trigger: "on_demand", Confidence: "medium",
		PredictedWinner: "SAU", PredictedScore: "1-0",
		WinProbability: 0.5, Reasoning: "ablation", InputsJSON: "{}",
		RenderedPrompt: "", ModelID: "test", PromptVersion: "v",
		Variant: "no-odds",
	})
	require.NoError(t, err)

	list, err := s.ListPredictionsByMatch("m1")
	require.NoError(t, err)
	require.Len(t, list, 2)
	// List is ordered by created_at DESC, so the no-odds prediction is first.
	require.Equal(t, "no-odds", list[0].Variant)
	require.Equal(t, "full", list[1].Variant)
}

func TestOpenIsIdempotentWithMigrations(t *testing.T) {
	// Open the same DB file twice in sequence to confirm the ALTER TABLE
	// migration handles the "column already exists" case without erroring.
	dbPath := filepath.Join(t.TempDir(), "wcp.db")
	s1, err := Open(dbPath)
	require.NoError(t, err)
	require.NoError(t, s1.Close())

	s2, err := Open(dbPath)
	require.NoError(t, err)
	require.NoError(t, s2.Close())
}

func TestExportJSON(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(filepath.Join(dir, "wcp.db"))
	defer s.Close()
	require.NoError(t, s.UpsertTeam(Team{Code: "ARG", Name: "Argentina"}))
	require.NoError(t, s.UpsertTeam(Team{Code: "SAU", Name: "Saudi Arabia"}))
	require.NoError(t, s.UpsertMatch(Match{
		ID: "m1", HomeTeamCode: "ARG", AwayTeamCode: "SAU",
		KickoffUTC: "2026-06-25T11:00:00Z", Stage: "group",
	}))
	_, _ = s.InsertPrediction(Prediction{
		MatchID: "m1", CreatedAt: "2026-06-25T10:30:00Z",
		Trigger: "scheduled", Confidence: "medium",
		PredictedWinner: "ARG", PredictedScore: "2-0",
		WinProbability: 0.71, Reasoning: "x", InputsJSON: "{}",
		RenderedPrompt: "", ModelID: "test", PromptVersion: "v",
	})

	outPath := filepath.Join(dir, "predictions.json")
	require.NoError(t, s.ExportJSON(outPath))

	body, err := os.ReadFile(outPath)
	require.NoError(t, err)
	require.Contains(t, string(body), `"matches"`)
	require.Contains(t, string(body), `"ARG"`)
}
