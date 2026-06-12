package plist

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"
)

const agentTmpl = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.wcp.{{.MatchID}}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.BinPath}}</string>
        <string>predict</string>
        <string>--match</string>
        <string>{{.MatchID}}</string>
        <string>--email</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>WCP_DB_PATH</key>
        <string>{{.DBAbsPath}}</string>
    </dict>
    <key>StartCalendarInterval</key>
    <dict>
        <key>Year</key>      <integer>{{.Year}}</integer>
        <key>Month</key>     <integer>{{.Month}}</integer>
        <key>Day</key>       <integer>{{.Day}}</integer>
        <key>Hour</key>      <integer>{{.Hour}}</integer>
        <key>Minute</key>    <integer>{{.Minute}}</integer>
    </dict>
    <key>StandardOutPath</key>
    <string>{{.LogPath}}</string>
    <key>StandardErrorPath</key>
    <string>{{.LogPath}}</string>
    <key>WorkingDirectory</key>
    <string>{{.WorkDir}}</string>
    <key>RunAtLoad</key>
    <false/>
</dict>
</plist>
`

type agentData struct {
	MatchID, BinPath, LogPath, WorkDir, DBAbsPath string
	Year, Month, Day, Hour, Minute                int
}

// WriteAgent writes (or overwrites) a launchd LaunchAgent plist that fires
// at kickoff minus 30 minutes (in local time, computed by launchd at runtime).
// workDir is the absolute path of the directory that contains `.env` and
// `wcp.db` (i.e. the backend dir) — launchd uses it as the process cwd and
// the plist pins WCP_DB_PATH to <workDir>/wcp.db so scheduled predictions
// can locate config and database regardless of how launchd resolves cwd.
func WriteAgent(dir, binPath, matchID, workDir string, kickoff time.Time) (string, error) {
	t := kickoff.Add(-30 * time.Minute).Local()
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, "Library", "Logs", "wcp", matchID+".log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return "", err
	}
	dbAbsPath := filepath.Join(workDir, "wcp.db")

	data := agentData{
		MatchID:   matchID,
		BinPath:   binPath,
		LogPath:   logPath,
		WorkDir:   workDir,
		DBAbsPath: dbAbsPath,
		Year:      t.Year(),
		Month:     int(t.Month()),
		Day:       t.Day(),
		Hour:      t.Hour(),
		Minute:    t.Minute(),
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "com.wcp."+matchID+".plist")
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	tmpl := template.Must(template.New("agent").Parse(agentTmpl))
	if err := tmpl.Execute(f, data); err != nil {
		return "", fmt.Errorf("render plist: %w", err)
	}
	return path, nil
}

const resultsAgentTmpl = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.wcp.results</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.BinPath}}</string>
        <string>results</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>WCP_DB_PATH</key>
        <string>{{.DBAbsPath}}</string>
    </dict>
    <key>StartCalendarInterval</key>
    <dict>
        <key>Hour</key>      <integer>{{.Hour}}</integer>
        <key>Minute</key>    <integer>{{.Minute}}</integer>
    </dict>
    <key>StandardOutPath</key>
    <string>{{.LogPath}}</string>
    <key>StandardErrorPath</key>
    <string>{{.LogPath}}</string>
    <key>WorkingDirectory</key>
    <string>{{.WorkDir}}</string>
    <key>RunAtLoad</key>
    <false/>
</dict>
</plist>
`

type resultsAgentData struct {
	BinPath, LogPath, WorkDir, DBAbsPath string
	Hour, Minute                         int
}

// resultsAgentHour is the daily fire time in machine-local time. 21:00 sits
// in the natural gap between the day's last World Cup match (latest finish
// ~17:00 ACST including ET/penalties on the 04:00Z slot) and the next day's
// first kickoff (~01:30 ACST on the 16:00Z slot), while the laptop is
// reliably awake.
const resultsAgentHour = 21

// WriteResultsAgent writes (or overwrites) a daily LaunchAgent that runs
// `wcp results` to pull finished scores from football-data.org and refresh
// predictions.json. Omitting Day/Month/Year from StartCalendarInterval
// makes launchd fire every day at the configured Hour/Minute (local time).
func WriteResultsAgent(dir, binPath, workDir string) (string, error) {
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, "Library", "Logs", "wcp", "results.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return "", err
	}
	data := resultsAgentData{
		BinPath:   binPath,
		LogPath:   logPath,
		WorkDir:   workDir,
		DBAbsPath: filepath.Join(workDir, "wcp.db"),
		Hour:      resultsAgentHour,
		Minute:    0,
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "com.wcp.results.plist")
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	tmpl := template.Must(template.New("results-agent").Parse(resultsAgentTmpl))
	if err := tmpl.Execute(f, data); err != nil {
		return "", fmt.Errorf("render results plist: %w", err)
	}
	return path, nil
}

// LoadAgent runs `launchctl load -w <path>` to activate the agent. No-op
// if not on macOS or if launchctl is not present.
func LoadAgent(path string) error {
	// Implementation: shell out to launchctl. Tested manually since it
	// mutates user-level state.
	return runLaunchctl("load", "-w", path)
}

// UnloadAgent runs `launchctl unload -w <path>`.
func UnloadAgent(path string) error {
	return runLaunchctl("unload", "-w", path)
}
