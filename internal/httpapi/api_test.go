package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/db"
	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/history"
	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/namecheck"
	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/ollama"
	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/ollama/ollamatest"
	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/settings"
)

type testEnv struct {
	srv      http.Handler
	settings *settings.Store
	fake     *ollamatest.Fake
}

// newEnv wires the real handler to a fake Ollama with the default model
// pulled, a fake GitHub/npm, and a stub frontend. Nothing reaches a real
// model or network service.
func newEnv(t *testing.T) *testEnv {
	t.Helper()
	return newEnvWithClient(t, nil)
}

// newEnvWithClient is newEnv with the HTTP client used to reach Ollama
// replaced, e.g. to give it a short timeout; nil keeps the fake's client.
func newEnvWithClient(t *testing.T, ollamaHTTP *http.Client) *testEnv {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	store := settings.NewStore(database)

	fake := ollamatest.New(t, ollama.DefaultModel)
	if _, err := ollama.SaveConfig(t.Context(), store, ollama.Config{URL: fake.URL, Model: ollama.DefaultModel}); err != nil {
		t.Fatal(err)
	}

	services := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/search/") {
			w.Write([]byte(`{"total_count":0,"items":[]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(services.Close)
	checker := namecheck.New()
	checker.GitHubAPI, checker.NPMAPI = services.URL, services.URL

	if ollamaHTTP == nil {
		ollamaHTTP = fake.Client()
	}
	stub := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html>stub spa</html>"))
	})
	return &testEnv{
		srv: NewServer(Deps{
			Settings: store, History: history.NewStore(database), Checker: checker,
			OllamaHTTP: ollamaHTTP, Frontend: stub,
		}),
		settings: store,
		fake:     fake,
	}
}

func (e *testEnv) do(t *testing.T, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	e.srv.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decoding %s: %v", rec.Body, err)
	}
	return v
}

type apiError struct {
	Error struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

func reply(names ...string) string {
	var entries []string
	for _, n := range names {
		entries = append(entries, fmt.Sprintf(`{"name":%q,"technique":"portmanteau","explanation":"because","tone":"witty"}`, n))
	}
	return `{"names":[` + strings.Join(entries, ",") + `]}`
}

func TestGenerateRecordsAndFavourites(t *testing.T) {
	e := newEnv(t)
	e.fake.SetReplies(reply("acceleread", "Acceleread", "skimurai", "kimi-no-name-wa"))

	rec := e.do(t, "POST", "/api/v1/names", map[string]any{
		"description": "a tool to ingest documents quicker",
		"inspiration": []string{"maplestory"},
		"tones":       []string{"witty"},
		// Two usable names out of four is enough not to ask again.
		"count":  4,
		"avoid":  []string{"Kimi no Name wa"},
		"client": "nuggets",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	gen := decode[history.Generation](t, rec)
	if gen.ID == 0 || gen.Client != "nuggets" || gen.Model != ollama.DefaultModel {
		t.Errorf("generation = %+v", gen)
	}
	var got []string
	for _, n := range gen.Names {
		got = append(got, n.Name.Name)
	}
	if strings.Join(got, ",") != "acceleread,skimurai" {
		t.Errorf("names = %v (duplicates and avoided names must be dropped)", got)
	}
	if gen.Names[0].Technique != "portmanteau" || gen.Names[0].Explanation != "because" || gen.Names[0].Tone != "witty" || gen.Names[0].ID == 0 {
		t.Errorf("name shape = %+v", gen.Names[0])
	}

	// Star, list favourites, unstar.
	id := gen.Names[1].ID
	rec = e.do(t, "PUT", fmt.Sprintf("/api/v1/names/%d/favourite", id), nil)
	if rec.Code != http.StatusOK || !decode[history.SavedName](t, rec).Favourite {
		t.Fatalf("star: %d %s", rec.Code, rec.Body)
	}
	favs := decode[struct{ Favourites []history.SavedName }](t, e.do(t, "GET", "/api/v1/favourites", nil))
	if len(favs.Favourites) != 1 || favs.Favourites[0].Name.Name != "skimurai" || favs.Favourites[0].Description == "" {
		t.Errorf("favourites = %+v", favs)
	}
	if rec := e.do(t, "PUT", "/api/v1/names/9999/favourite", nil); rec.Code != http.StatusNotFound {
		t.Errorf("star missing name: %d", rec.Code)
	}

	// History lists it; deleting keeps the favourite.
	hist := decode[struct {
		Generations []history.Generation
		NextBefore  *int64 `json:"next_before"`
	}](t, e.do(t, "GET", "/api/v1/generations", nil))
	if len(hist.Generations) != 1 || hist.NextBefore != nil || len(hist.Generations[0].Names) != 2 {
		t.Errorf("history = %+v", hist)
	}
	if rec := e.do(t, "DELETE", fmt.Sprintf("/api/v1/generations/%d", gen.ID), nil); rec.Code != http.StatusNoContent {
		t.Errorf("delete: %d %s", rec.Code, rec.Body)
	}
	favs = decode[struct{ Favourites []history.SavedName }](t, e.do(t, "GET", "/api/v1/favourites", nil))
	if len(favs.Favourites) != 1 {
		t.Errorf("favourite lost with its generation: %+v", favs)
	}

	// The prompt carried the inputs, and the schema asked for JSON.
	calls := e.fake.ChatCalls()
	if len(calls) != 1 || len(calls[0].Format) == 0 {
		t.Fatalf("chat calls = %+v", calls)
	}
	var user struct{ Content string }
	json.Unmarshal(calls[0].Messages[1], &user)
	for _, want := range []string{"ingest documents", "maplestory", "Kimi no Name wa"} {
		if !strings.Contains(user.Content, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestGenerateMoreLikeThis(t *testing.T) {
	e := newEnv(t)
	e.fake.SetReplies(reply("readrunner"))
	rec := e.do(t, "POST", "/api/v1/names", map[string]any{
		"description": "fast document reader",
		"like":        map[string]string{"name": "acceleread", "technique": "portmanteau", "explanation": "ACCELErate + READ"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if gen := decode[history.Generation](t, rec); gen.LikeName != "acceleread" || gen.Client != "api" {
		t.Errorf("generation = %+v", gen)
	}
	var user struct{ Content string }
	json.Unmarshal(e.fake.ChatCalls()[0].Messages[1], &user)
	if !strings.Contains(user.Content, `"acceleread"`) || !strings.Contains(user.Content, "same spirit") {
		t.Errorf("prompt lacks the seed:\n%s", user.Content)
	}
}

func TestGenerateErrors(t *testing.T) {
	e := newEnv(t)

	cases := []struct {
		name   string
		setup  func()
		body   any
		status int
		code   string
	}{
		{"missing description", nil, map[string]any{"tones": []string{"witty"}}, 400, "invalid_request"},
		{"unknown tone", nil, map[string]any{"description": "x", "tones": []string{"spicy"}}, 400, "invalid_request"},
		{"model not pulled", func() {
			ollama.SaveConfig(t.Context(), e.settings, ollama.Config{URL: e.fake.URL, Model: "qwen3:32b"})
		}, map[string]any{"description": "x"}, 503, "model_missing"},
		{"useless replies", func() {
			ollama.SaveConfig(t.Context(), e.settings, ollama.Config{URL: e.fake.URL, Model: ollama.DefaultModel})
			e.fake.SetReplies("no json here")
		}, map[string]any{"description": "x"}, 502, "no_names"},
		{"ollama down", func() {
			e.fake.Close()
		}, map[string]any{"description": "x"}, 503, "ollama_unreachable"},
	}
	for _, c := range cases {
		if c.setup != nil {
			c.setup()
		}
		rec := e.do(t, "POST", "/api/v1/names", c.body)
		if rec.Code != c.status || decode[apiError](t, rec).Error.Code != c.code {
			t.Errorf("%s: %d %s, want %d %s", c.name, rec.Code, rec.Body, c.status, c.code)
		}
	}

	// Nothing failed was recorded.
	hist := decode[struct{ Generations []history.Generation }](t, e.do(t, "GET", "/api/v1/generations", nil))
	if len(hist.Generations) != 0 {
		t.Errorf("failed generations were recorded: %+v", hist)
	}
}

func TestGenerateModelTimeout(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Reading the body lets the server notice the client hanging up.
		io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	t.Cleanup(slow.Close)
	e := newEnvWithClient(t, &http.Client{Timeout: 50 * time.Millisecond})
	if _, err := ollama.SaveConfig(t.Context(), e.settings, ollama.Config{URL: slow.URL, Model: ollama.DefaultModel}); err != nil {
		t.Fatal(err)
	}

	rec := e.do(t, "POST", "/api/v1/names", map[string]any{"description": "x"})
	if rec.Code != http.StatusGatewayTimeout || decode[apiError](t, rec).Error.Code != "model_timeout" {
		t.Errorf("got %d %s, want 504 model_timeout", rec.Code, rec.Body)
	}
}

func TestGenerateDialTimeoutIsUnreachable(t *testing.T) {
	dialTimesOut := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return nil, &net.OpError{Op: "dial", Net: network, Err: os.ErrDeadlineExceeded}
		},
	}}
	e := newEnvWithClient(t, dialTimesOut)

	rec := e.do(t, "POST", "/api/v1/names", map[string]any{"description": "x"})
	if rec.Code != http.StatusServiceUnavailable || decode[apiError](t, rec).Error.Code != "ollama_unreachable" {
		t.Errorf("got %d %s, want 503 ollama_unreachable", rec.Code, rec.Body)
	}
}

func TestGenerateRequiresJSONContentType(t *testing.T) {
	e := newEnv(t)
	req := httptest.NewRequest("POST", "/api/v1/names", strings.NewReader(`{"description":"x"}`))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	e.srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("text/plain POST: %d", rec.Code)
	}
}

func TestHealth(t *testing.T) {
	e := newEnv(t)
	type health struct {
		Status string        `json:"status"`
		Ollama ollama.Health `json:"ollama"`
	}

	h := decode[health](t, e.do(t, "GET", "/api/v1/health", nil))
	if h.Status != "ok" || !h.Ollama.Reachable || !h.Ollama.ModelPulled || h.Ollama.Model != ollama.DefaultModel {
		t.Errorf("ready health = %+v", h)
	}

	ollama.SaveConfig(t.Context(), e.settings, ollama.Config{URL: e.fake.URL, Model: "qwen3:32b"})
	h = decode[health](t, e.do(t, "GET", "/api/v1/health", nil))
	if h.Status != "degraded" || !h.Ollama.Reachable || h.Ollama.ModelPulled || !strings.Contains(h.Ollama.Error, "ollama pull qwen3:32b") {
		t.Errorf("missing-model health = %+v", h)
	}

	e.fake.Close()
	rec := e.do(t, "GET", "/api/v1/health", nil)
	h = decode[health](t, rec)
	if rec.Code != 200 || h.Status != "degraded" || h.Ollama.Reachable {
		t.Errorf("down health = %d %+v", rec.Code, h)
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	e := newEnv(t)
	rec := e.do(t, "PUT", "/api/v1/settings", map[string]string{"ollama_url": "http://192.168.1.5:11434/", "model": "gemma4:12b"})
	if rec.Code != 200 {
		t.Fatalf("PUT: %d %s", rec.Code, rec.Body)
	}
	type resp struct {
		URL      string        `json:"ollama_url"`
		Model    string        `json:"model"`
		Defaults ollama.Config `json:"defaults"`
	}
	got := decode[resp](t, e.do(t, "GET", "/api/v1/settings", nil))
	if got.URL != "http://192.168.1.5:11434" || got.Model != "gemma4:12b" || got.Defaults.Model != ollama.DefaultModel {
		t.Errorf("settings = %+v", got)
	}
	if rec := e.do(t, "PUT", "/api/v1/settings", map[string]string{"ollama_url": "nope", "model": "m"}); rec.Code != 400 {
		t.Errorf("bad URL: %d", rec.Code)
	}
}

func TestCheckAndTones(t *testing.T) {
	e := newEnv(t)
	res := decode[namecheck.Result](t, e.do(t, "GET", "/api/v1/check?name=Kimi+no+Name+wa", nil))
	if res.Slug != "kimi-no-name-wa" || res.GitHub.Status != namecheck.Free || res.NPM.Status != namecheck.Free {
		t.Errorf("check = %+v", res)
	}
	if rec := e.do(t, "GET", "/api/v1/check", nil); rec.Code != 400 {
		t.Errorf("check without name: %d", rec.Code)
	}

	tones := decode[struct {
		Tones []struct{ ID string } `json:"tones"`
	}](t, e.do(t, "GET", "/api/v1/tones", nil))
	if len(tones.Tones) < 8 {
		t.Errorf("tones = %+v", tones)
	}
}

func TestRouting(t *testing.T) {
	e := newEnv(t)
	if rec := e.do(t, "GET", "/api/v1/nope", nil); rec.Code != 404 || !strings.Contains(rec.Header().Get("Content-Type"), "json") {
		t.Errorf("unknown API path: %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	if rec := e.do(t, "GET", "/favourites", nil); rec.Code != 200 || !strings.Contains(rec.Body.String(), "stub spa") {
		t.Errorf("SPA route: %d", rec.Code)
	}
}

func TestLocalOnly(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	h := LocalOnly(ok)
	for host, want := range map[string]int{
		"127.0.0.1:7799": 204, "localhost:7799": 204, "[::1]:7799": 204, "LOCALHOST": 204,
		"evil.example:7799": 403, "192.168.1.9:7799": 403,
	} {
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		req.Host = host
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Errorf("Host %s: %d, want %d", host, rec.Code, want)
		}
	}
}
