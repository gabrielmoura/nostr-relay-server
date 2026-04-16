package service

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/cache"
	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/contracts"
	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/engine"
	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/model"
)

var (
	ErrSessionNotFound = errors.New("unknown session handle")
	ErrInvalidMessage  = errors.New("invalid negentropy payload")
)

type Options struct {
	FrameSizeLimit int
	Buckets        int
	CacheTTL       time.Duration
	SessionTTL     time.Duration
}

type Manager struct {
	store contracts.EventStore
	cache contracts.QueryCache

	frameSizeLimit int
	buckets        int
	cacheTTL       time.Duration
	sessionTTL     time.Duration

	mu       sync.RWMutex
	sessions map[string]*session
	inflight map[string]*inflightCall
}

type session struct {
	id         string
	filterKey  string
	reconciler *engine.Reconciler
	updatedAt  time.Time
}

type inflightCall struct {
	done chan struct{}
	refs []model.EventRef
	err  error
}

func NewManager(store contracts.EventStore, queryCache contracts.QueryCache, opts Options) *Manager {
	if store == nil {
		panic("negentropyV2: nil EventStore")
	}

	if queryCache == nil {
		queryCache = cache.NewMemoryQueryCache()
	}

	if opts.CacheTTL <= 0 {
		opts.CacheTTL = 30 * time.Second
	}

	if opts.SessionTTL <= 0 {
		opts.SessionTTL = 60 * time.Second
	}

	return &Manager{
		store:          store,
		cache:          queryCache,
		frameSizeLimit: opts.FrameSizeLimit,
		buckets:        opts.Buckets,
		cacheTTL:       opts.CacheTTL,
		sessionTTL:     opts.SessionTTL,
		sessions:       make(map[string]*session),
		inflight:       make(map[string]*inflightCall),
	}
}

func (m *Manager) Open(ctx context.Context, req model.OpenRequest) (model.Response, error) {
	m.PurgeExpired()

	initial, err := hex.DecodeString(req.InitialMessageHex)
	if err != nil {
		return model.Response{}, fmt.Errorf("%w: %v", ErrInvalidMessage, err)
	}

	if req.SessionID == "" {
		return model.Response{}, fmt.Errorf("%w: empty session id", ErrInvalidMessage)
	}

	filterKey, err := cache.BuildFilterKey(req.Filter)
	if err != nil {
		return model.Response{}, err
	}

	refs, err := m.loadRefs(ctx, filterKey, req.Filter)
	if err != nil {
		return model.Response{}, err
	}

	reconciler, err := engine.NewReconciler(refs, engine.Options{
		FrameSizeLimit: m.frameSizeLimit,
		Buckets:        m.buckets,
	})
	if err != nil {
		return model.Response{}, err
	}

	out, err := reconciler.ReconcileAsResponder(initial)
	if err != nil {
		return m.makeErr(req.SessionID, "protocol-error: bad negentropy message"), nil
	}

	m.mu.Lock()
	m.sessions[req.SessionID] = &session{
		id:         req.SessionID,
		filterKey:  filterKey,
		reconciler: reconciler,
		updatedAt:  time.Now(),
	}
	m.mu.Unlock()

	return model.Response{
		Type:       model.ResponseTypeMessage,
		SessionID:  req.SessionID,
		MessageHex: hex.EncodeToString(out),
		Done:       len(out) == 1,
	}, nil
}

func (m *Manager) OnMessage(_ context.Context, req model.MessageRequest) (model.Response, error) {
	m.PurgeExpired()

	if req.SessionID == "" {
		return model.Response{}, fmt.Errorf("%w: empty session id", ErrInvalidMessage)
	}

	raw, err := hex.DecodeString(req.MessageHex)
	if err != nil {
		return model.Response{}, fmt.Errorf("%w: %v", ErrInvalidMessage, err)
	}

	s, ok := m.getSession(req.SessionID)
	if !ok {
		return m.makeErr(req.SessionID, "closed: unknown subscription handle"), nil
	}

	out, err := s.reconciler.ReconcileAsResponder(raw)
	if err != nil {
		m.Close(req.SessionID)
		return m.makeErr(req.SessionID, "protocol-error: bad negentropy message"), nil
	}

	m.touchSession(req.SessionID)

	return model.Response{
		Type:       model.ResponseTypeMessage,
		SessionID:  req.SessionID,
		MessageHex: hex.EncodeToString(out),
		Done:       len(out) == 1,
	}, nil
}

func (m *Manager) Close(sessionID string) {
	m.mu.Lock()
	delete(m.sessions, sessionID)
	m.mu.Unlock()
}

func (m *Manager) PurgeExpired() {
	now := time.Now()

	m.mu.Lock()
	for id, s := range m.sessions {
		if now.Sub(s.updatedAt) > m.sessionTTL {
			delete(m.sessions, id)
		}
	}
	m.mu.Unlock()

	m.cache.PurgeExpired(now)
}

func (m *Manager) makeErr(sessionID string, reason string) model.Response {
	return model.Response{
		Type:      model.ResponseTypeError,
		SessionID: sessionID,
		Reason:    reason,
	}
}

func (m *Manager) getSession(id string) (*session, bool) {
	m.mu.RLock()
	s, ok := m.sessions[id]
	m.mu.RUnlock()

	return s, ok
}

func (m *Manager) touchSession(id string) {
	m.mu.Lock()
	if s, ok := m.sessions[id]; ok {
		s.updatedAt = time.Now()
	}
	m.mu.Unlock()
}

func (m *Manager) loadRefs(ctx context.Context, key string, filter model.Filter) ([]model.EventRef, error) {
	if refs, ok := m.cache.Get(key); ok {
		return refs, nil
	}

	result, err := m.loadOnce(ctx, key, func() ([]model.EventRef, error) {
		refs, loadErr := m.store.QueryEventRefs(ctx, filter)
		if loadErr != nil {
			return nil, loadErr
		}

		m.cache.Set(key, refs, m.cacheTTL)

		return refs, nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (m *Manager) loadOnce(ctx context.Context, key string, fn func() ([]model.EventRef, error)) ([]model.EventRef, error) {
	m.mu.Lock()
	if call, ok := m.inflight[key]; ok {
		m.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-call.done:
		}

		if call.err != nil {
			return nil, call.err
		}

		out := make([]model.EventRef, len(call.refs))
		copy(out, call.refs)

		return out, nil
	}

	call := &inflightCall{done: make(chan struct{})}
	m.inflight[key] = call
	m.mu.Unlock()

	refs, err := fn()

	m.mu.Lock()
	call.refs = refs
	call.err = err
	close(call.done)
	delete(m.inflight, key)
	m.mu.Unlock()

	if err != nil {
		return nil, err
	}

	out := make([]model.EventRef, len(refs))
	copy(out, refs)

	return out, nil
}
