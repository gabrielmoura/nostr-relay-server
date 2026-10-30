package cordn

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrInvalidGroupID = errors.New("cordn: group id is required")
	ErrInvalidPayload = errors.New("cordn: payload is required")
)

// Store is the durable coordinator boundary. It is deliberately separate from
// Nostr events and from RelaySync because Cordn owns its group-scoped cursors.
type Store interface {
	InsertCordnKeyPackage(context.Context, KeyPackage) error
	ListCordnKeyPackages(context.Context, string) ([]KeyPackage, error)
	ListAllCordnKeyPackages(context.Context) ([]KeyPackage, error)
	TakeCordnKeyPackage(context.Context, string) (KeyPackage, bool, error)
	DeleteCordnKeyPackage(context.Context, string) (KeyPackage, bool, error)
	InsertCordnWelcome(context.Context, Welcome) error
	ListCordnWelcomes(context.Context, string) ([]Welcome, error)
	DeleteCordnWelcome(context.Context, string, string, time.Time) error
	InsertCordnJoinRequest(context.Context, JoinRequest) error
	ListCordnJoinRequests(context.Context, string) ([]JoinRequest, error)
	DeleteCordnJoinRequest(context.Context, JoinRequest) error
	DeleteExpiredCordnRecords(context.Context, time.Time) error
	AppendCordnGroupMessage(context.Context, string, []byte, time.Time) (GroupMessage, error)
	ListCordnGroupMessages(context.Context, string, int64) ([]GroupMessage, error)
}

// CleanupExpired removes only queued delivery records and opaque group
// messages. KeyPackages remain until consumed or explicitly removed, matching
// their publication lifecycle.
func (s *Service) CleanupExpired(ctx context.Context, before time.Time) error {
	if err := s.store.DeleteExpiredCordnRecords(ctx, before.UTC()); err != nil {
		return fmt.Errorf("delete expired Cordn records: %w", err)
	}
	return nil
}

// Service coordinates durable Cordn records and in-process live fan-out.
// A stream is registered before backlog replay; callers deduplicate the rare
// overlap at the boundary using the stable (gid, cursor) pair.
type Service struct {
	store Store
	now   func() time.Time

	mu           sync.Mutex
	nextStream   uint64
	streamsByID  map[uint64]stream
	streamsByGID map[string]map[uint64]chan GroupMessage
}

type stream struct {
	groupID string
	output  chan GroupMessage
}

func NewService(store Store) *Service {
	return &Service{
		store:        store,
		now:          time.Now,
		streamsByID:  map[uint64]stream{},
		streamsByGID: map[string]map[uint64]chan GroupMessage{},
	}
}

func (s *Service) PublishGroupMessage(ctx context.Context, groupID string, payload []byte) (GroupMessage, error) {
	if groupID == "" {
		return GroupMessage{}, ErrInvalidGroupID
	}
	if len(payload) == 0 {
		return GroupMessage{}, ErrInvalidPayload
	}

	message, err := s.store.AppendCordnGroupMessage(ctx, groupID, payload, s.now().UTC())
	if err != nil {
		return GroupMessage{}, fmt.Errorf("persist Cordn group message: %w", err)
	}
	s.publishLive(message)
	return message, nil
}

func (s *Service) FetchGroupMessages(ctx context.Context, groupID string, afterCursor int64) ([]GroupMessage, error) {
	if groupID == "" {
		return nil, ErrInvalidGroupID
	}
	if afterCursor < 0 {
		return nil, fmt.Errorf("cordn: after cursor must not be negative")
	}
	messages, err := s.store.ListCordnGroupMessages(ctx, groupID, afterCursor)
	if err != nil {
		return nil, fmt.Errorf("fetch Cordn group messages: %w", err)
	}
	return messages, nil
}

func (s *Service) PublishKeyPackage(ctx context.Context, value KeyPackage) error {
	if value.Reference == "" || value.StablePubkey == "" || len(value.Payload) == 0 {
		return errors.New("cordn: incomplete key package")
	}
	return s.store.InsertCordnKeyPackage(ctx, value)
}

func (s *Service) ListKeyPackages(ctx context.Context) ([]KeyPackage, error) {
	return s.store.ListAllCordnKeyPackages(ctx)
}

func (s *Service) TakeKeyPackage(ctx context.Context, reference string) (KeyPackage, bool, error) {
	if reference == "" {
		return KeyPackage{}, false, errors.New("cordn: key package reference is required")
	}
	value, found, err := s.GetKeyPackage(ctx, reference)
	if err != nil {
		return KeyPackage{}, false, err
	}
	if !found {
		packages, listErr := s.store.ListCordnKeyPackages(ctx, reference)
		if listErr != nil || len(packages) == 0 {
			return KeyPackage{}, false, listErr
		}
		value = packages[0]
		for _, candidate := range packages {
			if !candidate.LastResort {
				value = candidate
				break
			}
		}
		found = true
	}
	if value.LastResort {
		return value, found, err
	}
	return s.store.TakeCordnKeyPackage(ctx, value.Reference)
}

