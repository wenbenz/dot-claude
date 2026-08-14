package main

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// dbPath resolves the SQLite state file location: $AUTOMAKE_DB, or
// ~/.claude/automake/state.db by default. The DB is global and cross-repo
// by design (see automake/README.md).
func dbPath() (string, error) {
	if p := viper.GetString("db"); p != "" {
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
