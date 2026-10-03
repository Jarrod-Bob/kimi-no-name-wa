// Package db opens the kimi-no-name-wa SQLite database and keeps its schema
// current.
package db

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	// Registers the "sqlite" driver. Pure Go — no cgo, no gcc on Windows.
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// DefaultPath returns the database path under the OS's per-user config
// directory (os.UserConfigDir): %AppData%\kimi-no-name-wa\kimi.db on Windows,
// ~/.config/kimi-no-name-wa/kimi.db on Linux. It creates the directory if
// needed. Never a relative path: the database would otherwise move depending
// on which directory the shortcut launched from.
func DefaultPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locating user config dir: %w", err)
	}
	dir := filepath.Join(base, "kimi-no-name-wa")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating %s: %w", dir, err)
	}
	return filepath.Join(dir, "kimi.db"), nil
}

// dsn builds the connection string. ToSlash matters on Windows: backslashes in
// a file: URI are misparsed as escape sequences.
func dsn(path string) string {
	return "file:" + filepath.ToSlash(path) +
		"?_pragma=foreign_keys(1)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=busy_timeout(5000)"
}

// Open connects to the database and migrates it to the latest schema.
func Open(path string) (*sql.DB, error) {
	database, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}

	// Single-user app: one connection removes SQLITE_BUSY entirely.
	database.SetMaxOpenConns(1)

	if err := database.Ping(); err != nil {
		database.Close()
		return nil, fmt.Errorf("connecting to %s: %w", path, err)
	}

	goose.SetBaseFS(migrationFS)
	goose.SetLogger(goose.NopLogger())
	// Goose's dialect name is "sqlite3" even though the driver is "sqlite".
	if err := goose.SetDialect("sqlite3"); err != nil {
		database.Close()
		return nil, fmt.Errorf("setting goose dialect: %w", err)
	}
	if err := goose.Up(database, "migrations"); err != nil {
		database.Close()
		return nil, fmt.Errorf("migrating: %w", err)
	}

	return database, nil
}
