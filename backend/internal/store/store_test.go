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
