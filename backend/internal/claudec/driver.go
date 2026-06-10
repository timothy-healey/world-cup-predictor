package claudec

import (
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
}

func NewDriver(binPath, modelID string) *Driver {
	return &Driver{binPath: binPath, modelID: modelID}
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
	if !errors.Is(err, errMalformedJSON) {
		return Result{}, err
	}
	// One retry with a corrective preamble.
	res, err = d.invoke(ctx, "Your previous response was not valid JSON. Please reply with ONLY the JSON object described.\n\n"+prompt)
	return res, err
}

var errMalformedJSON = errors.New("malformed json from claude")

func (d *Driver) invoke(ctx context.Context, prompt string) (Result, error) {
	timed, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(timed, d.binPath, "-p", prompt)
	out, err := cmd.Output()
	if err != nil {
		return Result{}, fmt.Errorf("claude invoke: %w", err)
	}
	// Find JSON in the output (claude may emit prefix/suffix text).
	start := strings.Index(string(out), "{")
	end := strings.LastIndex(string(out), "}")
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
