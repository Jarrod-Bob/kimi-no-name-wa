package history

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/db"
	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/namegen"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	return NewStore(database)
}

func record(t *testing.T, s *Store, description string, names ...string) Generation {
	t.Helper()
	var list []namegen.Name
	for _, n := range names {
		list = append(list, namegen.Name{Name: n, Technique: "pun", Explanation: "because", Tone: "witty"})
	}
	g, err := s.Record(context.Background(), "web", "gemma4:31b", namegen.Request{
		Description: description, Tones: []string{"witty"}, Like: &namegen.Seed{Name: "seed"},
	}, list)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestRecordAndList(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	first := record(t, s, "first", "a", "b")
	second := record(t, s, "second", "c")

	if len(first.Names) != 2 || first.Names[0].ID == 0 || *first.Names[0].GenerationID != first.ID {
		t.Fatalf("recorded = %+v", first)
	}

	gens, err := s.List(ctx, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(gens) != 2 || gens[0].ID != second.ID || gens[1].Description != "first" {
		t.Fatalf("list order wrong: %+v", gens)
	}
	if len(gens[1].Names) != 2 || gens[1].Names[1].Name.Name != "b" || gens[1].LikeName != "seed" || gens[1].Tones[0] != "witty" {
		t.Errorf("first generation round-trip = %+v", gens[1])
	}
	if gens[1].Inspiration == nil || gens[1].Keywords == nil {
		t.Errorf("empty lists must encode as [], not null")
	}

	page, err := s.List(ctx, 10, second.ID)
	if err != nil || len(page) != 1 || page[0].ID != first.ID {
		t.Errorf("paged list = %+v, %v", page, err)
	}
}

func TestFavouritesSurviveDeletion(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	g := record(t, s, "ideas bank", "nuggets", "crumbs")

	starred, err := s.SetFavourite(ctx, g.Names[0].ID, true)
	if err != nil || !starred.Favourite || starred.FavouritedAt == nil || starred.Description != "ideas bank" {
		t.Fatalf("SetFavourite = %+v, %v", starred, err)
	}
	again, _ := s.SetFavourite(ctx, g.Names[0].ID, true)
	if !again.FavouritedAt.Equal(*starred.FavouritedAt) {
		t.Errorf("re-starring moved favourited_at")
	}

	if err := s.Delete(ctx, g.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, g.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete err = %v", err)
	}

	favs, err := s.Favourites(ctx)
	if err != nil || len(favs) != 1 || favs[0].Name.Name != "nuggets" || favs[0].GenerationID != nil || favs[0].Description != "" {
		t.Fatalf("favourites after delete = %+v, %v", favs, err)
	}
	if _, err := s.SetFavourite(ctx, g.Names[1].ID, true); !errors.Is(err, ErrNotFound) {
		t.Errorf("unstarred name should be gone with its generation: %v", err)
	}

	unstarred, err := s.SetFavourite(ctx, favs[0].ID, false)
	if err != nil || unstarred.Favourite {
		t.Fatalf("unstar = %+v, %v", unstarred, err)
	}
	if favs, _ := s.Favourites(ctx); len(favs) != 0 {
		t.Errorf("favourites after unstar = %+v", favs)
	}
}
