// Package history keeps every generation and the names it produced, and which
// names are favourites. A favourite outlives its generation: deleting a
// generation removes its other names but keeps starred ones.
package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/namegen"
)

var ErrNotFound = errors.New("not found")

// SavedName is a generated name as stored.
type SavedName struct {
	ID           int64  `json:"id"`
	GenerationID *int64 `json:"generation_id"`
	namegen.Name
	Favourite    bool       `json:"favourite"`
	FavouritedAt *time.Time `json:"favourited_at"`
	// Description is what the name was generated for; empty once its
	// generation is deleted. Only the favourites list fills it in.
	Description string `json:"description,omitempty"`
}

// Generation is one call to the generator and its results.
type Generation struct {
	ID          int64       `json:"id"`
	CreatedAt   time.Time   `json:"created_at"`
	Client      string      `json:"client"`
	Model       string      `json:"model"`
	Description string      `json:"description"`
	Inspiration []string    `json:"inspiration"`
	Keywords    []string    `json:"keywords"`
	Tones       []string    `json:"tones"`
	LikeName    string      `json:"like_name"`
	Names       []SavedName `json:"names"`
}

// Store reads and writes generations and names.
type Store struct {
	db  *sql.DB
	now func() time.Time
}

func NewStore(database *sql.DB) *Store {
	return &Store{db: database, now: func() time.Time { return time.Now().UTC() }}
}

// Record saves a generation and its names, returning it with IDs filled in.
func (s *Store) Record(ctx context.Context, client, model string, req namegen.Request, names []namegen.Name) (Generation, error) {
	g := Generation{
		CreatedAt:   s.now(),
		Client:      client,
		Model:       model,
		Description: req.Description,
		Inspiration: nonNil(req.Inspiration),
		Keywords:    nonNil(req.Keywords),
		Tones:       nonNil(req.Tones),
		Names:       []SavedName{},
	}
	if req.Like != nil {
		g.LikeName = req.Like.Name
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return g, fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`INSERT INTO generations (created_at, client, model, description, inspiration, keywords, tones, like_name)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		g.CreatedAt, g.Client, g.Model, g.Description,
		encodeList(g.Inspiration), encodeList(g.Keywords), encodeList(g.Tones), g.LikeName,
	)
	if err != nil {
		return g, fmt.Errorf("inserting generation: %w", err)
	}
	if g.ID, err = res.LastInsertId(); err != nil {
		return g, err
	}
	for i, n := range names {
		res, err := tx.ExecContext(ctx,
			`INSERT INTO names (generation_id, position, name, technique, explanation, tone) VALUES (?, ?, ?, ?, ?, ?)`,
			g.ID, i, n.Name, n.Technique, n.Explanation, n.Tone,
		)
		if err != nil {
			return g, fmt.Errorf("inserting name: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return g, err
		}
		genID := g.ID
		g.Names = append(g.Names, SavedName{ID: id, GenerationID: &genID, Name: n})
	}
	return g, tx.Commit()
}

// List returns up to limit generations, newest first, with their names.
// beforeID pages: pass the last ID of the previous page, or 0 for the first.
func (s *Store) List(ctx context.Context, limit int, beforeID int64) ([]Generation, error) {
	query := `SELECT id, created_at, client, model, description, inspiration, keywords, tones, like_name FROM generations`
	args := []any{}
	if beforeID > 0 {
		query += ` WHERE id < ?`
		args = append(args, beforeID)
	}
	query += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing generations: %w", err)
	}
	gens := []Generation{}
	byID := map[int64]int{}
	for rows.Next() {
		var g Generation
		var inspiration, keywords, tones string
		if err := rows.Scan(&g.ID, &g.CreatedAt, &g.Client, &g.Model, &g.Description, &inspiration, &keywords, &tones, &g.LikeName); err != nil {
			rows.Close()
			return nil, err
		}
		g.Inspiration, g.Keywords, g.Tones = decodeList(inspiration), decodeList(keywords), decodeList(tones)
		g.Names = []SavedName{}
		byID[g.ID] = len(gens)
		gens = append(gens, g)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(gens) == 0 {
		return gens, nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(gens)), ",")
	ids := make([]any, len(gens))
	for i, g := range gens {
		ids[i] = g.ID
	}
	names, err := s.queryNames(ctx,
		`SELECT n.id, n.generation_id, n.name, n.technique, n.explanation, n.tone, n.favourited_at, ''
		 FROM names n WHERE n.generation_id IN (`+placeholders+`) ORDER BY n.generation_id, n.position`, ids...)
	if err != nil {
		return nil, err
	}
	for _, n := range names {
		i := byID[*n.GenerationID]
		gens[i].Names = append(gens[i].Names, n)
	}
	return gens, nil
}

// Delete removes a generation and its names, except favourites, which stay
// with no generation.
func (s *Store) Delete(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM names WHERE generation_id = ? AND favourited_at IS NULL`, id); err != nil {
		return fmt.Errorf("deleting names: %w", err)
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM generations WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting generation: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}

// SetFavourite stars or unstars a name and returns it.
func (s *Store) SetFavourite(ctx context.Context, nameID int64, favourite bool) (SavedName, error) {
	var at any
	if favourite {
		at = s.now()
	}
	// Re-starring keeps the original time, so favourites don't reshuffle.
	res, err := s.db.ExecContext(ctx,
		`UPDATE names SET favourited_at = CASE WHEN ? IS NULL THEN NULL ELSE COALESCE(favourited_at, ?) END WHERE id = ?`,
		at, at, nameID)
	if err != nil {
		return SavedName{}, fmt.Errorf("updating favourite: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return SavedName{}, ErrNotFound
	}
	names, err := s.queryNames(ctx, namesWithDescription+` WHERE n.id = ?`, nameID)
	if err != nil {
		return SavedName{}, err
	}
	if len(names) == 0 {
		return SavedName{}, ErrNotFound
	}
	return names[0], nil
}

// Favourites lists starred names, most recently starred first.
func (s *Store) Favourites(ctx context.Context) ([]SavedName, error) {
	return s.queryNames(ctx, namesWithDescription+` WHERE n.favourited_at IS NOT NULL ORDER BY n.favourited_at DESC, n.id DESC`)
}

const namesWithDescription = `SELECT n.id, n.generation_id, n.name, n.technique, n.explanation, n.tone, n.favourited_at, COALESCE(g.description, '')
	FROM names n LEFT JOIN generations g ON g.id = n.generation_id`

func (s *Store) queryNames(ctx context.Context, query string, args ...any) ([]SavedName, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying names: %w", err)
	}
	defer rows.Close()
	out := []SavedName{}
	for rows.Next() {
		var n SavedName
		var genID sql.NullInt64
		var favAt sql.NullTime
		if err := rows.Scan(&n.ID, &genID, &n.Name.Name, &n.Technique, &n.Explanation, &n.Tone, &favAt, &n.Description); err != nil {
			return nil, err
		}
		if genID.Valid {
			n.GenerationID = &genID.Int64
		}
		if favAt.Valid {
			t := favAt.Time
			n.FavouritedAt = &t
			n.Favourite = true
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func encodeList(list []string) string {
	encoded, _ := json.Marshal(nonNil(list))
	return string(encoded)
}

func decodeList(raw string) []string {
	var list []string
	if err := json.Unmarshal([]byte(raw), &list); err != nil || list == nil {
		return []string{}
	}
	return list
}

func nonNil(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}
