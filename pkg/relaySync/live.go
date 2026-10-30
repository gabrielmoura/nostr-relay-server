package relaysync

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
	"google.golang.org/protobuf/proto"
)

const (
	defaultLiveQueueEvents = 256
	defaultLiveQueueBytes  = 4 << 20
	defaultLiveBatchEvents = 64
	defaultLiveBatchBytes  = 1 << 20
)

// LiveState describes the transport-independent live replay lifecycle.
type LiveState uint8

const (
	LiveStateUnspecified LiveState = iota
	LiveStateNew
	LiveStateCatchingUp
	LiveStateLive
	LiveStatePaused
	LiveStateReconcileRequired
	LiveStateClosed
)

// ReconcileReason explains why replay cannot safely continue from its cursor.
type ReconcileReason string

const (
	ReconcileReasonGap          ReconcileReason = "GAP"
	ReconcileReasonEpochChanged ReconcileReason = "EPOCH_CHANGED"
	ReconcileReasonScopeChanged ReconcileReason = "SCOPE_CHANGED"
	ReconcileReasonSnapshotGone ReconcileReason = "SNAPSHOT_EXPIRED"
)

// LiveConfig limits retained journal changes and emitted batches. Heartbeat is
// adapter-driven: callers invoke HeartbeatDue and ObserveHeartbeat using their
// connection clock, so the core never starts a background goroutine.
type LiveConfig struct {
	QueueEvents       int
	QueueBytes        int
	BatchEvents       int
	BatchBytes        int
	HeartbeatInterval time.Duration
	HeartbeatTimeout  time.Duration
	Clock             Clock
}

func (c LiveConfig) withDefaults() LiveConfig {
	if c.QueueEvents == 0 {
		c.QueueEvents = defaultLiveQueueEvents
	}
	if c.QueueBytes == 0 {
		c.QueueBytes = defaultLiveQueueBytes
	}
	if c.BatchEvents == 0 {
		c.BatchEvents = defaultLiveBatchEvents
	}
	if c.BatchBytes == 0 {
		c.BatchBytes = defaultLiveBatchBytes
	}
	if c.Clock == nil {
		c.Clock = systemClock{}
	}
	return c
}

func (c LiveConfig) validate() error {
	if c.QueueEvents <= 0 || c.QueueBytes <= 0 || c.BatchEvents <= 0 || c.BatchBytes <= 0 {
		return fmt.Errorf("%w: %w", ErrInvalidLimits, &ValidationError{Field: "live", Reason: "queue and batch limits must be positive"})
	}
	if c.BatchEvents > c.QueueEvents || c.BatchBytes > c.QueueBytes {
		return fmt.Errorf("%w: %w", ErrInvalidLimits, &ValidationError{Field: "live", Reason: "batch limits must not exceed queue limits"})
	}
	if c.HeartbeatInterval < 0 || c.HeartbeatTimeout < 0 {
		return fmt.Errorf("%w: %w", ErrInvalidLimits, &ValidationError{Field: "live.heartbeat", Reason: "durations must not be negative"})
	}
	if c.HeartbeatTimeout > 0 && c.HeartbeatInterval == 0 {
		return fmt.Errorf("%w: %w", ErrInvalidLimits, &ValidationError{Field: "live.heartbeat", Reason: "timeout requires an interval"})
	}
	return nil
}

// FlowControl is the receiver's current send allowance. A zero allowance or
// Paused value stops emission without discarding buffered changes.
type FlowControl struct {
	MaxEvents int
	MaxBytes  int
	Paused    bool
}

// LiveBatch is an ordered, contiguous journal prefix emitted to a transport.
// Its changes remain retained until acknowledged or the connection is closed.
type LiveBatch struct {
	Changes []JournalChange
	First   Cursor
	Last    Cursor
	Bytes   int
}

// LiveHeartbeat is a transport-neutral heartbeat payload. ObservedCursor is
// informational and never advances a checkpoint.
type LiveHeartbeat struct {
	Nonce          uint64
	ObservedCursor Cursor
	SentAt         time.Time
}

// LiveSync coordinates bounded journal replay. It is safe for one producer
// calling Pump/NextBatch and one receiver calling Ack/flow-control only when
// those calls are externally serialized by the transport adapter.
//
// It intentionally owns no goroutine. A transport chooses when to call Pump,
// which keeps journal blocking, cancellation, and connection lifetime outside
// the reusable domain core.
type LiveSync struct {
	journal     JournalProvider
	checkpoints CheckpointStore
	key         CheckpointKey
	scope       Scope
	config      LiveConfig

	stream   Journal
	state    LiveState
	reason   ReconcileReason
	expected Cursor
	queue    []liveChange
	inflight []liveChange
	bytes    int
	flow     FlowControl

	lastSentHeartbeat time.Time
	lastPeerHeartbeat time.Time
	nextNonce         uint64
}

type liveChange struct {
	change JournalChange
	bytes  int
}

