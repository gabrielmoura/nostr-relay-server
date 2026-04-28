package jobs

import (
	"context"
	"fmt"
	"sync"

	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
)

type MemoryRegistry struct {
	mu       sync.RWMutex
	handlers map[string]RegisteredHandler
}

func NewRegistry() *MemoryRegistry {
	return &MemoryRegistry{handlers: make(map[string]RegisteredHandler)}
}

func (r *MemoryRegistry) Register(name string, handler RegisteredHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if name == "" {
		return fmt.Errorf("handler name is required")
	}
	if handler.Decode == nil {
		return fmt.Errorf("handler %q decode is required", name)
	}
	if handler.Handle == nil {
		return fmt.Errorf("handler %q handle is required", name)
	}

	r.handlers[name] = handler
	return nil
}

func (r *MemoryRegistry) Lookup(name string) (RegisteredHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, ok := r.handlers[name]
	return handler, ok
}

func RegisterTyped[T Job](r *MemoryRegistry, name string, handler func(context.Context, T) error) error {
	return r.Register(name, RegisteredHandler{
		Decode: func(payload []byte) (Job, error) {
			var value T
			if err := json.Unmarshal(payload, &value); err != nil {
				return nil, fmt.Errorf("decode %s payload: %w", name, err)
			}
			return value, nil
		},
		Handle: func(ctx context.Context, job Job) error {
			typed, ok := job.(T)
			if !ok {
				return fmt.Errorf("job %s has unexpected type %T", name, job)
			}
			return handler(ctx, typed)
		},
	})
}
