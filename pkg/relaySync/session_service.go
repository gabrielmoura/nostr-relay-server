package relaysync

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"sync"
	"time"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
	"google.golang.org/protobuf/proto"
)

func limitsToProto(limits Limits) *relaysyncv2.Limits {
	return &relaysyncv2.Limits{MaxEventBytes: uint32(limits.MaxEventBytes), MaxTagCount: uint32(limits.MaxTagCount), MaxTagValues: uint32(limits.MaxTagValues), MaxTagValueBytes: uint32(limits.MaxTagValueBytes), MaxEventsPerBatch: uint32(limits.MaxEventsPerBatch), MaxBatchBytes: uint32(limits.MaxBatchBytes), MaxFrameBytes: uint32(limits.MaxFrameBytes), MaxSessionTtlSeconds: uint32(limits.MaxSessionTTL / time.Second), MaxConcurrentRounds: uint32(limits.MaxConcurrentRounds)}
}
func snapshotToProto(descriptor SnapshotDescriptor) *relaysyncv2.Snapshot {
	return &relaysyncv2.Snapshot{Version: append([]byte(nil), descriptor.Version...), Scope: &relaysyncv2.Scope{Name: descriptor.Scope.Name, Version: descriptor.Scope.Version, Hash: append([]byte(nil), descriptor.Scope.Hash[:]...)}, Watermark: &relaysyncv2.Cursor{Epoch: descriptor.Watermark.Epoch, Sequence: descriptor.Watermark.Sequence}, ExpiresAtUnix: descriptor.ExpiresAt.Unix()}
}

var ErrSessionToken = errors.New("relaysync: invalid session token")

type SessionServiceOptions struct {
	Store      SessionStore
	Snapshots  SnapshotProvider
	Authorizer PeerAuthorizer
	Clock      Clock
	Limits     Limits
	SessionTTL time.Duration
}

type SessionService struct {
	options     SessionServiceOptions
	mu          sync.Mutex
	snapshots   map[SessionID]Snapshot
	idempotent  map[string]sessionResult
	finishHooks []func(SessionID)
}

type sessionResult struct {
	sessionID   SessionID
	response    *relaysyncv2.OpenSessionResponse
	token       []byte
	requestHash [32]byte
}

func NewSessionService(options SessionServiceOptions) (*SessionService, error) {
	if options.Store == nil || options.Authorizer == nil {
		return nil, ErrUnauthorized
	}
	if options.Clock == nil {
		options.Clock = systemClock{}
	}
	if options.SessionTTL <= 0 {
		options.SessionTTL = 30 * time.Minute
	}
	options.Limits = options.Limits.WithDefaults()
	if err := options.Limits.Validate(); err != nil {
		return nil, err
	}
	return &SessionService{options: options, snapshots: make(map[SessionID]Snapshot), idempotent: make(map[string]sessionResult)}, nil
}

func (s *SessionService) Open(ctx context.Context, principal Principal, request *relaysyncv2.OpenSessionRequest) (*relaysyncv2.OpenSessionResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if request == nil || request.Scope == nil || request.ProtocolVersion != 2 {
		return nil, ErrInvalidEvent
	}
	if len(request.Scope.Hash) != 0 && len(request.Scope.Hash) != 32 {
		return nil, ErrInvalidIdentifier
	}
	scope := Scope{Name: request.Scope.Name, Version: request.Scope.Version}
	copy(scope.Hash[:], request.Scope.Hash)
	if _, err := s.options.Authorizer.Authorize(ctx, principal, ActionOpenSession, scope); err != nil {
		return nil, err
	}
	key := principal.ID + "\x00" + scope.Name + "\x00" + string(request.IdempotencyKey)
	requestBytes, err := (proto.MarshalOptions{Deterministic: true}).Marshal(request)
	if err != nil {
		return nil, err
	}
	requestHash := sha256.Sum256(requestBytes)
	if len(request.IdempotencyKey) > 0 {
		s.mu.Lock()
		existing, ok := s.idempotent[key]
		s.mu.Unlock()
		if ok {
			if subtle.ConstantTimeCompare(existing.requestHash[:], requestHash[:]) != 1 {
				return nil, ErrConflict
			}
			return cloneSessionResponse(existing.response), nil
		}
	}
	var id SessionID
	if _, err := rand.Read(id[:]); err != nil {
		return nil, err
	}
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return nil, err
	}
	now := s.options.Clock.Now()
	record := SessionRecord{ID: id, TokenHash: sha256.Sum256(token), Peer: principal, Scope: scope, State: SessionStateCreated, ExpiresAt: now.Add(s.options.SessionTTL)}
	response := &relaysyncv2.OpenSessionResponse{SessionId: append([]byte(nil), id[:]...), ProtocolVersion: 2, Compression: relaysyncv2.Compression_COMPRESSION_NONE, Limits: limitsToProto(s.options.Limits), ResumeToken: append([]byte(nil), token...)}
	if s.options.Snapshots != nil {
		snapshot, err := s.options.Snapshots.OpenSnapshot(ctx, scope)
		if err != nil {
			return nil, err
		}
		descriptor := snapshot.Descriptor()
		record.SnapshotVersion = append([]byte(nil), descriptor.Version...)
		record.Watermark = descriptor.Watermark
		response.Snapshot = snapshotToProto(descriptor)
		s.mu.Lock()
		s.snapshots[id] = snapshot
		s.mu.Unlock()
	}
	if err := s.options.Store.CreateSession(ctx, record); err != nil {
		if snapshot := s.takeSnapshot(id); snapshot != nil {
			_ = snapshot.Close()
		}
		return nil, err
	}
	if len(request.IdempotencyKey) > 0 {
		s.mu.Lock()
		s.idempotent[key] = sessionResult{sessionID: id, response: cloneSessionResponse(response), token: append([]byte(nil), token...), requestHash: requestHash}
		s.mu.Unlock()
	}
	return response, nil
}

