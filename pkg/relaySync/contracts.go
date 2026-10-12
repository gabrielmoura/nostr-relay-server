package relaysync

import (
	"context"
	"time"
)

// SnapshotVersion identifies an immutable snapshot view. Callers must treat it
// as opaque.
type SnapshotVersion []byte

// Cursor identifies a journal position within an epoch.
type Cursor struct {
	Epoch    uint64
	Sequence uint64
}

// Scope identifies the canonically negotiated visibility policy.
type Scope struct {
	Name    string
	Version uint32
	Hash    [32]byte
}

// EventMetadata is the minimal event reference used during reconciliation.
type EventMetadata struct {
	ID        EventID
	CreatedAt int64
}

// MetadataCursor is exclusive and preserves the total (created_at, event_id)
// ordering required by a snapshot.
type MetadataCursor = EventMetadata

// PartitionKey identifies a snapshot partition. The partition algorithm version
// makes the key safe to evolve.
type PartitionKey struct {
	AlgorithmVersion uint32
	Bucket           int64
	PrefixBits       uint8
	Prefix           [32]byte
}

// SnapshotDescriptor describes a stable snapshot and its journal watermark.
type SnapshotDescriptor struct {
	Version    SnapshotVersion
	Scope      Scope
	Watermark  Cursor
	ExpiresAt  time.Time
	Partitions []PartitionKey
}

// EventItem preserves requested event IDs even when an event is no longer
// visible in the snapshot.
type EventItem struct {
	ID    EventID
	Event *Event
}

// SnapshotProvider opens an immutable, bounded-lifetime read view.
type SnapshotProvider interface {
	OpenSnapshot(ctx context.Context, scope Scope) (Snapshot, error)
}

// Snapshot exposes one stable event view. Close must be idempotent.
type Snapshot interface {
	Descriptor() SnapshotDescriptor
	OpenMetadata(ctx context.Context, partition PartitionKey, after *MetadataCursor) (MetadataIterator, error)
	GetEvents(ctx context.Context, ids []EventID) (EventIterator, error)
	Close() error
}

// MetadataIterator is ordered by (created_at, event_id).
type MetadataIterator interface {
	Next(ctx context.Context) (EventMetadata, bool, error)
	Close() error
}

// EventIterator returns an EventItem for every requested ID, including absent
// events whose Event field is nil.
type EventIterator interface {
	Next(ctx context.Context) (EventItem, bool, error)
	Close() error
}

// EventBatch binds a retry-safe operation ID to an ordered event batch.
type EventBatch struct {
	OperationID string
	Events      []Event
}

// WriteStatus records a durable per-event write outcome.
type WriteStatus uint8

const (
	WriteStatusUnspecified WriteStatus = iota
	WriteAccepted
	WriteAlreadyExists
	WriteRejectedPermanent
	WriteFailedTemporary
)

// WriteResult corresponds to the event at the same input index.
type WriteResult struct {
	ID            EventID
	Status        WriteStatus
	Code          string
	Message       string
	PolicyVersion uint32
	DecisionID    string
}

// EventWriter durably writes an ordered batch. A returned error means the
// batch outcome is unknown and callers must retry with the same OperationID.
type EventWriter interface {
	WriteEvents(ctx context.Context, batch EventBatch) ([]WriteResult, error)
}

// ChangeType describes an observable visibility transition.
type ChangeType uint8

const (
	ChangeTypeUnspecified ChangeType = iota
	ChangeUpsert
	ChangeDelete
)

// JournalChange is one commit-ordered visibility change.
type JournalChange struct {
	Cursor   Cursor
	Type     ChangeType
	ID       EventID
	Metadata EventMetadata
	Event    *Event
}

// JournalProvider opens a commit-ordered journal after an exclusive cursor.
type JournalProvider interface {
	OpenJournal(ctx context.Context, scope Scope, after Cursor) (Journal, error)
}

// Journal is a sequential, closeable stream of changes.
type Journal interface {
	Epoch() uint64
	Next(ctx context.Context) (JournalChange, error)
	Close() error
}

// Principal is the stable authenticated peer identity. It is never an IP
// address.
type Principal struct {
	ID string
}

// Action is a separately authorized RelaySync capability.
type Action uint8

const (
	ActionUnspecified Action = iota
	ActionOpenSession
	ActionPull
	ActionPush
	ActionReadChanges
	ActionLive
)

// AuthorizedScope may be narrower than a peer's requested scope.
type AuthorizedScope struct {
	Scope   Scope
	Actions []Action
}

// PeerAuthorizer authorizes a principal for one requested capability.
type PeerAuthorizer interface {
	Authorize(ctx context.Context, principal Principal, action Action, requested Scope) (AuthorizedScope, error)
}

// CheckpointKey identifies one peer and scope checkpoint.
type CheckpointKey struct {
	PeerID string
	Scope  Scope
}

// Checkpoint is advanced only after a durable terminal write decision.
type Checkpoint struct {
	Key        CheckpointKey
	Cursor     Cursor
	DecisionID string
}

// CheckpointStore atomically advances checkpoints within one epoch.
type CheckpointStore interface {
	LoadCheckpoint(ctx context.Context, key CheckpointKey) (Checkpoint, bool, error)
	AdvanceCheckpoint(ctx context.Context, next Checkpoint) error
	ResetCheckpoint(ctx context.Context, next Checkpoint) error
}

// SessionID is a binary session identifier.
type SessionID [16]byte

// SessionState is the serialized session lifecycle state.
type SessionState uint8

const (
	SessionStateUnspecified SessionState = iota
	SessionStateCreated
	SessionStateSnapshotOpen
	SessionStateReconciling
	SessionStateTransferring
	SessionStateCatchUp
	SessionStateLive
	SessionStateCompleted
	SessionStateExpired
	SessionStateCancelled
	SessionStateFailed
)

// SessionRecord is a value object. Stores must return independent copies.
type SessionRecord struct {
	ID              SessionID
	TokenHash       [32]byte
	Peer            Principal
	Scope           Scope
	SnapshotVersion SnapshotVersion
	State           SessionState
	Watermark       Cursor
	ExpiresAt       time.Time
	Version         uint64
}

// SessionUpdate performs a compare-and-swap update. Only the lifecycle state,
// watermark and expiry are mutable after creation.
type SessionUpdate struct {
	ID              SessionID
	ExpectedVersion uint64
	State           SessionState
	Watermark       Cursor
	ExpiresAt       time.Time
}

type SessionCreator interface {
	CreateSession(ctx context.Context, session SessionRecord) error
}
type SessionReader interface {
	GetSession(ctx context.Context, id SessionID) (SessionRecord, error)
}
type SessionUpdater interface {
	UpdateSession(ctx context.Context, update SessionUpdate) error
}
type SessionCloser interface {
	DeleteSession(ctx context.Context, id SessionID) error
}

// SessionStore is the complete durable session contract.
type SessionStore interface {
	SessionCreator
	SessionReader
	SessionUpdater
	SessionCloser
}
