package jobs

import (
	"context"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/internal/events"
)

// ReportProgress publishes a best-effort progress update for the job bound to
// ctx. It is safe to call outside a queue execution: the call is ignored.
func ReportProgress(ctx context.Context, current, total int64, stage string) {
	state, ok := ExecutionFromContext(ctx)
	if !ok || state == nil {
		return
	}
	PublishEvent(events.JobEvent{
		ID:              state.ID.String(),
		Queue:           state.Queue,
		Name:            state.Name,
		Status:          StatusRunning.String(),
		ProgressCurrent: current,
		ProgressTotal:   total,
		Stage:           stage,
		UpdatedAt:       time.Now().UTC(),
	})
}

// PublishEvent emits a durable job transition on the shared event bus.
func PublishEvent(event events.JobEvent) {
	if event.UpdatedAt.IsZero() {
		event.UpdatedAt = time.Now().UTC()
	}
	events.Default().Publish(events.TopicJob, event)
}
