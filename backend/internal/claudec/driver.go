package claudec

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

type Driver struct {
	binPath string
	modelID string
	timeout time.Duration
}

func NewDriver(binPath, modelID string) *Driver {
	return &Driver{binPath: binPath, modelID: modelID}
}

func (d *Driver) invokeTimeout() time.Duration {
	if d.timeout > 0 {
		return d.timeout
	}
	return 5 * time.Minute
}

type Result struct {
	Winner         string   `json:"winner"`
	PredictedScore string   `json:"predicted_score"`
	WinProbability float64  `json:"win_probability"`
	Reasoning      []string `json:"reasoning"`
}

func (d *Driver) Predict(ctx context.Context, prompt string) (Result, error) {
	res, err := d.invoke(ctx, prompt)
	if err == nil {
		return res, nil
	}
	if !shouldRetry(err) {
		return Result{}, err
	}
	// Check outer ctx — if it's already done, don't bother retrying.
	if ctx.Err() != nil {
		return Result{}, err
	}
	res, err = d.invoke(ctx, "Your previous response was not valid JSON. Please reply with ONLY the JSON object described.\n\n"+prompt)
	return res, err
}

func shouldRetry(err error) bool {
	if errors.Is(err, errMalformedJSON) {
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return true
	}
	return false
}

var errMalformedJSON = errors.New("malformed json from claude")

func (d *Driver) invoke(ctx context.Context, prompt string) (Result, error) {
	timed, cancel := context.WithTimeout(ctx, d.invokeTimeout())
	defer cancel()
	cmd := exec.CommandContext(timed, d.binPath, "-p", prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return Result{}, fmt.Errorf("claude invoke: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}
	out := stdout.Bytes()
	// Find JSON in the output (claude may emit prefix/suffix text).
	start := bytes.IndexByte(out, '{')
	end := bytes.LastIndexByte(out, '}')
	if start < 0 || end <= start {
		return Result{}, errMalformedJSON
	}
	var r Result
	if err := json.Unmarshal(out[start:end+1], &r); err != nil {
		return Result{}, errMalformedJSON
	}
	if r.Winner == "" || r.PredictedScore == "" {
		return Result{}, errMalformedJSON
	}
	return r, nil
}

func (d *Driver) ModelID() string { return d.modelID }
