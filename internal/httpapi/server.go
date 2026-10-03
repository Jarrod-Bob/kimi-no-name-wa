// Package httpapi is the JSON API under /api/v1, shared by the web UI and
// other local apps, plus the embedded frontend at /.
package httpapi

import (
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/history"
	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/namecheck"
	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/ollama"
	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/settings"
)

// Deps are what the API needs. OllamaHTTP is the client used for model
// calls; it needs a long timeout because the first request after Ollama
// starts loads the model into VRAM.
type Deps struct {
	Settings   *settings.Store
	History    *history.Store
	Checker    *namecheck.Checker
	OllamaHTTP *http.Client
	Frontend   http.Handler
}

type server struct {
	Deps
}

// NewServer builds the full handler: API routes plus the embedded frontend.
func NewServer(deps Deps) http.Handler {
	if deps.OllamaHTTP == nil {
		deps.OllamaHTTP = &http.Client{Timeout: 5 * time.Minute}
	}
	s := &server{Deps: deps}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", s.health)
	mux.HandleFunc("GET /api/v1/tones", s.tones)
	mux.HandleFunc("POST /api/v1/names", s.generate)
	mux.HandleFunc("GET /api/v1/generations", s.listGenerations)
	mux.HandleFunc("DELETE /api/v1/generations/{id}", s.deleteGeneration)
	mux.HandleFunc("GET /api/v1/favourites", s.favourites)
	mux.HandleFunc("PUT /api/v1/names/{id}/favourite", s.favourite(true))
	mux.HandleFunc("DELETE /api/v1/names/{id}/favourite", s.favourite(false))
	mux.HandleFunc("GET /api/v1/check", s.check)
	mux.HandleFunc("GET /api/v1/settings", s.getSettings)
	mux.HandleFunc("PUT /api/v1/settings", s.putSettings)

	// Anything else under /api/ is a JSON 404, never the SPA's index.html.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "Not found.")
	})
	if deps.Frontend != nil {
		mux.Handle("/", deps.Frontend)
	}
	return recoverer(logger(mux))
}

func (s *server) ollamaClient(cfg ollama.Config) *ollama.Client {
	return ollama.NewClient(cfg.URL, s.OllamaHTTP)
}

// LocalOnly rejects requests whose Host header isn't a loopback name, so a
// web page can't reach the API through DNS rebinding (a hostile domain
// re-pointed at 127.0.0.1). cmd/kimi-no-name-wa applies it whenever the
// server listens on a loopback address.
func LocalOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		host = strings.Trim(strings.ToLower(host), "[]")
		ip := net.ParseIP(host)
		if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			writeError(w, http.StatusForbidden, "forbidden_host", "kimi-no-name-wa only answers requests addressed to localhost or 127.0.0.1.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// recoverer turns a panic into a 500 instead of killing the server mid-session.
func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				log.Printf("panic serving %s %s: %v", r.Method, r.URL.Path, v)
				writeError(w, http.StatusInternalServerError, "internal", "Something went wrong.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
		}
	})
}
