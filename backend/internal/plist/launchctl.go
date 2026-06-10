package plist

import (
	"fmt"
	"os/exec"
	"runtime"
)

func runLaunchctl(args ...string) error {
	if runtime.GOOS != "darwin" {
		return nil // no-op on non-macOS dev machines
	}
	cmd := exec.Command("launchctl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl %v: %w (%s)", args, err, string(out))
	}
	return nil
}
