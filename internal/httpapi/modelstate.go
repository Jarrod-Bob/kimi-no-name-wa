package httpapi

import (
	"sync"

	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/ollama"
)

// modelFailure remembers whether the last generation failed because Ollama
// couldn't run the model (model_error), e.g. a GPU that won't initialise.
// The model can be pulled and Ollama reachable while every generation still
// fails, and only trying to run it shows that, so health reports what the
// last attempt found instead of loading the model itself.
type modelFailure struct {
	mu     sync.Mutex
	failed *ollama.Config // the URL and model that last failed; nil once one works
}

func (m *modelFailure) record(cfg ollama.Config, failed bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch {
	case failed:
		m.failed = &cfg
	case m.failed != nil && *m.failed == cfg:
		m.failed = nil
	}
}

// failedFor reports whether the last generation with cfg failed to run the model.
func (m *modelFailure) failedFor(cfg ollama.Config) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.failed != nil && *m.failed == cfg
}
