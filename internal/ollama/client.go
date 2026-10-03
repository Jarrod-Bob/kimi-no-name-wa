// Package ollama talks to a local Ollama server over its HTTP API
// (https://github.com/ollama/ollama/blob/main/docs/api.md). It is the app's
// only route to a model; nothing here ever calls a cloud service.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client talks to one Ollama server.
type Client struct {
	BaseURL    string // e.g. "http://127.0.0.1:11434", no trailing slash
	HTTPClient *http.Client
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), HTTPClient: httpClient}
}

// UnreachableError means nothing answered at BaseURL: Ollama isn't installed,
// isn't running, or the URL is wrong.
type UnreachableError struct {
	URL string
	Err error
}

func (e *UnreachableError) Error() string {
	return fmt.Sprintf("Ollama isn't reachable at %s: %v", e.URL, e.Err)
}

func (e *UnreachableError) Unwrap() error { return e.Err }

// ModelMissingError means Ollama is running but the model hasn't been pulled.
type ModelMissingError struct {
	Model string
}

func (e *ModelMissingError) Error() string {
	return fmt.Sprintf("model %q isn't pulled; run: ollama pull %s", e.Model, e.Model)
}

// APIError is any other non-2xx answer, with Ollama's {"error":"…"} message
// when it sent one.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("ollama: %d %s", e.StatusCode, http.StatusText(e.StatusCode))
	}
	return fmt.Sprintf("ollama: %d %s", e.StatusCode, e.Message)
}

// Version returns the server's version string (GET /api/version).
func (c *Client) Version(ctx context.Context) (string, error) {
	var out struct {
		Version string `json:"version"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/version", nil, &out); err != nil {
		return "", err
	}
	return out.Version, nil
}

// Models lists the models pulled on the server (GET /api/tags), e.g.
// "gemma4:31b".
func (c *Client) Models(ctx context.Context) ([]string, error) {
	var out struct {
		Models []struct {
			Name  string `json:"name"`
			Model string `json:"model"`
		} `json:"models"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/tags", nil, &out); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(out.Models))
	for _, m := range out.Models {
		name := m.Name
		if name == "" {
			name = m.Model
		}
		if name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

// HasModel reports whether model is among pulled. A model named without a tag
// ("gemma4") means its ":latest" tag, as it does on the Ollama CLI.
func HasModel(pulled []string, model string) bool {
	want := strings.ToLower(strings.TrimSpace(model))
	if want == "" {
		return false
	}
	if !strings.Contains(want, ":") {
		want += ":latest"
	}
	for _, p := range pulled {
		have := strings.ToLower(p)
		if !strings.Contains(have, ":") {
			have += ":latest"
		}
		if have == want {
			return true
		}
	}
	return false
}

// ChatRequest is one non-streaming chat turn: a system prompt, a user
// message, and a JSON schema the reply must follow.
type ChatRequest struct {
	Model   string
	System  string
	User    string
	Format  json.RawMessage // JSON schema for structured output; nil for free text
	Options map[string]any  // sampling options (temperature, top_p, ...)
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatBody struct {
	Model    string          `json:"model"`
	Messages []chatMessage   `json:"messages"`
	Stream   bool            `json:"stream"`
	Format   json.RawMessage `json:"format,omitempty"`
	Options  map[string]any  `json:"options,omitempty"`
	// Think is false so reasoning models answer straight away instead of
	// spending a minute deliberating over a list of puns. It is a pointer so
	// the retry for models that reject the field can leave it out.
	Think *bool `json:"think,omitempty"`
}

// Chat sends req to POST /api/chat and returns the assistant's reply text.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (string, error) {
	noThink := false
	body := chatBody{
		Model: req.Model,
		Messages: []chatMessage{
			{Role: "system", Content: req.System},
			{Role: "user", Content: req.User},
		},
		Format:  req.Format,
		Options: req.Options,
		Think:   &noThink,
	}
	var out struct {
		Message chatMessage `json:"message"`
	}
	err := c.do(ctx, http.MethodPost, "/api/chat", body, &out)
	// Some Ollama versions refuse the think field for models without a
	// thinking mode; ask again without it.
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusBadRequest &&
		strings.Contains(strings.ToLower(apiErr.Message), "think") {
		body.Think = nil
		err = c.do(ctx, http.MethodPost, "/api/chat", body, &out)
	}
	if err != nil {
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return "", &ModelMissingError{Model: req.Model}
		}
		return "", err
	}
	return out.Message.Content, nil
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	var reader io.Reader
	if in != nil {
		encoded, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("encoding request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		// A malformed BaseURL never reaches a server, so it reads as
		// unreachable to the person fixing it in Settings.
		return &UnreachableError{URL: c.BaseURL, Err: err}
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// The client's own timeout means Ollama accepted the request but
		// didn't answer in time, which is normal while a big model loads.
		var urlErr *url.Error
		if errors.As(err, &urlErr) && urlErr.Timeout() {
			return fmt.Errorf("%w: %v", context.DeadlineExceeded, err)
		}
		return &UnreachableError{URL: c.BaseURL, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		var envelope struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &envelope)
		return &APIError{StatusCode: resp.StatusCode, Message: envelope.Error}
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding ollama response from %s: %w", path, err)
	}
	return nil
}
