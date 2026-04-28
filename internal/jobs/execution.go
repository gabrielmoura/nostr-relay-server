package jobs

import (
	"context"
	"fmt"

	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
)

type executionContextKey struct{}

type ExecutionState struct {
	ID       JobID
	Queue    string
	Name     string
	result   any
	resultOK bool
}

func WithExecutionState(ctx context.Context, state *ExecutionState) context.Context {
	return context.WithValue(ctx, executionContextKey{}, state)
}

func ExecutionFromContext(ctx context.Context) (*ExecutionState, bool) {
	state, ok := ctx.Value(executionContextKey{}).(*ExecutionState)
	return state, ok
}

func SetResult(ctx context.Context, result any) error {
	state, ok := ExecutionFromContext(ctx)
	if !ok {
		return fmt.Errorf("job execution state not found in context")
	}
	state.result = result
	state.resultOK = true
	return nil
}

func ResultJSON(ctx context.Context) (string, error) {
	state, ok := ExecutionFromContext(ctx)
	if !ok || !state.resultOK {
		return "", nil
	}
	payload, err := json.Marshal(state.result)
	if err != nil {
		return "", fmt.Errorf("marshal job result: %w", err)
	}
	return string(payload), nil
}
