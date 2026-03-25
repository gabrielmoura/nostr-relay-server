package stream

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	nostr_custom "github.com/gabrielmoura/nostr-relay-server/infra/nostr-custom"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/gabrielmoura/nostr-relay-server/pkg/nostrpool"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

const (
	defaultEventQueueSize = 1024
	defaultReqQueueSize   = 256
	defaultWorkerCount    = 2
)

type requestJob struct {
	ws     *dto.WsServer
	filter nostr.Filter
	id     *string
}

type Dispatcher struct {
	eventQueue chan nostr.Event
	reqQueue   chan requestJob
	started    bool
	workers    int
	droppedEvt atomic.Uint64
	droppedReq atomic.Uint64
	mu         sync.RWMutex
}

var dispatcher = &Dispatcher{}

type DispatcherStats struct {
	Started          bool   `json:"started"`
	WorkerCount      int    `json:"worker_count"`
	EventQueueLen    int    `json:"event_queue_len"`
	EventQueueCap    int    `json:"event_queue_cap"`
	RequestQueueLen  int    `json:"request_queue_len"`
	RequestQueueCap  int    `json:"request_queue_cap"`
	DroppedEventJobs uint64 `json:"dropped_event_jobs"`
	DroppedReqJobs   uint64 `json:"dropped_request_jobs"`
}

func Start(ctx context.Context) {
	dispatcher.mu.Lock()
	defer dispatcher.mu.Unlock()
	if dispatcher.started {
		return
	}
	dispatcher.eventQueue = make(chan nostr.Event, defaultEventQueueSize)
	dispatcher.reqQueue = make(chan requestJob, defaultReqQueueSize)
	dispatcher.started = true
	dispatcher.workers = defaultWorkerCount

	for i := 0; i < defaultWorkerCount; i++ {
		go dispatcher.runEventWorker(ctx)
		go dispatcher.runRequestWorker(ctx)
	}
}

func ForwardEvent(event nostr.Event) {
	if !config.Cfg.Stream.StreamUp || !shouldForwardEvent(event.Kind) {
		return
	}
	if !dispatcher.enqueueEvent(event) {
		log.Logger.Debug("dropping upstream event due to full queue", zap.String("event_id", event.ID), zap.Int("kind", event.Kind))
	}
}

func ForwardRequest(ws *dto.WsServer, filter nostr.Filter, id *string) {
	if !config.Cfg.Stream.StreamDown {
		return
	}
	if !dispatcher.enqueueRequest(requestJob{ws: ws, filter: filter, id: id}) {
		log.Logger.Debug("dropping downstream request due to full queue", zap.String("subscription_id", valueOrEmpty(id)))
	}
}

func shouldForwardEvent(kind int) bool {
	switch kind {
	case nostr.KindTextNote, nostr.KindDeletion, nostr.KindReaction, nostr.KindProfileMetadata, nostr.KindRepost, nostr_custom.KindEditContent:
		return true
	default:
		return false
	}
}

func (d *Dispatcher) enqueueEvent(event nostr.Event) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.started {
		return false
	}
	select {
	case d.eventQueue <- event:
		return true
	default:
		d.droppedEvt.Add(1)
		return false
	}
}

func (d *Dispatcher) enqueueRequest(job requestJob) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.started {
		return false
	}
	select {
	case d.reqQueue <- job:
		return true
	default:
		d.droppedReq.Add(1)
		return false
	}
}

func Snapshot() DispatcherStats {
	dispatcher.mu.RLock()
	defer dispatcher.mu.RUnlock()

	stats := DispatcherStats{
		Started:          dispatcher.started,
		WorkerCount:      dispatcher.workers,
		DroppedEventJobs: dispatcher.droppedEvt.Load(),
		DroppedReqJobs:   dispatcher.droppedReq.Load(),
	}

	if dispatcher.started {
		stats.EventQueueLen = len(dispatcher.eventQueue)
		stats.EventQueueCap = cap(dispatcher.eventQueue)
		stats.RequestQueueLen = len(dispatcher.reqQueue)
		stats.RequestQueueCap = cap(dispatcher.reqQueue)
	}

	return stats
}

func (d *Dispatcher) runEventWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-d.eventQueue:
			metrics.NostrRelayEventForwardedTotal.Inc()
			if err := nostrpool.Publish(&event); err != nil {
				metrics.NostrRelayEventForwardedFailuresTotal.Inc()
				if !errors.Is(err, nostrpool.ErrNotRelayConnected) {
					log.Logger.Warn("failed to publish event to relay pool", zap.Error(err), zap.String("event_id", event.ID))
				}
			}
		}
	}
}

func (d *Dispatcher) runRequestWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-d.reqQueue:
			allEvents, err := nostrpool.Subscribe(nostr.Filters{job.filter})
			if err != nil {
				if !errors.Is(err, nostrpool.ErrNotRelayConnected) {
					log.Logger.Warn("failed to collect events from relay pool", zap.Error(err))
				}
				continue
			}

			for ev := range allEvents {
				metrics.NostrRelayRequestForwardedTotal.Inc()
				_ = db.DbQueries.InsertEvent(job.ws.Ctx, ev)
				job.ws.ChanSender <- nostr.EventEnvelope{Event: *ev, SubscriptionID: job.id}
			}
		}
	}
}

func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
