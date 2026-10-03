-- +goose Up

-- One row per call to the generator, from the web UI or the HTTP API.
-- inspiration, keywords and tones are JSON arrays of strings.
CREATE TABLE generations (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at  TIMESTAMP NOT NULL,
    client      TEXT NOT NULL DEFAULT '',
    model       TEXT NOT NULL,
    description TEXT NOT NULL,
    inspiration TEXT NOT NULL DEFAULT '[]',
    keywords    TEXT NOT NULL DEFAULT '[]',
    tones       TEXT NOT NULL DEFAULT '[]',
    like_name   TEXT NOT NULL DEFAULT ''
);

-- Every name a generation produced. A favourite outlives the generation it
-- came from: deleting a generation removes its other names and detaches
-- favourites (generation_id becomes NULL).
CREATE TABLE names (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    generation_id INTEGER REFERENCES generations(id) ON DELETE SET NULL,
    position      INTEGER NOT NULL,
    name          TEXT NOT NULL,
    technique     TEXT NOT NULL,
    explanation   TEXT NOT NULL,
    tone          TEXT NOT NULL,
    favourited_at TIMESTAMP
);

CREATE INDEX idx_generations_created ON generations(created_at DESC);
CREATE INDEX idx_names_generation ON names(generation_id, position);
CREATE INDEX idx_names_favourited ON names(favourited_at DESC) WHERE favourited_at IS NOT NULL;

-- App-level key/value preferences (internal/settings).
CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- +goose Down
DROP TABLE settings;
DROP TABLE names;
DROP TABLE generations;
