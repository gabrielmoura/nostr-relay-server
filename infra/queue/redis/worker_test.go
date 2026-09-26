package redisqueue

import (
	"context"
	"errors"
	"testing"
)

func TestIsExpectedShutdown(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if !isExpectedShutdown(ctx, context.Canceled) {
		t.Fatal("context cancellation must be treated as an expected worker shutdown")
	}
	if isExpectedShutdown(ctx, errors.New("redis unavailable")) {
		t.Fatal("non-cancellation error must not be treated as a worker shutdown")
	}
}
