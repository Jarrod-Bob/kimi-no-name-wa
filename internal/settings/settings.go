// Package settings stores small app-level key/value preferences in the
// settings table. It is deliberately dumb: no validation, no defaults beyond
// "missing", so every caller decides for itself what an absent key means.
// Each integration owns its keys (internal/ollama/config.go).
package settings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Store reads and writes rows of the settings table.
type Store struct {
	db *sql.DB
}

func NewStore(database *sql.DB) *Store {
	return &Store{db: database}
}

// Get returns the value for key and true, or "" and false if the row is
// missing.
func (s *Store) Get(ctx context.Context, key string) (string, bool, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("reading setting %q: %w", key, err)
	}
	return value, true, nil
}

// Set writes key, replacing any existing value.
func (s *Store) Set(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	if err != nil {
		return fmt.Errorf("writing setting %q: %w", key, err)
	}
	return nil
}
