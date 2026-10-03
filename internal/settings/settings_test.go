package settings

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/db"
)

func TestGetSet(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	store := NewStore(database)
	ctx := context.Background()

	if _, ok, err := store.Get(ctx, "missing"); err != nil || ok {
		t.Fatalf("Get(missing) = ok %v, err %v; want false, nil", ok, err)
	}
	if err := store.Set(ctx, "k", "one"); err != nil {
		t.Fatal(err)
	}
	if err := store.Set(ctx, "k", "two"); err != nil {
		t.Fatal(err)
	}
	if v, ok, err := store.Get(ctx, "k"); err != nil || !ok || v != "two" {
		t.Fatalf("Get(k) = %q, %v, %v; want two, true, nil", v, ok, err)
	}
}
