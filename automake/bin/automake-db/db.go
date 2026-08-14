package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS issues (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  status      TEXT NOT NULL,
  description TEXT NOT NULL,
  ticket      TEXT,
  created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE IF NOT EXISTS dependencies (
  id         INTEGER NOT NULL REFERENCES issues(id),
  dependency INTEGER NOT NULL REFERENCES issues(id),
  PRIMARY KEY (id, dependency),
  CHECK (id != dependency)
);

CREATE TABLE IF NOT EXISTS work (
  run      INTEGER PRIMARY KEY AUTOINCREMENT,
  id       INTEGER NOT NULL REFERENCES issues(id),
  agent    TEXT NOT NULL,
  context  TEXT,
  started  TEXT NOT NULL,
  finished TEXT,
  output   TEXT,
  repo     TEXT NOT NULL,
  branch   TEXT,
  worktree TEXT,
  pr       TEXT
);
`

// dbPath resolves the SQLite state file location: $AUTOMAKE_DB, or
// ~/.claude/automake/state.db by default. The DB is global and cross-repo
// by design (see automake/README.md).
func dbPath() (string, error) {
	if p := os.Getenv("AUTOMAKE_DB"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".claude", "automake", "state.db"), nil
}

func openDB() (*sql.DB, error) {
	path, err := dbPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("applying schema: %w", err)
	}
	return db, nil
}

func nowRFC3339() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
}

// nullIfEmpty lets optional flags round-trip through SQLite as NULL rather
// than an empty string, so downstream JSON output can omit them cleanly.
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
