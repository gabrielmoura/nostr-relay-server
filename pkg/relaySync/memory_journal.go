package relaysync

import (
	"context"
	"io"
	"sync"
)

func contextErr(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// MemoryJournalProvider is a bounded journal intended for tests and local
// adapters. It is not durable across process restarts.
type MemoryJournalProvider struct {
	mu        sync.RWMutex
	epoch     uint64
	retention int
	changes   []JournalChange
	next      uint64
}

func NewMemoryJournalProvider(epoch uint64, retention int) *MemoryJournalProvider {
	if epoch == 0 {
		epoch = 1
	}
	if retention <= 0 {
		retention = 1024
	}
	return &MemoryJournalProvider{epoch: epoch, retention: retention}
}

func (p *MemoryJournalProvider) Append(change JournalChange) Cursor {
	p.mu.Lock()
	defer p.mu.Unlock()
	change.Cursor.Epoch = p.epoch
	p.next++
	change.Cursor.Sequence = p.next
	p.changes = append(p.changes, change)
	if len(p.changes) > p.retention {
		p.changes = p.changes[len(p.changes)-p.retention:]
	}
	return change.Cursor
}

func (p *MemoryJournalProvider) OpenJournal(ctx context.Context, _ Scope, after Cursor) (Journal, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if after.Epoch != 0 && after.Epoch != p.epoch {
		return nil, ErrJournalEpochChanged
	}
	first := p.next + 1
	if len(p.changes) > 0 {
		first = p.changes[0].Cursor.Sequence
	}
	if after.Sequence+1 < first && len(p.changes) > 0 {
		return nil, ErrJournalGap
	}
	start := 0
	for start < len(p.changes) && p.changes[start].Cursor.Sequence <= after.Sequence {
		start++
	}
	items := append([]JournalChange(nil), p.changes[start:]...)
	return &memoryJournal{epoch: p.epoch, changes: items}, nil
}

type memoryJournal struct {
	mu      sync.Mutex
	epoch   uint64
	changes []JournalChange
	index   int
	closed  bool
}

func (j *memoryJournal) Epoch() uint64 { return j.epoch }
func (j *memoryJournal) Next(ctx context.Context) (JournalChange, error) {
	if err := contextErr(ctx); err != nil {
		return JournalChange{}, err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.closed {
		return JournalChange{}, io.EOF
	}
	if j.index >= len(j.changes) {
		return JournalChange{}, io.EOF
	}
	change := j.changes[j.index]
	j.index++
	return change, nil
}
func (j *memoryJournal) Close() error { j.mu.Lock(); j.closed = true; j.mu.Unlock(); return nil }
