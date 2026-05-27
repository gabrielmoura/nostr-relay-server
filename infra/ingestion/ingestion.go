package ingestion

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	dbstore "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/infra/pubsub"
	"github.com/gabrielmoura/nostr-relay-server/infra/stream"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/groups"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	policies "github.com/gabrielmoura/nostr-relay-server/internal/policies"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

type EventBatch struct {
	Events   []*nostr.Event
	Received time.Time
}

type IngestionConfig struct {
	BatchSize    int
	BatchTimeout time.Duration
	Workers      int
	QueueSize    int
}

var (
	cfg        IngestionConfig
	eventsChan chan *nostr.Event
	workers    []*worker
	started    bool
	startMutex sync.Mutex

	statsBatchProcessed   atomic.Int64
	statsEventsInserted   atomic.Int64
	statsDuplicates       atomic.Int64
	statsErrors           atomic.Int64
	queryFirstStoredEvent = firstEvent
	deleteStoredEventByID = func(ctx context.Context, id, deletedBy string) error {
		return db.DbQueries.DeleteEvent(ctx, id, deletedBy)
	}
)

type worker struct {
	id    int
	batch []*nostr.Event
	timer *time.Timer
	mu    sync.Mutex
}

func Init() {
	cfg = IngestionConfig{
		BatchSize:    1000,
		BatchTimeout: 100 * time.Millisecond,
		Workers:      4,
		QueueSize:    10000,
	}

	if config.Cfg.Ingestion.BatchSize > 0 {
		cfg.BatchSize = config.Cfg.Ingestion.BatchSize
	}
	if config.Cfg.Ingestion.BatchTimeoutMs > 0 {
		cfg.BatchTimeout = time.Duration(config.Cfg.Ingestion.BatchTimeoutMs) * time.Millisecond
	}
	if config.Cfg.Ingestion.Workers > 0 {
		cfg.Workers = config.Cfg.Ingestion.Workers
	}
	if config.Cfg.Ingestion.QueueSize > 0 {
		cfg.QueueSize = config.Cfg.Ingestion.QueueSize
	}

	eventsChan = make(chan *nostr.Event, cfg.QueueSize)

	log.Logger.Info("ingestion initialized",
		zap.Int("batch_size", cfg.BatchSize),
		zap.Duration("batch_timeout", cfg.BatchTimeout),
		zap.Int("workers", cfg.Workers),
		zap.Int("queue_size", cfg.QueueSize),
	)
}

func Start(ctx context.Context) {
	startMutex.Lock()
	defer startMutex.Unlock()

	if started {
		return
	}
	started = true

	workers = make([]*worker, cfg.Workers)
	for i := 0; i < cfg.Workers; i++ {
		w := &worker{
			id:    i,
			timer: time.NewTimer(cfg.BatchTimeout),
		}
		workers[i] = w
		go w.run(ctx, i)
	}

	log.Logger.Info("ingestion workers started", zap.Int("count", cfg.Workers))
}

func Stop() {
	startMutex.Lock()
	defer startMutex.Unlock()

	if !started {
		return
	}
	started = false

	close(eventsChan)
	for _, w := range workers {
		if w != nil && w.timer != nil {
			w.timer.Stop()
		}
	}

	log.Logger.Info("ingestion stopped")
}

func Push(event *nostr.Event) bool {
	if !started {
		return false
	}

	select {
	case eventsChan <- event:
		return true
	default:
		metrics.NostrRelayIngestionBackpressure.Inc()
		return false
	}
}

func (w *worker) run(ctx context.Context, workerID int) {
	for {
		select {
		case event, ok := <-eventsChan:
			if !ok {
				w.flush(ctx)
				return
			}

			if w.isDuplicate(event) {
				statsDuplicates.Add(1)
				metrics.NostrRelayIngestionDuplicates.Inc()
				continue
			}

			w.batch = append(w.batch, event)

			if len(w.batch) >= cfg.BatchSize {
				w.flush(ctx)
				w.resetTimer()
			}

		case <-w.timer.C:
			w.flush(ctx)
			w.resetTimer()

		case <-ctx.Done():
			w.flush(ctx)
			return
		}
	}
}

func (w *worker) flush(ctx context.Context) {
	w.mu.Lock()
	batch := w.batch
	w.batch = nil
	w.mu.Unlock()

	if len(batch) == 0 {
		return
	}

	startTime := time.Now()
	err := insertBatch(ctx, batch)
	duration := time.Since(startTime)

	if err != nil {
		statsErrors.Add(1)
		metrics.NostrRelayIngestionErrors.Inc()
		log.Logger.Error("batch insert failed",
			zap.Error(err),
			zap.Int("batch_size", len(batch)),
			zap.Duration("duration", duration),
		)
	} else {
		statsBatchProcessed.Add(1)
		statsEventsInserted.Add(int64(len(batch)))
		metrics.NostrRelayBatchProcessed.Inc()
		metrics.NostrRelayEventsInserted.Add(float64(len(batch)))
		metrics.NostrRelayIngestionDuration.Observe(duration.Seconds())

		log.Logger.Debug("batch inserted",
			zap.Int("count", len(batch)),
			zap.Duration("duration", duration),
		)
	}
}

