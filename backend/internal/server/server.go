package server

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/timhealey/world-cup-predictor/backend/internal/store"
)

//go:embed all:dist
var distFS embed.FS

// PredictFn runs a prediction for a match and re-exports JSON. The implementation
// lives in cmd/wcp/main.go to keep server.go free of the wide dependency surface
// (claudec, fetchers, predict pipeline, mailer).
type PredictFn func(ctx context.Context, matchID string) error

type Config struct {
	Port     int
	DBPath   string
	JSONPath string
	Predict  PredictFn
}

type Server struct {
	cfg Config
	mu  sync.Mutex // serialise concurrent /api/predict calls
}

func New(cfg Config) *Server {
	return &Server{cfg: cfg}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/predictions.json", s.handleJSON)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/predict", s.handlePredict)
	mux.HandleFunc("/", s.handleStatic)

	addr := fmt.Sprintf("127.0.0.1:%d", s.cfg.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	fmt.Printf("wcp serve listening on http://%s\n", addr)

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func (s *Server) handleJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	f, err := os.Open(s.cfg.JSONPath)
	if err != nil {
		http.Error(w, "predictions.json not found - run wcp predict first", http.StatusNotFound)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.Copy(w, f)
}

func (s *Server) handlePredict(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	matchID := r.URL.Query().Get("match")
	if matchID == "" {
		http.Error(w, "missing ?match=<id>", http.StatusBadRequest)
		return
	}
	if s.cfg.Predict == nil {
		http.Error(w, "predict not configured", http.StatusInternalServerError)
		return
	}
	if !s.mu.TryLock() {
		http.Error(w, "another prediction is in progress", http.StatusConflict)
		return
	}
	defer s.mu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 7*time.Minute)
	defer cancel()
	if err := s.cfg.Predict(ctx, matchID); err != nil {
		http.Error(w, fmt.Sprintf("predict: %v", err), http.StatusInternalServerError)
		return
	}
	// Return the freshly-written prediction by reading the JSON.
	f, err := os.Open(s.cfg.JSONPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("read json: %v", err), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.Copy(w, f)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		http.Error(w, "dist not built", http.StatusInternalServerError)
		return
	}
	// SPA fallback: if the requested path doesn't exist, serve index.html.
	clean := filepath.Clean(r.URL.Path)
	if clean == "/" {
		clean = "/index.html"
	}
	if _, err := fs.Stat(sub, clean[1:]); err != nil {
		clean = "/index.html"
	}
	http.ServeFileFS(w, r, sub, clean[1:])
}

// DistHasIndex reports whether the embedded frontend bundle is present.
// Used by `wcp doctor` (in B16) to flag missing builds.
func DistHasIndex() bool {
	_, err := fs.Stat(distFS, "dist/index.html")
	return err == nil
}

// EnsureJSONExists is a one-time check used at startup. If the JSON file is
// missing, we still start (the static UI can render its error fallback).
func EnsureJSONExists(s *store.Store, path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return s.ExportJSON(path)
}
