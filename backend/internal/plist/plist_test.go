package plist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWriteResultsAgentEmitsValidPlist(t *testing.T) {
	dir := t.TempDir()
	binPath := "/usr/local/bin/wcp"
	workDir := "/Users/test/world-cup-predictor/backend"

	path, err := WriteResultsAgent(dir, binPath, workDir)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "com.wcp.results.plist"), path)

	body, err := os.ReadFile(path)
	require.NoError(t, err)
	s := string(body)
	require.Contains(t, s, "<key>Label</key>")
	require.Contains(t, s, "com.wcp.results")
	require.Contains(t, s, binPath)
	require.Contains(t, s, "<string>results</string>")
	require.NotContains(t, s, "--email")
	require.Contains(t, s, "<key>WorkingDirectory</key>")
	require.Contains(t, s, workDir)
	require.Contains(t, s, "<key>EnvironmentVariables</key>")
	require.Contains(t, s, "<key>WCP_DB_PATH</key>")
	require.Contains(t, s, filepath.Join(workDir, "wcp.db"))
	require.Contains(t, s, "StartCalendarInterval")
	// Daily schedule: Hour=21, Minute=0 — no Day/Month/Year keys
	// (omitting them makes launchd fire every day at this time).
	require.Contains(t, s, "<integer>21</integer>")
	require.NotContains(t, s, "<key>Day</key>")
	require.NotContains(t, s, "<key>Month</key>")
	require.NotContains(t, s, "<key>Year</key>")
}

func TestWriteAgentEmitsValidPlist(t *testing.T) {
	dir := t.TempDir()
	binPath := "/usr/local/bin/wcp"
	workDir := "/Users/test/world-cup-predictor/backend"

	kickoff, _ := time.Parse(time.RFC3339, "2026-06-25T11:00:00Z")
	matchID := "2026-06-25-ARG-vs-SAU"

	path, err := WriteAgent(dir, binPath, matchID, workDir, kickoff)
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
	require.Contains(t, s, "<key>WorkingDirectory</key>")
	require.Contains(t, s, workDir)
	require.Contains(t, s, "<key>EnvironmentVariables</key>")
	require.Contains(t, s, "<key>WCP_DB_PATH</key>")
	require.Contains(t, s, filepath.Join(workDir, "wcp.db"))
	// 30 minutes before 11:00 UTC == 10:30 UTC. Local interpretation is launchd's job
	// but the values should be there.
	require.True(t, strings.Contains(s, "<integer>30</integer>") || strings.Contains(s, "<integer>0</integer>"))
}
