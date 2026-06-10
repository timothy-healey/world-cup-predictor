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

	"github.com/timhealey/world-cup-predictor/backend/internal/trace"
)

// claudeBin is the minimal interface the fetchers need from claudec.Driver.
type claudeBin interface {
	BinPathRaw() string
}

func runJSON[T any](ctx context.Context, d claudeBin, ns, prompt string) (T, error) {
	var zero T
	timed, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(timed, d.BinPathRaw(), "-p", prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	trace.SubprocessStart(ns, len(prompt))
	start := time.Now()
	err := cmd.Run()
	dur := time.Since(start)
	if err != nil {
		wrapped := fmt.Errorf("claude invoke: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
		trace.SubprocessError(ns, dur, wrapped)
		return zero, wrapped
	}
	trace.SubprocessEnd(ns, dur, stdout.Len())

	out := stdout.Bytes()
	startIdx := bytes.IndexByte(out, '{')
	end := bytes.LastIndexByte(out, '}')
	if startIdx < 0 || end <= startIdx {
		return zero, errors.New("malformed json")
	}
	if err := json.Unmarshal(out[startIdx:end+1], &zero); err != nil {
		return zero, err
	}
	return zero, nil
}
