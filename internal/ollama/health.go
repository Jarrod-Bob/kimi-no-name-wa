package ollama

import (
	"context"
	"errors"
	"time"
)

// Health is what GET /api/v1/health reports about the model server.
type Health struct {
	URL         string   `json:"url"`
	Reachable   bool     `json:"reachable"`
	Version     string   `json:"version,omitempty"`
	Model       string   `json:"model"`
	ModelPulled bool     `json:"model_pulled"`
	Models      []string `json:"models"`
	// Error is a sentence for a person: why the server or model isn't
	// usable, and what to do about it. Empty when everything is ready.
	Error string `json:"error,omitempty"`
}

// Ready reports whether a generation can be attempted.
func (h Health) Ready() bool { return h.Reachable && h.ModelPulled }

// Probe asks the server for its version and pulled models. It never fails:
// every problem is described in the returned Health instead.
func Probe(ctx context.Context, c *Client, model string) Health {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	h := Health{URL: c.BaseURL, Model: model, Models: []string{}}
	models, err := c.Models(ctx)
	if err != nil {
		var unreachable *UnreachableError
		if errors.As(err, &unreachable) || errors.Is(err, context.DeadlineExceeded) {
			h.Error = "Ollama isn't reachable at " + c.BaseURL + ". Install it from https://ollama.com and make sure it's running, or fix the URL in Settings."
		} else {
			h.Error = "Ollama answered at " + c.BaseURL + " but listing models failed: " + err.Error()
		}
		return h
	}
	h.Reachable = true
	h.Models = models
	if v, err := c.Version(ctx); err == nil {
		h.Version = v
	}
	h.ModelPulled = HasModel(models, model)
	if !h.ModelPulled {
		h.Error = "Ollama is running but the model " + model + " isn't pulled yet. Run: ollama pull " + model
	}
	return h
}