// NewLiveSync creates a coordinator. Resume opens the journal from the durable
// checkpoint; construction does not open storage or start background work.
func NewLiveSync(journal JournalProvider, checkpoints CheckpointStore, key CheckpointKey, scope Scope, config LiveConfig) (*LiveSync, error) {
	if journal == nil || checkpoints == nil {
		return nil, ErrNotFound
	}
	config = config.withDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}
	return &LiveSync{journal: journal, checkpoints: checkpoints, key: key, scope: scope, config: config, state: LiveStateNew}, nil
}

// Resume starts journal replay from the last durable checkpoint, or from the
// beginning when no checkpoint exists. It must be called before Pump.
func (s *LiveSync) Resume(ctx context.Context) error {
	checkpoint, ok, err := s.checkpoints.LoadCheckpoint(ctx, s.key)
	if err != nil {
		return err
	}
	after := Cursor{}
	if ok {
		after = checkpoint.Cursor
	}
	return s.open(ctx, after)
}

// Start starts journal replay after the supplied exclusive cursor. It is used
// when a caller has just completed HTTP reconciliation at a known watermark.
func (s *LiveSync) Start(ctx context.Context, after Cursor) error {
	return s.open(ctx, after)
}

func (s *LiveSync) open(ctx context.Context, after Cursor) error {
	if s.state != LiveStateNew && s.state != LiveStateClosed {
		return ErrConflict
	}
	stream, err := s.journal.OpenJournal(ctx, s.scope, after)
	if err != nil {
		return s.requireReconcileFor(err)
	}
	if after.Epoch != 0 && stream.Epoch() != after.Epoch {
		_ = stream.Close()
		return s.requireReconcile(ReconcileReasonEpochChanged)
	}
	s.stream = stream
	s.expected = after
	if s.expected.Epoch == 0 {
		s.expected.Epoch = stream.Epoch()
	}
	s.queue = []liveChange{}
	s.inflight = []liveChange{}
	s.bytes = 0
	s.reason = ""
	s.state = LiveStateCatchingUp
	now := s.config.Clock.Now()
	s.lastSentHeartbeat = now
	s.lastPeerHeartbeat = now
	return nil
}

// Pump reads at most limit changes into the bounded retained queue. A full
// queue is normal backpressure, not an error. An EOF means catch-up reached
// the current journal head and transitions the coordinator to Live.
func (s *LiveSync) Pump(ctx context.Context, limit int) (int, error) {
	if s.state == LiveStateReconcileRequired {
		return 0, ErrReconcileRequired
	}
	if s.state == LiveStateClosed || s.stream == nil {
		return 0, ErrConflict
	}
	if limit <= 0 {
		limit = s.config.QueueEvents
	}
	read := 0
	for read < limit && s.hasQueueCapacity() {
		change, err := s.stream.Next(ctx)
		if errors.Is(err, io.EOF) {
			if s.state == LiveStateCatchingUp {
				s.state = LiveStateLive
			}
			return read, nil
		}
		if err != nil {
			return read, s.requireReconcileFor(err)
		}
		if change.Cursor.Epoch != s.expected.Epoch || change.Cursor.Sequence != s.expected.Sequence+1 {
			return read, s.requireReconcile(ReconcileReasonGap)
		}
		size := journalChangeSize(change)
		if size > s.config.QueueBytes {
			return read, s.requireReconcile(ReconcileReasonGap)
		}
		if s.bytes+size > s.config.QueueBytes {
			return read, nil
		}
		s.queue = append(s.queue, liveChange{change: change, bytes: size})
		s.bytes += size
		s.expected = change.Cursor
		read++
	}
	return read, nil
}

func (s *LiveSync) hasQueueCapacity() bool {
	return len(s.queue)+len(s.inflight) < s.config.QueueEvents && s.bytes < s.config.QueueBytes
}

// SetFlowControl replaces the receiver's current send allowance. Credits are
// consumed by NextBatch and are deliberately not replenished by Ack.
func (s *LiveSync) SetFlowControl(flow FlowControl) error {
	if flow.MaxEvents < 0 || flow.MaxBytes < 0 {
		return fmt.Errorf("%w: %w", ErrInvalidLimits, &ValidationError{Field: "flow_control", Reason: "credits must not be negative"})
	}
	s.flow = flow
	if flow.Paused || flow.MaxEvents == 0 || flow.MaxBytes == 0 {
		if s.state == LiveStateCatchingUp || s.state == LiveStateLive {
			s.state = LiveStatePaused
		}
		return nil
	}
	if s.state == LiveStatePaused {
		s.state = LiveStateLive
	}
	return nil
}

