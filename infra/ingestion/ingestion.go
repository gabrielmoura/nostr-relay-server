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
	"github.com/gabrielmoura/nostr-relay-server/internal/dmimport"
	"github.com/gabrielmoura/nostr-relay-server/internal/groups"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	policies "github.com/gabrielmoura/nostr-relay-server/internal/policies"
	"github.com/gabrielmoura/nostr-relay-server/internal/profile"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

type EventBatch struct {
	Events   []*nostr.Event
	Received time.Time
}

type batchResult struct {
	Inserted   int
	Duplicates int
	Rejected   int
	Uncertain  int
}

type Receipt struct {
	Inserted  bool
	Duplicate bool
	Uncertain bool
	Reason    string
}
type queueItem struct {
	event     *nostr.Event
	dmImport  bool
	receipt   chan Receipt
	admission dmimport.Admission
}

type IngestionConfig struct {
	BatchSize    int
	BatchTimeout time.Duration
	Workers      int
	QueueSize    int
}

var (
	cfg        IngestionConfig
	eventsChan chan queueItem
	workers    []*worker
	started    bool
	startMutex sync.Mutex
	stopping   chan struct{}
	pushes     sync.WaitGroup
	workerWG   sync.WaitGroup

	statsBatchProcessed   atomic.Int64
	statsEventsInserted   atomic.Int64
	statsDuplicates       atomic.Int64
	statsErrors           atomic.Int64
	queryFirstStoredEvent = firstEvent
	deleteStoredEventByID = func(ctx context.Context, id, deletedBy string) error {
		return db.DbQueries.DeleteEvent(ctx, id, deletedBy)
	}
	upsertImportedProfile = profile.Upsert
)

type worker struct {
	id    int
	batch []queueItem
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

	eventsChan = make(chan queueItem, cfg.QueueSize)
	stopping = make(chan struct{})
	metrics.NostrRelayIngestionQueueDepth.Set(0)

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
	// Shutdown cancels the application context before calling Stop. Workers use
	// a cancellation-free child so Stop can close and drain the bounded queue
	// before database/pubsub dependencies are torn down.
	workerCtx := context.WithoutCancel(ctx)
	for i := 0; i < cfg.Workers; i++ {
		w := &worker{
			id:    i,
			timer: time.NewTimer(cfg.BatchTimeout),
		}
		workers[i] = w
		workerWG.Add(1)
		go func(worker *worker, workerID int) {
			defer workerWG.Done()
			worker.run(workerCtx, workerID)
		}(w, i)
	}

	log.Logger.Info("ingestion workers started", zap.Int("count", cfg.Workers))
}

func Stop() {
	startMutex.Lock()
	if !started {
		startMutex.Unlock()
		return
	}
	started = false
	close(stopping)
	startMutex.Unlock()

	// No sender can race a channel close: every accepted sender is registered
	// while started is protected by startMutex and observes the stop signal.
	pushes.Wait()

	close(eventsChan)
	for _, w := range workers {
		if w != nil && w.timer != nil {
			w.timer.Stop()
		}
	}
	workerWG.Wait()
	metrics.NostrRelayIngestionQueueDepth.Set(0)

	log.Logger.Info("ingestion stopped")
}

func Push(event *nostr.Event) bool {
	queue, stop, ok := beginPush()
	if !ok {
		return false
	}
	defer pushes.Done()

	select {
	case queue <- queueItem{event: event}:
		metrics.NostrRelayIngestionQueueDepth.Set(float64(len(queue)))
		return true
	case <-stop:
		return false
	default:
		metrics.NostrRelayIngestionBackpressure.Inc()
		metrics.NostrRelayIngestionQueueDepth.Set(float64(len(queue)))
		return false
	}
}

