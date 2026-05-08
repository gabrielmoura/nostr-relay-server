package jobs

import (
	"context"
	"time"

	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
)

type Job interface {
	Name() string
}

type RegisteredHandler struct {
	Decode func([]byte) (Job, error)
	Handle func(context.Context, Job) error
}

type Dispatcher interface {
	Dispatch(ctx context.Context, job Job, opts ...DispatchOption) (JobID, error)
	DispatchIn(ctx context.Context, job Job, delay time.Duration, opts ...DispatchOption) (JobID, error)
	DispatchAt(ctx context.Context, job Job, runAt time.Time, opts ...DispatchOption) (JobID, error)
}

type Monitor interface {
	Get(ctx context.Context, queue string, id JobID) (Snapshot, error)
	List(ctx context.Context, queue string, filter ListFilter) ([]Snapshot, error)
	Retry(ctx context.Context, queue string, id JobID) error
	Cancel(ctx context.Context, queue string, id JobID) error
	Resume(ctx context.Context, queue string, id JobID) error
	Delete(ctx context.Context, queue string, filter DeleteFilter) (int64, error)
}

type Registry interface {
	Register(name string, handler RegisteredHandler) error
	Lookup(name string) (RegisteredHandler, bool)
}

type Snapshot struct {
	ID          JobID
	Queue       string
	Priority    Priority
	Name        string
	Status      Status
	Attempts    uint8
	MaxAttempts uint8
	CreatedAt   time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
	RunAt       *time.Time
	LastError   string
	Payload     json.RawMessage
	Result      json.RawMessage
}

type ListFilter struct {
	Limit int64
}

type DeleteFilter struct {
	JobName  string
	Statuses []Status
}