// NextBatch returns the next ordered batch within both flow-control and batch
// bounds. Returned changes remain in-flight until Ack durably advances the
// checkpoint.
func (s *LiveSync) NextBatch() (LiveBatch, error) {
	if s.state == LiveStateReconcileRequired {
		return LiveBatch{}, ErrReconcileRequired
	}
	if s.state == LiveStateClosed {
		return LiveBatch{}, ErrConflict
	}
	if s.flow.Paused || s.flow.MaxEvents == 0 || s.flow.MaxBytes == 0 {
		return LiveBatch{}, ErrFlowControl
	}
	if len(s.queue) == 0 {
		return LiveBatch{}, io.EOF
	}
	maxEvents := min(s.config.BatchEvents, s.flow.MaxEvents)
	maxBytes := min(s.config.BatchBytes, s.flow.MaxBytes)
	batch := LiveBatch{Changes: make([]JournalChange, 0, maxEvents)}
	count := 0
	for count < len(s.queue) && count < maxEvents {
		item := s.queue[count]
		if count > 0 && batch.Bytes+item.bytes > maxBytes {
			break
		}
		if count == 0 && item.bytes > maxBytes {
			return LiveBatch{}, ErrFlowControl
		}
		batch.Changes = append(batch.Changes, item.change)
		batch.Bytes += item.bytes
		count++
	}
	if count == 0 {
		return LiveBatch{}, ErrFlowControl
	}
	batch.First = batch.Changes[0].Cursor
	batch.Last = batch.Changes[len(batch.Changes)-1].Cursor
	s.inflight = append(s.inflight, s.queue[:count]...)
	s.queue = append([]liveChange{}, s.queue[count:]...)
	s.flow.MaxEvents -= count
	s.flow.MaxBytes -= batch.Bytes
	return batch, nil
}

// Ack accepts the highest contiguous cursor durably processed by the peer. It
// persists the checkpoint before releasing retained changes, so a crash before
// this method returns results in safe replay from the prior checkpoint.
func (s *LiveSync) Ack(ctx context.Context, cursor Cursor, decisionID string) error {
	if s.state == LiveStateReconcileRequired {
		return ErrReconcileRequired
	}
	if len(s.inflight) == 0 {
		return ErrConflict
	}
	if cursor.Epoch != s.inflight[0].change.Cursor.Epoch {
		return ErrConflict
	}
	count := 0
	for count < len(s.inflight) && s.inflight[count].change.Cursor.Sequence <= cursor.Sequence {
		count++
	}
	if count == 0 || s.inflight[count-1].change.Cursor != cursor {
		return ErrConflict
	}
	if err := s.checkpoints.AdvanceCheckpoint(ctx, Checkpoint{Key: s.key, Cursor: cursor, DecisionID: decisionID}); err != nil {
		return err
	}
	for _, item := range s.inflight[:count] {
		s.bytes -= item.bytes
	}
	s.inflight = append([]liveChange{}, s.inflight[count:]...)
	return nil
}

// HeartbeatDue returns a heartbeat when the interval elapsed. It returns a
// timeout error and closes only the live connection state when peer liveness is
// lost; durable sessions and checkpoints are untouched.
func (s *LiveSync) HeartbeatDue(now time.Time) (LiveHeartbeat, bool, error) {
	if s.state == LiveStateClosed || s.state == LiveStateReconcileRequired {
		return LiveHeartbeat{}, false, ErrConflict
	}
	if s.config.HeartbeatTimeout > 0 && now.Sub(s.lastPeerHeartbeat) > s.config.HeartbeatTimeout {
		_ = s.Close()
		return LiveHeartbeat{}, false, ErrHeartbeatTimeout
	}
	if s.config.HeartbeatInterval == 0 || now.Sub(s.lastSentHeartbeat) < s.config.HeartbeatInterval {
		return LiveHeartbeat{}, false, nil
	}
	s.nextNonce++
	s.lastSentHeartbeat = now
	return LiveHeartbeat{Nonce: s.nextNonce, ObservedCursor: s.expected, SentAt: now}, true, nil
}

// ObserveHeartbeat records liveness only. It intentionally does not imply ACK.
func (s *LiveSync) ObserveHeartbeat(now time.Time) {
	s.lastPeerHeartbeat = now
}

// State returns the current coordinator state and any terminal reconcile reason.
func (s *LiveSync) State() (LiveState, ReconcileReason) { return s.state, s.reason }

// Close releases the journal stream. It is idempotent and never changes a
// checkpoint; callers reconnect through a new coordinator and Resume.
func (s *LiveSync) Close() error {
	if s.state == LiveStateClosed {
		return nil
	}
	s.state = LiveStateClosed
	if s.stream == nil {
		return nil
	}
	err := s.stream.Close()
	s.stream = nil
	return err
}

func (s *LiveSync) requireReconcileFor(err error) error {
	switch {
	case errors.Is(err, ErrJournalEpochChanged):
		return s.requireReconcile(ReconcileReasonEpochChanged)
	case errors.Is(err, ErrJournalGap):
		return s.requireReconcile(ReconcileReasonGap)
	default:
		return err
	}
}

func (s *LiveSync) requireReconcile(reason ReconcileReason) error {
	s.reason = reason
	s.state = LiveStateReconcileRequired
	if s.stream != nil {
		_ = s.stream.Close()
		s.stream = nil
	}
	return ErrReconcileRequired
}

func journalChangeSize(change JournalChange) int {
	message := &relaysyncv2.JournalChange{
		Cursor:     &relaysyncv2.Cursor{Epoch: change.Cursor.Epoch, Sequence: change.Cursor.Sequence},
		ChangeType: uint32(change.Type),
		EventId:    append([]byte(nil), change.ID[:]...),
	}
	if change.Event != nil {
		message.Event = EventToProto(*change.Event)
	}
	return proto.Size(message)
}