// PushDMImport waits for the database outcome so browser clients can report
// durable insertions separately from idempotent duplicates.
func PushDMImport(ctx context.Context, event *nostr.Event, admission dmimport.Admission) (Receipt, bool) {
	if !admission.ValidFor(event) {
		return Receipt{Reason: "restricted: invalid DM import admission"}, false
	}
	if ctx.Err() != nil {
		return Receipt{Reason: "error: import queue wait timed out"}, false
	}
	queue, stop, ok := beginPush()
	if !ok {
		return Receipt{Reason: "error: ingestion is not running"}, false
	}
	receipt := make(chan Receipt, 1)
	item := queueItem{event: event, dmImport: true, receipt: receipt, admission: admission}

	// A full queue is a backpressure signal, not an automatic rejection. The
	// bounded wait below is still governed by the caller's request deadline.
	select {
	case queue <- item:
		metrics.NostrRelayIngestionQueueDepth.Set(float64(len(queue)))
		pushes.Done()
		return waitDMImportReceipt(ctx, receipt)
	case <-stop:
		pushes.Done()
		return Receipt{Reason: "error: ingestion is stopping"}, false
	default:
		metrics.NostrRelayIngestionBackpressure.Inc()
		metrics.NostrRelayIngestionQueueDepth.Set(float64(len(queue)))
	}

	select {
	case queue <- item:
		metrics.NostrRelayIngestionQueueDepth.Set(float64(len(queue)))
		pushes.Done()
		return waitDMImportReceipt(ctx, receipt)
	case <-stop:
		pushes.Done()
		return Receipt{Reason: "error: ingestion is stopping"}, false
	case <-ctx.Done():
		pushes.Done()
		return Receipt{Reason: "error: import queue wait timed out"}, false
	}
}

func beginPush() (chan queueItem, <-chan struct{}, bool) {
	startMutex.Lock()
	defer startMutex.Unlock()
	if !started {
		return nil, nil, false
	}
	pushes.Add(1)
	return eventsChan, stopping, true
}

func waitDMImportReceipt(ctx context.Context, receipt <-chan Receipt) (Receipt, bool) {
	select {
	case result := <-receipt:
		return result, result.Reason == ""
	case <-ctx.Done():
		metrics.NostrRelayIngestionUncertain.Inc()
		return Receipt{
			Uncertain: true,
			Reason:    "error: import outcome uncertain (timed out after queueing)",
		}, false
	}
}

