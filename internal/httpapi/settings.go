package httpapi

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/ollama"
)

type healthResponse struct {
	// Status is "ok" when names can be generated, else "degraded"; the
	// server itself is up either way.
	Status string        `json:"status"`
	Ollama ollama.Health `json:"ollama"`
}

// modelLoadError is health's error after a generation failed with
// model_error. It clears once a generation with the same settings works.
const modelLoadError = "The model failed to load or run on the last try, so names can't be generated. Check Ollama's log or the GPU on its host, then generate again."

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	cfg, err := ollama.LoadConfig(r.Context(), s.Settings)
	if err != nil {
		log.Printf("loading ollama config: %v", err)
		writeError(w, http.StatusInternalServerError, "internal", "Couldn't read the settings.")
		return
	}
	h := ollama.Probe(r.Context(), s.ollamaClient(cfg), cfg.Model)
	status := "ok"
	if !h.Ready() {
		status = "degraded"
	} else if s.modelFailure.failedFor(cfg) {
		status = "degraded"
		h.Error = modelLoadError
	}
	writeJSON(w, http.StatusOK, healthResponse{Status: status, Ollama: h})
}

type settingsResponse struct {
	ollama.Config
	Defaults ollama.Config `json:"defaults"`
}

func defaults() ollama.Config {
	return ollama.Config{URL: ollama.DefaultURL, Model: ollama.DefaultModel}
}

func (s *server) getSettings(w http.ResponseWriter, r *http.Request) {
	cfg, err := ollama.LoadConfig(r.Context(), s.Settings)
	if err != nil {
		log.Printf("loading ollama config: %v", err)
		writeError(w, http.StatusInternalServerError, "internal", "Couldn't read the settings.")
		return
	}
	writeJSON(w, http.StatusOK, settingsResponse{Config: cfg, Defaults: defaults()})
}

func (s *server) putSettings(w http.ResponseWriter, r *http.Request) {
	var cfg ollama.Config
	if !decodeJSON(w, r, &cfg) {
		return
	}
	saved, err := ollama.SaveConfig(r.Context(), s.Settings, cfg)
	if errors.Is(err, ollama.ErrInvalidURL) || errors.Is(err, ollama.ErrInvalidModel) {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if err != nil {
		log.Printf("saving ollama config: %v", err)
		writeError(w, http.StatusInternalServerError, "internal", "Couldn't save the settings.")
		return
	}
	writeJSON(w, http.StatusOK, settingsResponse{Config: saved, Defaults: defaults()})
}

func (s *server) check(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" || utf8.RuneCountInString(name) > 100 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Pass the name to check as ?name=, under 100 characters.")
		return
	}
	writeJSON(w, http.StatusOK, s.Checker.Check(r.Context(), name))
}
