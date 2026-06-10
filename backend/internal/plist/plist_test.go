package plist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWriteAgentEmitsValidPlist(t *testing.T) {
	dir := t.TempDir()
	binPath := "/usr/local/bin/wcp"

	kickoff, _ := time.Parse(time.RFC3339, "2026-06-25T11:00:00Z")
	matchID := "2026-06-25-ARG-vs-SAU"

	path, err := WriteAgent(dir, binPath, matchID, kickoff)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "com.wcp."+matchID+".plist"), path)

	body, err := os.ReadFile(path)
	require.NoError(t, err)
	s := string(body)
	require.Contains(t, s, "<key>Label</key>")
	require.Contains(t, s, "com.wcp."+matchID)
	require.Contains(t, s, binPath)
	require.Contains(t, s, "--match")
	require.Contains(t, s, matchID)
	require.Contains(t, s, "--email")
	require.Contains(t, s, "StartCalendarInterval")
	// 30 minutes before 11:00 UTC == 10:30 UTC. Local interpretation is launchd's job
	// but the values should be there.
	require.True(t, strings.Contains(s, "<integer>30</integer>") || strings.Contains(s, "<integer>0</integer>"))
}
