package server

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHealth(t *testing.T) {
	srv := New(Config{Port: 28765, JSONPath: filepath.Join(t.TempDir(), "predictions.json")})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.Start(ctx) }()
	waitForServer(t, "http://127.0.0.1:28765/api/health")

	resp, err := http.Get("http://127.0.0.1:28765/api/health")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"ok":true}` {
		t.Fatalf("body: %s", body)
	}
}

func TestPredictionsJSONServed(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "predictions.json")
	if err := os.WriteFile(jsonPath, []byte(`{"generated_at":"now","teams":[],"matches":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	srv := New(Config{Port: 28766, JSONPath: jsonPath})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.Start(ctx) }()
	waitForServer(t, "http://127.0.0.1:28766/api/health")

	resp, err := http.Get("http://127.0.0.1:28766/predictions.json")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"generated_at":"now","teams":[],"matches":[]}` {
		t.Fatalf("body: %s", body)
	}
}

func TestPredictRequiresMatchParam(t *testing.T) {
	srv := New(Config{
		Port:     28767,
		JSONPath: filepath.Join(t.TempDir(), "predictions.json"),
		Predict:  func(_ context.Context, _ string) error { return nil },
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.Start(ctx) }()
	waitForServer(t, "http://127.0.0.1:28767/api/health")

	resp, err := http.Post("http://127.0.0.1:28767/api/predict", "", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func waitForServer(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server did not come up at %s", url)
}
