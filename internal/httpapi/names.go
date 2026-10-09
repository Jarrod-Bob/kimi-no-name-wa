package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/namegen"
	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/ollama"
)

// generateRequest is POST /api/v1/names: the generator's inputs plus who is
// asking, which history shows ("web" for the UI, e.g. "nuggets" for an app).
type generateRequest struct {
	namegen.Request
	Client string `json:"client"`
}

func (s *server) generate(w http.ResponseWriter, r *http.Request) {
	var body generateRequest
	if !decodeJSON(w, r, &body) {
		return
	}
	req, err := body.Request.Normalize()
	if err != nil {
		var verr *namegen.ValidationError
		if errors.As(err, &verr) {
			writeError(w, http.StatusBadRequest, "invalid_request", verr.Message)
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	client := strings.TrimSpace(body.Client)
	if client == "" {
		client = "api"
	}
	if utf8.RuneCountInString(client) > 40 {
		writeError(w, http.StatusBadRequest, "invalid_request", "client must be under 40 characters.")
		return
	}

	cfg, err := ollama.LoadConfig(r.Context(), s.Settings)
	if err != nil {
		log.Printf("loading ollama config: %v", err)
		writeError(w, http.StatusInternalServerError, "internal", "Couldn't read the settings.")
		return
	}

	names, err := namegen.Generate(r.Context(), s.ollamaClient(cfg), cfg.Model, req)
	if err != nil {
		status, code, message := modelError(cfg, err)
		if code == "model_error" {
			s.modelFailure.record(cfg, true)
		}
		writeError(w, status, code, message)
		return
	}
	s.modelFailure.record(cfg, false)

	gen, err := s.History.Record(r.Context(), client, cfg.Model, req, names)
	if err != nil {
		log.Printf("recording generation: %v", err)
		writeError(w, http.StatusInternalServerError, "internal", "The names were generated but couldn't be saved.")
		return
	}
	writeJSON(w, http.StatusOK, gen)
}

// modelError explains why the model couldn't be used, in a way the UI can
// show as is and an API caller can branch on.
func modelError(cfg ollama.Config, err error) (status int, code, message string) {
	var unreachable *ollama.UnreachableError
	var missing *ollama.ModelMissingError
	var apiErr *ollama.APIError
	switch {
	case errors.As(err, &unreachable):
		return http.StatusServiceUnavailable, "ollama_unreachable",
			"Ollama isn't reachable at " + cfg.URL + ". Install it from https://ollama.com and make sure it's running, or fix the URL in Settings."
	case errors.As(err, &missing):
		return http.StatusServiceUnavailable, "model_missing",
			"The model " + cfg.Model + " isn't pulled yet. Run: ollama pull " + cfg.Model
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout, "model_timeout", "The model took too long to answer. Try again; the first request after starting Ollama loads the model and is the slowest."
	case errors.Is(err, context.Canceled):
		// The caller went away; nobody is listening for an answer.
		return 499, "canceled", "Request canceled."
	case errors.Is(err, namegen.ErrNoNames):
		return http.StatusBadGateway, "no_names", "The model didn't return any usable names. Try again, or try a different model in Settings."
	case errors.As(err, &apiErr):
		return http.StatusBadGateway, "model_error", "Ollama couldn't run the model: " + apiErr.Error()
	default:
		log.Printf("generating: %v", err)
		return http.StatusBadGateway, "model_error", "Talking to the model failed: " + err.Error()
	}
}

func (s *server) tones(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"tones":         namegen.Tones,
		"default_tones": namegen.DefaultTones,
		"techniques":    namegen.Techniques,
		"max_count":     namegen.MaxCount,
		"default_count": namegen.DefaultCount,
	})
}
