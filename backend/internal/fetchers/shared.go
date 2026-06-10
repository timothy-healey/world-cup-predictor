package fetchers

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"time"
)

// claudeBin is the minimal interface the fetchers need from claudec.Driver.
type claudeBin interface {
	BinPathRaw() string
}

func runJSON[T any](ctx context.Context, d claudeBin, prompt string) (T, error) {
	var zero T
	timed, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(timed, d.BinPathRaw(), "-p", prompt)
	out, err := cmd.Output()
	if err != nil {
		return zero, err
	}
	start := strings.Index(string(out), "{")
	end := strings.LastIndex(string(out), "}")
	if start < 0 || end <= start {
		return zero, errors.New("malformed json")
	}
	if err := json.Unmarshal(out[start:end+1], &zero); err != nil {
		return zero, err
	}
	return zero, nil
}
