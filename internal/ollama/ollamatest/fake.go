// Package ollamatest is an in-process fake Ollama server for tests. Nothing
// in this repo's tests may call a real model or network service; they point
// the app at one of these instead.
package ollamatest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// ChatCall is one POST /api/chat the fake received, decoded.
type ChatCall struct {
	Model    string            `json:"model"`
	Messages []json.RawMessage `json:"messages"`
	Stream   bool              `json:"stream"`
	Format   json.RawMessage   `json:"format"`
	Options  map[string]any    `json:"options"`
	Think    *bool             `json:"think"`
}

// Fake is a fake Ollama. Set its fields before the code under test calls it;
// they are read under a lock, so tests may also change them between calls.
type Fake struct {
	*httptest.Server

	mu sync.Mutex
	// Models are reported by GET /api/tags.
	Models []string
	// Replies are returned, in order, as the assistant content of successive
	// /api/chat calls; the last one repeats once they run out.
	Replies []string
	// ChatStatus, when non-zero, makes /api/chat fail with this status and
	// ChatError as Ollama's {"error": ...} message.
	ChatStatus int
	ChatError  string
	// Calls records every /api/chat request.
	Calls []ChatCall
}

// New starts a fake with the given pulled models, closed when the test ends.
func New(t *testing.T, models ...string) *Fake {
	t.Helper()
	f := &Fake{Models: models}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"version": "0.0.0-fake"})
	})
	mux.HandleFunc("GET /api/tags", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		type model struct {
			Name  string `json:"name"`
			Model string `json:"model"`
		}
		out := struct {
			Models []model `json:"models"`
		}{Models: []model{}}
		for _, m := range f.Models {
			out.Models = append(out.Models, model{Name: m, Model: m})
		}
		writeJSON(w, http.StatusOK, out)
	})
	mux.HandleFunc("POST /api/chat", func(w http.ResponseWriter, r *http.Request) {
		var call ChatCall
		if err := json.NewDecoder(r.Body).Decode(&call); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad json"})
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.Calls = append(f.Calls, call)
		if f.ChatStatus != 0 {
			writeJSON(w, f.ChatStatus, map[string]string{"error": f.ChatError})
			return
		}
		if !hasModel(f.Models, call.Model) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "model '" + call.Model + "' not found"})
			return
		}
		reply := `{"names":[]}`
		if len(f.Replies) > 0 {
			reply = f.Replies[0]
			if len(f.Replies) > 1 {
				f.Replies = f.Replies[1:]
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"model":   call.Model,
			"message": map[string]string{"role": "assistant", "content": reply},
			"done":    true,
		})
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

// SetReplies replaces the queued chat replies.
func (f *Fake) SetReplies(replies ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Replies = replies
}

// ChatCalls returns a copy of the chat requests received so far.
func (f *Fake) ChatCalls() []ChatCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]ChatCall(nil), f.Calls...)
}

func hasModel(models []string, want string) bool {
	for _, m := range models {
		if m == want {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
