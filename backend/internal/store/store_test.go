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
