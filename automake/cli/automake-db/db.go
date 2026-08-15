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

// schemaVersion is bumped whenever schema.sql changes. It is stored in the
// DB's user_version so openDB can skip re-applying the DDL on every
// invocation -- see openDB for why that matters.
const schemaVersion = 1

// Pragmas are set in the DSN, not with a separate `PRAGMA` statement, because
// database/sql hands each statement whichever pooled connection is free: a
// pragma exec'd once applies only to the one connection that happened to run
// it. The driver replays DSN pragmas on every connection it opens.
//
//   - busy_timeout: the pipeline runs agents concurrently (coder and
//     test-writer in parallel, plus any other pipeline sharing this global
//     DB), and SQLite's default is to fail a contended write immediately with
//     SQLITE_BUSY rather than wait.
//   - journal_mode(WAL): lets readers proceed during a write.
//   - foreign_keys: off by default in SQLite, so the dependencies -> issues
//     references would otherwise not be enforced.
const dsnPragmas = "_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"

func openDB() (*sql.DB, error) {
	path, err := dbPath()
	if err != nil {
		return nil, err
	}
	return openDBAt(path)
}

func openDBAt(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}
	db, err := sql.Open("sqlite", path+"?"+dsnPragmas)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	if err := ensureSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// ensureSchema applies schema.sql only when the DB predates the current
// schemaVersion. Every command opens the DB, so unconditionally exec'ing the
// DDL would make even `issue get` take a write lock and contend with the
// concurrent agents above; reading user_version first keeps read-only
// commands read-only.
func ensureSchema(db *sql.DB) error {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}
	if version >= schemaVersion {
		return nil
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("applying schema: %w", err)
	}
	// PRAGMA does not accept bound parameters; schemaVersion is an untainted
	// integer constant.
	if _, err := db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, schemaVersion)); err != nil {
		return fmt.Errorf("recording schema version: %w", err)
	}
	return nil
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
