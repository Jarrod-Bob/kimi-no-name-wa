package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/ollama/ollamatest"
)

func TestChatSendsSchemaAndReturnsContent(t *testing.T) {
	fake := ollamatest.New(t, "gemma4:31b")
	fake.SetReplies(`{"names":[]}`)
	c := NewClient(fake.URL+"/", nil)

	got, err := c.Chat(context.Background(), ChatRequest{
		Model: "gemma4:31b", System: "sys", User: "usr",
		Format:  json.RawMessage(`{"type":"object"}`),
		Options: map[string]any{"temperature": 1.0},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if got != `{"names":[]}` {
		t.Errorf("content = %q", got)
	}
	calls := fake.ChatCalls()
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(calls))
	}
	call := calls[0]
	if call.Stream || string(call.Format) != `{"type":"object"}` || len(call.Messages) != 2 {
		t.Errorf("unexpected request: %+v", call)
	}
	if call.Think == nil || *call.Think {
		t.Errorf("think = %v, want explicit false", call.Think)
	}
}

func TestChatModelMissing(t *testing.T) {
	fake := ollamatest.New(t) // nothing pulled
	_, err := NewClient(fake.URL, nil).Chat(context.Background(), ChatRequest{Model: "gemma4:31b"})
	var missing *ModelMissingError
	if !errors.As(err, &missing) || missing.Model != "gemma4:31b" {
		t.Fatalf("err = %v, want ModelMissingError", err)
	}
	if !strings.Contains(err.Error(), "ollama pull gemma4:31b") {
		t.Errorf("error should say how to pull: %v", err)
	}
}

func TestChatRetriesWithoutThinkWhenRejected(t *testing.T) {
	var sawThink []bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, has := body["think"]
		sawThink = append(sawThink, has)
		if has {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"\"m\" does not support thinking"}`))
			return
		}
		w.Write([]byte(`{"message":{"role":"assistant","content":"ok"}}`))
	}))
	t.Cleanup(srv.Close)

	got, err := NewClient(srv.URL, nil).Chat(context.Background(), ChatRequest{Model: "m"})
	if err != nil || got != "ok" {
		t.Fatalf("Chat = %q, %v", got, err)
	}
	if len(sawThink) != 2 || !sawThink[0] || sawThink[1] {
		t.Errorf("think presence per call = %v, want [true false]", sawThink)
	}
}

func TestUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // nothing listens here any more

	_, err := NewClient(url, nil).Models(context.Background())
	var unreachable *UnreachableError
	if !errors.As(err, &unreachable) {
		t.Fatalf("err = %v, want UnreachableError", err)
	}
}

func TestProbe(t *testing.T) {
	fake := ollamatest.New(t, "gemma4:31b", "llama3.2:latest")
	c := NewClient(fake.URL, nil)

	h := Probe(context.Background(), c, "gemma4:31b")
	if !h.Ready() || h.Version != "0.0.0-fake" || h.Error != "" || len(h.Models) != 2 {
		t.Errorf("ready probe = %+v", h)
	}

	h = Probe(context.Background(), c, "qwen3:32b")
	if h.Ready() || !h.Reachable || !strings.Contains(h.Error, "ollama pull qwen3:32b") {
		t.Errorf("missing-model probe = %+v", h)
	}

	fake.Close()
	h = Probe(context.Background(), c, "gemma4:31b")
	if h.Reachable || h.Ready() || !strings.Contains(h.Error, "isn't reachable") {
		t.Errorf("down probe = %+v", h)
	}
}

func TestHasModel(t *testing.T) {
	pulled := []string{"gemma4:31b", "llama3.2:latest"}
	cases := map[string]bool{
		"gemma4:31b": true, "GEMMA4:31B": true, "llama3.2": true, "llama3.2:latest": true,
		"gemma4": false, "gemma4:26b": false, "": false,
	}
	for model, want := range cases {
		if got := HasModel(pulled, model); got != want {
			t.Errorf("HasModel(%q) = %v, want %v", model, got, want)
		}
	}
}

func TestConfigNormalize(t *testing.T) {
	cfg, err := Config{URL: " http://127.0.0.1:11434/ ", Model: " gemma4:12b "}.Normalize()
	if err != nil || cfg.URL != "http://127.0.0.1:11434" || cfg.Model != "gemma4:12b" {
		t.Fatalf("Normalize = %+v, %v", cfg, err)
	}
	if _, err := (Config{URL: "127.0.0.1:11434", Model: "m"}).Normalize(); !errors.Is(err, ErrInvalidURL) {
		t.Errorf("schemeless URL err = %v", err)
	}
	if _, err := (Config{URL: DefaultURL, Model: "two words"}).Normalize(); !errors.Is(err, ErrInvalidModel) {
		t.Errorf("spaced model err = %v", err)
	}
}
