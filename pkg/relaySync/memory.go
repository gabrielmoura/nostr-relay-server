package relaysync

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Clock supports deterministic session expiration tests.
type Clock interface{ Now() time.Time }

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// MemorySessionStore is process-local and intended for tests or single-process
// deployments. It does not provide distributed session affinity.
type MemorySessionStore struct {
	mu       sync.Mutex
	clock    Clock
	sessions map[SessionID]SessionRecord
}

func NewMemorySessionStore(clock Clock) *MemorySessionStore {
	if clock == nil {
		clock = systemClock{}
	}
	return &MemorySessionStore{clock: clock, sessions: make(map[SessionID]SessionRecord)}
}

func (s *MemorySessionStore) CreateSession(ctx context.Context, session SessionRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if session.ExpiresAt.IsZero() {
		return &ValidationError{Field: "session.expires_at", Reason: "is required"}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked()
	if _, exists := s.sessions[session.ID]; exists {
		return ErrConflict
	}
	if session.Version == 0 {
		session.Version = 1
	}
	s.sessions[session.ID] = cloneSession(session)
	return nil
}

func (s *MemorySessionStore) GetSession(ctx context.Context, id SessionID) (SessionRecord, error) {
	if err := ctx.Err(); err != nil {
		return SessionRecord{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked()
	session, ok := s.sessions[id]
	if !ok {
		return SessionRecord{}, ErrNotFound
	}
	return cloneSession(session), nil
}

func (s *MemorySessionStore) UpdateSession(ctx context.Context, update SessionUpdate) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if update.ExpiresAt.IsZero() {
		return &ValidationError{Field: "session.expires_at", Reason: "is required"}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked()
	session, ok := s.sessions[update.ID]
	if !ok {
		return ErrNotFound
	}
	if session.Version != update.ExpectedVersion {
		return ErrConflict
	}
	if update.Watermark.Epoch != session.Watermark.Epoch || update.Watermark.Sequence < session.Watermark.Sequence {
		return ErrConflict
	}
	session.State = update.State
	session.Watermark = update.Watermark
	session.ExpiresAt = update.ExpiresAt
	session.Version++
	s.sessions[update.ID] = session
	return nil
}

func (s *MemorySessionStore) DeleteSession(ctx context.Context, id SessionID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}

func (s *MemorySessionStore) purgeExpiredLocked() {
	now := s.clock.Now()
	for id, session := range s.sessions {
		if !session.ExpiresAt.After(now) {
			delete(s.sessions, id)
		}
	}
}

func cloneSession(session SessionRecord) SessionRecord {
	session.SnapshotVersion = append(SnapshotVersion{}, session.SnapshotVersion...)
	return session
}

// MemoryCheckpointStore is a process-local, mutex-protected checkpoint store.
type MemoryCheckpointStore struct {
	mu          sync.Mutex
	checkpoints map[CheckpointKey]Checkpoint
}

func NewMemoryCheckpointStore() *MemoryCheckpointStore {
	return &MemoryCheckpointStore{checkpoints: make(map[CheckpointKey]Checkpoint)}
}

func (s *MemoryCheckpointStore) LoadCheckpoint(ctx context.Context, key CheckpointKey) (Checkpoint, bool, error) {
	if err := ctx.Err(); err != nil {
		return Checkpoint{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	checkpoint, ok := s.checkpoints[key]
	return checkpoint, ok, nil
}

func (s *MemoryCheckpointStore) AdvanceCheckpoint(ctx context.Context, next Checkpoint) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.checkpoints[next.Key]
	if ok && (current.Cursor.Epoch != next.Cursor.Epoch || next.Cursor.Sequence < current.Cursor.Sequence) {
		return ErrConflict
	}
	s.checkpoints[next.Key] = next
	return nil
}

func (s *MemoryCheckpointStore) ResetCheckpoint(ctx context.Context, next Checkpoint) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkpoints[next.Key] = next
	return nil
}

// MemoryPeerAuthorizer is a minimal explicit grant registry for tests.
type MemoryPeerAuthorizer struct {
	mu     sync.RWMutex
	grants map[string]AuthorizedScope
}

func NewMemoryPeerAuthorizer() *MemoryPeerAuthorizer {
	return &MemoryPeerAuthorizer{grants: make(map[string]AuthorizedScope)}
}

func (a *MemoryPeerAuthorizer) Grant(principal Principal, scope AuthorizedScope) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.grants[principal.ID] = cloneAuthorizedScope(scope)
}

func (a *MemoryPeerAuthorizer) Authorize(ctx context.Context, principal Principal, action Action, requested Scope) (AuthorizedScope, error) {
	if err := ctx.Err(); err != nil {
		return AuthorizedScope{}, err
	}
	a.mu.RLock()
	grant, ok := a.grants[principal.ID]
	a.mu.RUnlock()
	if !ok || grant.Scope != requested {
		return AuthorizedScope{}, ErrUnauthorized
	}
	for _, allowed := range grant.Actions {
		if allowed == action {
			return cloneAuthorizedScope(grant), nil
		}
	}
	return AuthorizedScope{}, fmt.Errorf("%w: action not granted", ErrUnauthorized)
}

func cloneAuthorizedScope(scope AuthorizedScope) AuthorizedScope {
	scope.Actions = append([]Action{}, scope.Actions...)
	return scope
}
