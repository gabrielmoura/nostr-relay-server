// Package relaysync contains NRServer-specific lifecycle wiring for the
// framework-independent RelaySync core.
package relaysync

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Projector is the small persistence port required by the supervised worker.
// The infra/db Queries type implements it without exposing database details to
// the lifecycle code.
type Projector interface {
	ProjectRelaySyncOutbox(context.Context, uint64, int) (int, error)
	PurgeRelaySyncOutbox(context.Context, time.Time, int) (int64, error)
	PurgeRelaySyncJournal(context.Context, time.Time, int) (int64, error)
}

type Options struct {
	Projector          Projector
	Epoch              uint64
	Interval           time.Duration
	BatchSize          int
	OutboxRetention    time.Duration
	JournalRetention   time.Duration
	RetentionBatchSize int
	Now                func() time.Time
}

// Runtime is a finite-cycle projector worker. It has no global state and is
// safe to close more than once.
type Runtime struct {
	cancel  context.CancelFunc
	done    chan struct{}
	once    sync.Once
	mu      sync.RWMutex
	lastErr error
	cycles  uint64
}

func Start(ctx context.Context, opts Options) (*Runtime, error) {
	if opts.Projector == nil {
		return nil, errors.New("relaysync: projector is required")
	}
	if opts.Epoch == 0 || opts.Interval <= 0 || opts.BatchSize <= 0 || opts.RetentionBatchSize <= 0 {
		return nil, errors.New("relaysync: invalid projector options")
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	workerCtx, cancel := context.WithCancel(ctx)
	r := &Runtime{cancel: cancel, done: make(chan struct{})}
	go r.run(workerCtx, opts)
	return r, nil
}

func (r *Runtime) run(ctx context.Context, opts Options) {
	defer close(r.done)
	defer r.cancel()
	// Project immediately so a restart does not wait for a full interval.
	r.cycle(ctx, opts)
	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.cycle(ctx, opts)
		}
	}
}

func (r *Runtime) cycle(ctx context.Context, opts Options) {
	if ctx.Err() != nil {
		return
	}
	_, err := opts.Projector.ProjectRelaySyncOutbox(ctx, opts.Epoch, opts.BatchSize)
	if err == nil && opts.OutboxRetention > 0 {
		_, err = opts.Projector.PurgeRelaySyncOutbox(ctx, opts.Now().Add(-opts.OutboxRetention), opts.RetentionBatchSize)
	}
	if err == nil && opts.JournalRetention > 0 {
		_, err = opts.Projector.PurgeRelaySyncJournal(ctx, opts.Now().Add(-opts.JournalRetention), opts.RetentionBatchSize)
	}
	r.mu.Lock()
	r.cycles++
	r.lastErr = err
	r.mu.Unlock()
}

func (r *Runtime) Close(ctx context.Context) error {
	if r == nil {
		return nil
	}
	r.once.Do(r.cancel)
	select {
	case <-r.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type Status struct {
	Cycles  uint64
	LastErr error
}

func (r *Runtime) Status() Status {
	if r == nil {
		return Status{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return Status{Cycles: r.cycles, LastErr: r.lastErr}
}
