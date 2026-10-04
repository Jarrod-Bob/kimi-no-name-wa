package db

import (
	"path/filepath"
	"testing"
)

func TestOpenMigratesAndReopens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kimi.db")

	database, err := Open(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	for _, table := range []string{"generations", "names", "settings"} {
		var name string
		err := database.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
		if err != nil {
			t.Errorf("table %s missing: %v", table, err)
		}
	}
	database.Close()

	// A second open must find the schema current and not fail re-applying it.
	database, err = Open(path)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	database.Close()
}
