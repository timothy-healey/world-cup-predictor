package store

import (
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
