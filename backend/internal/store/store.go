package store

import (
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

func nowUTC() string { return time.Now().UTC().Format(time.RFC3339) }

//go:embed schema.sql
var schemaSQL string

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{db: db}, nil
}

// migrate runs idempotent ALTER TABLE migrations for columns added after
// the initial schema. CREATE TABLE IF NOT EXISTS in schema.sql doesn't
// touch existing tables, so any post-v1 column lives here.
func migrate(db *sql.DB) error {
	// predictions.variant — added when ablation experiments were planned.
	// Default 'full' so any pre-migration row reads as a production prediction.
	if _, err := db.Exec(`ALTER TABLE predictions ADD COLUMN variant TEXT NOT NULL DEFAULT 'full'`); err != nil {
		// SQLite returns "duplicate column name" when the column already
		// exists. modernc.org/sqlite surfaces this as a plain error string;
		// there's no error code to match against. The column-already-exists
		// case is the only acceptable failure here.
		if !isDuplicateColumnErr(err) {
			return err
		}
	}
	// predictions.trace_json — per-prediction debug trace (5-entry JSON array).
	// Nullable: predictions written before this column exists read as null and
	// the dashboard hides the trace trigger for them.
	if _, err := db.Exec(`ALTER TABLE predictions ADD COLUMN trace_json TEXT`); err != nil {
		if !isDuplicateColumnErr(err) {
			return err
		}
	}
	return nil
}

func isDuplicateColumnErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate column") || strings.Contains(msg, "already exists")
}

func (s *Store) DB() *sql.DB  { return s.db }
func (s *Store) Close() error { return s.db.Close() }