func (s *Service) GetKeyPackage(ctx context.Context, reference string) (KeyPackage, bool, error) {
	if reference == "" {
		return KeyPackage{}, false, errors.New("cordn: key package reference is required")
	}
	packages, err := s.store.ListAllCordnKeyPackages(ctx)
	if err != nil {
		return KeyPackage{}, false, err
	}
	for _, value := range packages {
		if value.Reference == reference {
			return value, true, nil
		}
	}
	return KeyPackage{}, false, nil
}

func (s *Service) RemoveKeyPackage(ctx context.Context, reference, stablePubkey string) (bool, error) {
	packages, err := s.store.ListCordnKeyPackages(ctx, stablePubkey)
	if err != nil {
		return false, err
	}
	for _, value := range packages {
		if value.Reference == reference {
			_, _, err := s.store.DeleteCordnKeyPackage(ctx, reference)
			return err == nil, err
		}
	}
	return false, nil
}

func (s *Service) StoreWelcome(ctx context.Context, value Welcome) error {
	if value.TargetPubkey == "" || value.KeyPackageRef == "" || len(value.Payload) == 0 {
		return errors.New("cordn: incomplete welcome")
	}
	return s.store.InsertCordnWelcome(ctx, value)
}

func (s *Service) TakeWelcomes(ctx context.Context, pubkey string, consumed []Welcome) ([]Welcome, error) {
	for _, value := range consumed {
		if err := s.store.DeleteCordnWelcome(ctx, pubkey, value.KeyPackageRef, value.CreatedAt); err != nil {
			return nil, err
		}
	}
	return s.store.ListCordnWelcomes(ctx, pubkey)
}

func (s *Service) StoreJoinRequest(ctx context.Context, value JoinRequest) error {
	if value.GroupID == "" || value.RequesterPubkey == "" || value.KeyPackageRef == "" {
		return errors.New("cordn: incomplete join request")
	}
	return s.store.InsertCordnJoinRequest(ctx, value)
}

func (s *Service) ListJoinRequests(ctx context.Context, groupID string) ([]JoinRequest, error) {
	if groupID == "" {
		return nil, ErrInvalidGroupID
	}
	return s.store.ListCordnJoinRequests(ctx, groupID)
}

func (s *Service) TakeJoinRequests(ctx context.Context, groupID string, consumed []JoinRequest) ([]JoinRequest, error) {
	for _, value := range consumed {
		if value.GroupID != groupID {
			continue
		}
		if err := s.store.DeleteCordnJoinRequest(ctx, value); err != nil {
			return nil, err
		}
	}
	return s.store.ListCordnJoinRequests(ctx, groupID)
}

// SubscribeGroupMessages registers live delivery before fetching the durable
// backlog. The consumer must process both channels through one deduplicating
// pipeline keyed by (gid, cursor).
func (s *Service) SubscribeGroupMessages(ctx context.Context, groupID string, afterCursor int64) ([]GroupMessage, <-chan GroupMessage, func(), error) {
	if groupID == "" {
		return nil, nil, nil, ErrInvalidGroupID
	}
	if afterCursor < 0 {
		return nil, nil, nil, fmt.Errorf("cordn: after cursor must not be negative")
	}

	streamID, live, cancel := s.registerStream(groupID)
	backlog, err := s.FetchGroupMessages(ctx, groupID, afterCursor)
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}
	return backlog, live, func() {
		_ = streamID
		cancel()
	}, nil
}

func (s *Service) registerStream(groupID string) (uint64, <-chan GroupMessage, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextStream++
	streamID := s.nextStream
	output := make(chan GroupMessage, 128)
	if s.streamsByGID[groupID] == nil {
		s.streamsByGID[groupID] = map[uint64]chan GroupMessage{}
	}
	s.streamsByGID[groupID][streamID] = output
	s.streamsByID[streamID] = stream{groupID: groupID, output: output}

	var once sync.Once
	cancel := func() {
		once.Do(func() { s.unregisterStream(streamID) })
	}
	return streamID, output, cancel
}

func (s *Service) unregisterStream(streamID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	registered, ok := s.streamsByID[streamID]
	if !ok {
		return
	}
	delete(s.streamsByID, streamID)
	delete(s.streamsByGID[registered.groupID], streamID)
	if len(s.streamsByGID[registered.groupID]) == 0 {
		delete(s.streamsByGID, registered.groupID)
	}
	close(registered.output)
}

func (s *Service) publishLive(message GroupMessage) {
	s.mu.Lock()
	stalled := make([]uint64, 0)
	for streamID, output := range s.streamsByGID[message.GroupID] {
		select {
		case output <- message:
		default:
			// Never silently lose a cursor: close a stalled live stream so the
			// client resumes from its last durable cursor.
			stalled = append(stalled, streamID)
		}
	}
	s.mu.Unlock()
	for _, streamID := range stalled {
		s.unregisterStream(streamID)
	}
}
