package ollama

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/settings"
)

// Settings keys this package owns. Nothing else reads or writes them directly.
const (
	KeyURL   = "ollama.url"
	KeyModel = "ollama.model"
)

// DefaultURL is where `ollama serve` listens out of the box.
const DefaultURL = "http://127.0.0.1:11434"

// DefaultModel is Google's Gemma 4 31B (dense): strong at creative wordplay
// and multilingual (Japanese) riffs, about 20 GB at its default quantisation,
// so it fits a 32 GB GPU with room for context. Smaller GPUs can switch to
// gemma4:26b (MoE, about 18 GB) or gemma4:12b (about 8 GB) in Settings.
const DefaultModel = "gemma4:31b"

// Config is where the generator finds its model.
type Config struct {
	URL   string `json:"ollama_url"`
	Model string `json:"model"`
}

var (
	ErrInvalidURL   = errors.New("Ollama URL must be an http:// or https:// address, like " + DefaultURL + ".")
	ErrInvalidModel = errors.New("Model must be a model name like " + DefaultModel + ", with no spaces.")
)

// LoadConfig reads the stored config, falling back to the defaults for
// anything never saved.
func LoadConfig(ctx context.Context, store *settings.Store) (Config, error) {
	cfg := Config{URL: DefaultURL, Model: DefaultModel}
	if v, ok, err := store.Get(ctx, KeyURL); err != nil {
		return cfg, err
	} else if ok {
		cfg.URL = v
	}
	if v, ok, err := store.Get(ctx, KeyModel); err != nil {
		return cfg, err
	} else if ok {
		cfg.Model = v
	}
	return cfg, nil
}

// Normalize trims cfg and checks it, returning ErrInvalidURL or
// ErrInvalidModel.
func (cfg Config) Normalize() (Config, error) {
	cfg.URL = strings.TrimRight(strings.TrimSpace(cfg.URL), "/")
	cfg.Model = strings.TrimSpace(cfg.Model)
	u, err := url.Parse(cfg.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return cfg, ErrInvalidURL
	}
	if cfg.Model == "" || strings.ContainsAny(cfg.Model, " \t\r\n") || len(cfg.Model) > 200 {
		return cfg, ErrInvalidModel
	}
	return cfg, nil
}

// SaveConfig normalizes and stores cfg.
func SaveConfig(ctx context.Context, store *settings.Store, cfg Config) (Config, error) {
	cfg, err := cfg.Normalize()
	if err != nil {
		return cfg, err
	}
	if err := store.Set(ctx, KeyURL, cfg.URL); err != nil {
		return cfg, err
	}
	if err := store.Set(ctx, KeyModel, cfg.Model); err != nil {
		return cfg, err
	}
	return cfg, nil
}