func (w *worker) run(ctx context.Context, workerID int) {
	for {
		select {
		case item, ok := <-eventsChan:
			metrics.NostrRelayIngestionQueueDepth.Set(float64(len(eventsChan)))
			if !ok {
				w.flush(ctx)
				return
			}

			if !item.dmImport && w.isDuplicate(item.event) {
				statsDuplicates.Add(1)
				metrics.NostrRelayIngestionDuplicates.Inc()
				continue
			}

			w.batch = append(w.batch, item)

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
	normal := make([]*nostr.Event, 0, len(batch))
	imports := make([]queueItem, 0, len(batch))
	for _, item := range batch {
		if item.dmImport {
			imports = append(imports, item)
		} else {
			normal = append(normal, item.event)
		}
	}
	result, err := insertBatch(ctx, normal)
	importResult, importReceipts, importErr := insertDMImportBatch(ctx, imports)
	result.Inserted += importResult.Inserted
	result.Duplicates += importResult.Duplicates
	result.Rejected += importResult.Rejected
	if accepted := importResult.Inserted + importResult.Duplicates; accepted > 0 {
		metrics.NostrEventsAcceptedTotal.Add(float64(accepted))
	}
	if err == nil && importErr != nil {
		err = importErr
	}
	for index, item := range imports {
		if item.receipt != nil {
			item.receipt <- importReceipts[index]
		}
	}
	duration := time.Since(startTime)
	outcome := "success"
	if err != nil {
		outcome = "error"
	}
	metrics.NostrRelayIngestionDuration.WithLabelValues(outcome).Observe(duration.Seconds())
	if result.Inserted > 0 {
		statsEventsInserted.Add(int64(result.Inserted))
		metrics.NostrRelayEventsInserted.Add(float64(result.Inserted))
	}
	if result.Duplicates > 0 {
		statsDuplicates.Add(int64(result.Duplicates))
		metrics.NostrRelayIngestionDuplicates.Add(float64(result.Duplicates))
	}
	if result.Rejected > 0 {
		metrics.NostrRelayIngestionRejected.Add(float64(result.Rejected))
	}
	if result.Uncertain > 0 {
		metrics.NostrRelayIngestionUncertain.Add(float64(result.Uncertain))
	}

	if err != nil {
		statsErrors.Add(1)
		metrics.NostrRelayIngestionErrors.Inc()
		fields := []zap.Field{
			zap.Error(err),
			zap.Int("worker_id", w.id),
			zap.Int("batch_size", len(batch)),
			zap.Duration("duration", duration),
		}

		var batchErr *dbstore.BatchInsertError
		if errors.As(err, &batchErr) {
			fields = append(fields,
				zap.Int("event_index", batchErr.Index),
				zap.String("event_id", batchErr.EventID),
				zap.Int("kind", batchErr.Kind),
				zap.String("pubkey", batchErr.Pubkey),
				zap.Int("content_length", batchErr.ContentLength),
				zap.Int("tags_count", batchErr.TagsCount),
				zap.Int("raw_size", batchErr.RawSize),
			)
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			fields = append(fields,
				zap.String("sqlstate", pgErr.Code),
				zap.String("pg_message", pgErr.Message),
				zap.String("pg_detail", pgErr.Detail),
				zap.String("pg_table", pgErr.TableName),
				zap.String("pg_constraint", pgErr.ConstraintName),
			)
		}

		log.Logger.Error("batch insert failed", fields...)
	} else {
		statsBatchProcessed.Add(1)
		metrics.NostrRelayBatchProcessed.Inc()

		log.Logger.Debug("batch inserted",
			zap.Int("inserted", result.Inserted),
			zap.Int("duplicates", result.Duplicates),
			zap.Int("rejected", result.Rejected),
			zap.Int("uncertain", result.Uncertain),
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

func insertBatch(ctx context.Context, events []*nostr.Event) (batchResult, error) {
	result := batchResult{}
	if len(events) == 0 {
		return result, nil
	}

	accepted := make([]*nostr.Event, 0, len(events))
	stored := make([]*nostr.Event, 0, len(events))

	for _, evt := range events {
		reject, reason := policies.P.ValidateBatchEvent(ctx, evt)
		if reject {
			result.Rejected++
			log.Logger.Debug("ingestion policy rejected event", zap.String("event_id", evt.ID), zap.String("reason", reason))
			continue
		}

		if err := prepareEventForStorage(ctx, evt); err != nil {
			if errors.Is(err, dbstore.ErrDupEvent) {
				result.Duplicates++
				continue
			}
			return result, err
		}

		accepted = append(accepted, evt)
		if !nostr.IsEphemeralKind(evt.Kind) {
			stored = append(stored, evt)
		}
	}

	if len(stored) > 0 {
		batchInsertResult, err := db.DbQueries.InsertEventBatch(ctx, stored)
		result.Inserted += batchInsertResult.Inserted
		result.Duplicates += batchInsertResult.Duplicates
		result.Rejected += len(batchInsertResult.RejectedEventIDs)
		if err != nil {
			return result, err
		}
		accepted = removeRejectedEvents(accepted, batchInsertResult.RejectedEventIDs)
	}

	for _, evt := range accepted {
		if !nostr.IsEphemeralKind(evt.Kind) && policies.ContentSpamFlagged(evt) {
			if err := db.DbQueries.UpsertContentSpamFlag(ctx, evt.ID); err != nil {
				log.Logger.Warn("content spam flag persistence failed", zap.String("event_id", evt.ID), zap.Error(err))
			}
		}
		listener.NotifyListeners(evt)
		stream.ForwardEvent(*evt)
		if ps := pubsub.GetPubSub(); ps != nil && ps.IsEnabled() {
			_ = ps.PublishEvent(ctx, evt)
		}

		serialized, _ := json.Marshal(evt)
		_ = cache.SetEvent(evt.ID, string(serialized))
		metrics.NostrKindEventCounter.WithLabelValues(metrics.GetKindName(evt.Kind)).Inc()
		if err := groups.AfterStoreEvent(ctx, evt); err != nil {
			log.Logger.Warn("nip29 post-persist handling failed",
				zap.String("event_id", evt.ID),
				zap.Int("kind", evt.Kind),
				zap.Error(err),
			)
		}
	}

	return result, nil
}

// insertDMImportBatch validates admitted browser imports, performs one set-based
// write for the stored events, and maps the database result back to each caller.
// A receipt is produced for every input so a disconnected browser never causes a
// worker to block and no event is reported as inserted before PostgreSQL confirms it.
func insertDMImportBatch(ctx context.Context, imports []queueItem) (batchResult, []Receipt, error) {
	result := batchResult{}
	receipts := make([]Receipt, len(imports))
	stored := make([]*nostr.Event, 0, len(imports))
	storedIndexes := make([]int, 0, len(imports))
	ephemeralIndexes := make([]int, 0, len(imports))

	for index, item := range imports {
		if !item.admission.ValidFor(item.event) {
			receipts[index] = Receipt{Reason: "restricted: invalid DM import admission"}
			result.Rejected++
			continue
		}
		if reject, reason := policies.P.ValidateDMImportEvent(ctx, item.event); reject {
			log.Logger.Debug("DM import policy rejected event", zap.String("event_id", item.event.ID), zap.String("reason", reason))
			receipts[index] = Receipt{Reason: reason}
			result.Rejected++
			continue
		}
		if err := validateImportStorageCandidate(item.event); err != nil {
			if errors.Is(err, dbstore.ErrDupEvent) {
				receipts[index] = Receipt{Duplicate: true}
				result.Duplicates++
				continue
			}
			receipts[index] = Receipt{Reason: "error: import preparation failed"}
			result.Rejected++
			continue
		}
		if nostr.IsEphemeralKind(item.event.Kind) {
			ephemeralIndexes = append(ephemeralIndexes, index)
			continue
		}
		stored = append(stored, item.event)
		storedIndexes = append(storedIndexes, index)
	}

	for _, index := range ephemeralIndexes {
		receipts[index] = Receipt{Inserted: true}
		result.Inserted++
	}
	if len(stored) == 0 {
		return result, receipts, nil
	}

	batch, err := db.DbQueries.InsertDMImportBatch(ctx, stored)
	result = applyDMImportBatchResult(stored, storedIndexes, batch, result, receipts, err == nil)
	for position, evt := range stored {
		if receipts[storedIndexes[position]].Inserted {
			notifyStoredImport(ctx, evt)
		}
	}
	if err != nil {
		return result, receipts, err
	}
	return result, receipts, nil
}

func validateImportStorageCandidate(evt *nostr.Event) error {
	if evt == nil {
		return errors.New("nil import event")
	}
	return nil
}

// applyDMImportBatchResult maps the database's set-based result to the input
// order. Events absent from both returned lists are ON CONFLICT duplicates.
func applyDMImportBatchResult(stored []*nostr.Event, storedIndexes []int, batch dbstore.BatchInsertResult, result batchResult, receipts []Receipt, complete bool) batchResult {
	inserted := make(map[string]struct{}, len(batch.InsertedIDs))
	for _, id := range batch.InsertedIDs {
		inserted[id] = struct{}{}
	}
	rejected := make(map[string]struct{}, len(batch.RejectedEventIDs))
	for _, id := range batch.RejectedEventIDs {
		rejected[id] = struct{}{}
	}
	consumedInserted := make(map[string]struct{}, len(batch.InsertedIDs))
	for position, evt := range stored {
		index := storedIndexes[position]
		if _, rejectedByDB := rejected[evt.ID]; rejectedByDB {
			receipts[index] = Receipt{Reason: "error: import rejected"}
			result.Rejected++
			continue
		}
		if _, insertedByDB := inserted[evt.ID]; insertedByDB {
			if _, alreadyConsumed := consumedInserted[evt.ID]; alreadyConsumed {
				receipts[index] = Receipt{Duplicate: true}
				result.Duplicates++
				continue
			}
			consumedInserted[evt.ID] = struct{}{}
			receipts[index] = Receipt{Inserted: true}
			result.Inserted++
			continue
		}
		if !complete {
			receipts[index] = Receipt{
				Uncertain: true,
				Reason:    "error: import outcome uncertain after persistence failure",
			}
			result.Uncertain++
			continue
		}
		receipts[index] = Receipt{Duplicate: true}
		result.Duplicates++
	}
	return result
}

func notifyStoredImport(ctx context.Context, evt *nostr.Event) {
	materializeImportedEvent(ctx, evt)
	listener.NotifyListeners(evt)
	stream.ForwardEvent(*evt)
	if ps := pubsub.GetPubSub(); ps != nil && ps.IsEnabled() {
		_ = ps.PublishEvent(ctx, evt)
	}
}

func materializeImportedEvent(ctx context.Context, evt *nostr.Event) {
	if evt == nil || evt.Kind != nostr.KindProfileMetadata {
		return
	}
	if err := upsertImportedProfile(ctx, evt); err != nil {
		log.Logger.Warn("imported profile materialization failed", zap.String("event_id", evt.ID), zap.Error(err))
	}
}

func removeRejectedEvents(events []*nostr.Event, rejectedIDs []string) []*nostr.Event {
	if len(rejectedIDs) == 0 {
		return events
	}

	rejected := make(map[string]struct{}, len(rejectedIDs))
	for _, id := range rejectedIDs {
		rejected[id] = struct{}{}
	}

	accepted := make([]*nostr.Event, 0, len(events)-len(rejectedIDs))
	for _, event := range events {
		if _, ok := rejected[event.ID]; !ok {
			accepted = append(accepted, event)
		}
	}

	return accepted
}

func prepareEventForStorage(ctx context.Context, evt *nostr.Event) error {
	if nostr.IsEphemeralKind(evt.Kind) {
		return nil
	}

	previous, err := replacementTarget(ctx, evt)
	if err != nil {
		return err
	}
	if previous != nil {
		if err := deleteStoredEventByID(ctx, previous.ID, evt.ID); err != nil {
			return err
		}
	}

	return nil
}

// replacementTarget finds the superseded event but never mutates storage. DM
// imports use it before their batch write and delete only after that write has
// confirmed the replacement, avoiding loss when a batch fails.
func replacementTarget(ctx context.Context, evt *nostr.Event) (*nostr.Event, error) {
	if nostr.IsEphemeralKind(evt.Kind) {
		return nil, nil
	}
	if nostr.IsReplaceableKind(evt.Kind) {
		previous, err := queryFirstStoredEvent(ctx, nostr.Filter{Authors: []string{evt.PubKey}, Kinds: []int{evt.Kind}})
		if err != nil || previous == nil || !isOlder(previous, evt) {
			return nil, err
		}
		return previous, nil
	}
	if nostr.IsAddressableKind(evt.Kind) {
		d := evt.Tags.GetFirst([]string{"d", ""})
		if d == nil {
			return nil, nil
		}
		previous, err := queryFirstStoredEvent(ctx, nostr.Filter{Authors: []string{evt.PubKey}, Kinds: []int{evt.Kind}, Tags: nostr.TagMap{"d": []string{d.Value()}}})
		if err != nil || previous == nil || !isOlder(previous, evt) {
			return nil, err
		}
		return previous, nil
	}
	return nil, nil
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
