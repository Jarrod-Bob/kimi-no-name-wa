package namegen

import (
	"context"
	"errors"

	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/ollama"
)

// Chatter is the model: ollama.Client satisfies it.
type Chatter interface {
	Chat(ctx context.Context, req ollama.ChatRequest) (string, error)
}

// ErrNoNames means the model answered twice without a single usable name.
var ErrNoNames = errors.New("the model didn't return any usable names; try again, or try a different model")

// samplingOptions are Google's recommended settings for Gemma 4, warm enough
// for wordplay. num_ctx leaves room for a long avoid list.
var samplingOptions = map[string]any{
	"temperature": 1.0,
	"top_p":       0.95,
	"top_k":       64,
	"num_ctx":     8192,
}

// Generate asks model for names matching req, which must already be
// Normalize'd. It asks for a couple more than req.Count to make up for
// entries that fail validation. When a reply leaves fewer than half the
// names wanted (typically a re-roll where the model repeated names it was
// told to avoid), it asks once more, avoiding those too, and merges.
func Generate(ctx context.Context, chat Chatter, model string, req Request) ([]Name, error) {
	var names []Name
	for attempt := 0; attempt < 2; attempt++ {
		ask := req
		ask.Count = min(req.Count-len(names)+2, MaxCount+2)
		ask.Avoid = append(append([]string(nil), req.Avoid...), namesOf(names)...)
		content, err := chat.Chat(ctx, ollama.ChatRequest{
			Model:   model,
			System:  systemPrompt,
			User:    buildUserPrompt(ask),
			Format:  schema(req.Tones),
			Options: samplingOptions,
		})
		if err != nil {
			if len(names) > 0 {
				// The first reply was usable; don't throw it away.
				break
			}
			return nil, err
		}
		fresh, _ := Parse(content, req.Tones, ask.Avoid)
		names = append(names, fresh...)
		if len(names)*2 >= req.Count {
			break
		}
	}
	if len(names) == 0 {
		return nil, ErrNoNames
	}
	if len(names) > req.Count {
		names = names[:req.Count]
	}
	return names, nil
}

func namesOf(names []Name) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = n.Name
	}
	return out
}