func (w *worker) resetTimer() {
	if w.timer == nil {
		w.timer = time.NewTimer(cfg.BatchTimeout)
	} else {
		w.timer.Reset(cfg.BatchTimeout)
	}
}

func (w *worker) isDuplicate(event *nostr.Event) bool {
	isDup, err := cache.SetDedup(event.ID)
	return err == nil && isDup
}

func insertBatch(ctx context.Context, events []*nostr.Event) error {
	if len(events) == 0 {
		return nil
	}

	accepted := make([]*nostr.Event, 0, len(events))
	stored := make([]*nostr.Event, 0, len(events))

	for _, evt := range events {
		reject, reason := policies.P.ValidateBatchEvent(ctx, evt)
		if reject {
			statsErrors.Add(1)
			log.Logger.Debug("ingestion policy rejected event", zap.String("event_id", evt.ID), zap.String("reason", reason))
			continue
		}

		if err := prepareEventForStorage(ctx, evt); err != nil {
			if errors.Is(err, dbstore.ErrDupEvent) {
				statsDuplicates.Add(1)
				metrics.NostrRelayEventDuplicateRejections.Inc()
				continue
			}
			return err
		}

		accepted = append(accepted, evt)
		if !nostr.IsEphemeralKind(evt.Kind) {
			stored = append(stored, evt)
		}
	}

	if len(stored) > 0 {
		if err := db.DbQueries.InsertEventBatch(ctx, stored); err != nil {
			return err
		}
	}

	for _, evt := range accepted {
		listener.NotifyListeners(evt)
		stream.ForwardEvent(*evt)
		if ps := pubsub.GetPubSub(); ps != nil && ps.IsEnabled() {
			_ = ps.PublishEvent(ctx, evt)
		}

		serialized, _ := json.Marshal(evt)
		_ = cache.SetEvent(evt.ID, string(serialized))
		metrics.NostrKindEventCounter.WithLabelValues(metrics.GetKindName(evt.Kind)).Inc()
		metrics.NostrUserEventCounter.WithLabelValues(evt.PubKey).Inc()
		if err := groups.AfterStoreEvent(ctx, evt); err != nil {
			log.Logger.Warn("nip29 post-persist handling failed",
				zap.String("event_id", evt.ID),
				zap.Int("kind", evt.Kind),
				zap.Error(err),
			)
		}
	}

	return nil
}

func prepareEventForStorage(ctx context.Context, evt *nostr.Event) error {
	if nostr.IsEphemeralKind(evt.Kind) {
		return nil
	}

	if nostr.IsReplaceableKind(evt.Kind) {
		previous, err := queryFirstStoredEvent(ctx, nostr.Filter{Authors: []string{evt.PubKey}, Kinds: []int{evt.Kind}})
		if err != nil {
			return err
		}
		if previous != nil && isOlder(previous, evt) {
			if err := deleteStoredEventByID(ctx, previous.ID, evt.ID); err != nil {
				return err
			}
		}
	}

	if nostr.IsAddressableKind(evt.Kind) {
		d := evt.Tags.GetFirst([]string{"d", ""})
		if d != nil {
			previous, err := queryFirstStoredEvent(ctx, nostr.Filter{Authors: []string{evt.PubKey}, Kinds: []int{evt.Kind}, Tags: nostr.TagMap{"d": []string{d.Value()}}})
			if err != nil {
				return err
			}
			if previous != nil && isOlder(previous, evt) {
				if err := deleteStoredEventByID(ctx, previous.ID, evt.ID); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func firstEvent(ctx context.Context, filter nostr.Filter) (*nostr.Event, error) {
	ch, err := db.DbQueries.QueryEventsChan(ctx, filter)
	if err != nil {
		return nil, err
	}
	for evt := range ch {
		return evt, nil
	}
	return nil, nil
}

func isOlder(previous, next *nostr.Event) bool {
	return previous.CreatedAt < next.CreatedAt || (previous.CreatedAt == next.CreatedAt && previous.ID > next.ID)
}

func GetStats() IngestionStats {
	return IngestionStats{
		BatchProcessed: statsBatchProcessed.Load(),
		EventsInserted: statsEventsInserted.Load(),
		Duplicates:     statsDuplicates.Load(),
		Errors:         statsErrors.Load(),
		QueueDepth:     len(eventsChan),
	}
}

type IngestionStats struct {
	BatchProcessed int64
	EventsInserted int64
	Duplicates     int64
	Errors         int64
	QueueDepth     int
}