func (s *SessionService) Delete(ctx context.Context, principal Principal, id SessionID, token []byte) error {
	return s.finish(ctx, principal, id, token)
}

// Complete closes a successful finite synchronization cycle. It removes the
// durable session record and all process-local snapshot/idempotency state.
// Checkpoints, when used by the caller, are intentionally managed by the
// CheckpointStore and are not deleted here.
func (s *SessionService) Complete(ctx context.Context, principal Principal, id SessionID, token []byte) error {
	return s.finish(ctx, principal, id, token)
}

func (s *SessionService) finish(ctx context.Context, principal Principal, id SessionID, token []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	record, err := s.options.Store.GetSession(ctx, id)
	if err != nil {
		return err
	}
	if record.Peer.ID != principal.ID {
		return ErrUnauthorized
	}
	hash := sha256.Sum256(token)
	if subtle.ConstantTimeCompare(hash[:], record.TokenHash[:]) != 1 {
		return ErrSessionToken
	}
	if err := s.options.Store.DeleteSession(ctx, id); err != nil {
		return err
	}
	s.clearIdempotency(id)
	var closeErr error
	if snapshot := s.takeSnapshot(id); snapshot != nil {
		closeErr = snapshot.Close()
	}
	s.notifyFinished(id)
	return closeErr
}

func (s *SessionService) Snapshot(id SessionID) (Snapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot, ok := s.snapshots[id]
	return snapshot, ok
}

func (s *SessionService) Session(ctx context.Context, principal Principal, id SessionID) (SessionRecord, error) {
	record, err := s.options.Store.GetSession(ctx, id)
	if err != nil {
		return SessionRecord{}, err
	}
	if record.Peer.ID != principal.ID {
		return SessionRecord{}, ErrUnauthorized
	}
	return record, nil
}

// RequireAction verifies that the authenticated principal is still authorized
// for a session operation. Authorization is intentionally rechecked per
// operation so revoking a scope does not leave an open session privileged.
func (s *SessionService) RequireAction(ctx context.Context, principal Principal, id SessionID, action Action) (SessionRecord, error) {
	record, err := s.Session(ctx, principal, id)
	if err != nil {
		return SessionRecord{}, err
	}
	authorized, err := s.options.Authorizer.Authorize(ctx, principal, action, record.Scope)
	if err != nil {
		return SessionRecord{}, err
	}
	if authorized.Scope != record.Scope {
		return SessionRecord{}, ErrUnauthorized
	}
	return record, nil
}

// Close releases all snapshots and process-local idempotency state owned by
// the service. It is safe to call more than once and should be invoked by the
// host during engine shutdown.
func (s *SessionService) Close() error {
	s.mu.Lock()
	snapshots := make([]Snapshot, 0, len(s.snapshots))
	ids := make([]SessionID, 0, len(s.snapshots))
	for id, snapshot := range s.snapshots {
		ids = append(ids, id)
		snapshots = append(snapshots, snapshot)
		delete(s.snapshots, id)
	}
	s.idempotent = make(map[string]sessionResult)
	s.mu.Unlock()
	var closeErr error
	for _, snapshot := range snapshots {
		closeErr = errors.Join(closeErr, snapshot.Close())
	}
	for _, id := range ids {
		s.notifyFinished(id)
	}
	return closeErr
}

func (s *SessionService) takeSnapshot(id SessionID) Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot := s.snapshots[id]
	delete(s.snapshots, id)
	return snapshot
}

func (s *SessionService) clearIdempotency(id SessionID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, result := range s.idempotent {
		if result.sessionID == id {
			delete(s.idempotent, key)
		}
	}
}

// RegisterFinishHook registers a process-local cleanup callback invoked after
// a session is completed or deleted. Hooks must be short and non-blocking.
func (s *SessionService) RegisterFinishHook(hook func(SessionID)) error {
	if hook == nil {
		return ErrInvalidEvent
	}
	s.mu.Lock()
	s.finishHooks = append(s.finishHooks, hook)
	s.mu.Unlock()
	return nil
}

func (s *SessionService) notifyFinished(id SessionID) {
	s.mu.Lock()
	hooks := append([]func(SessionID){}, s.finishHooks...)
	s.mu.Unlock()
	for _, hook := range hooks {
		hook(id)
	}
}
func cloneSessionResponse(response *relaysyncv2.OpenSessionResponse) *relaysyncv2.OpenSessionResponse {
	if response == nil {
		return nil
	}
	return proto.Clone(response).(*relaysyncv2.OpenSessionResponse)
}
