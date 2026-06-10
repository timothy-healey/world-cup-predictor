package fetchers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return zero, fmt.Errorf("claude invoke: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}
	out := stdout.Bytes()
	start := bytes.IndexByte(out, '{')
	end := bytes.LastIndexByte(out, '}')
	if start < 0 || end <= start {
		return zero, errors.New("malformed json")
	}
	if err := json.Unmarshal(out[start:end+1], &zero); err != nil {
		return zero, err
	}
	return zero, nil
}
