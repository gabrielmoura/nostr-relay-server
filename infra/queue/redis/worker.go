package redisqueue

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	iredis "github.com/gabrielmoura/nostr-relay-server/infra/redis"
	"github.com/gabrielmoura/nostr-relay-server/internal/jobs"
	syncjob "github.com/gabrielmoura/nostr-relay-server/internal/sync"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const syncRemoteConcurrencyReason = "remote relay concurrency cap reached"

type Worker struct {
	name       string
	client     *iredis.Client
	registry   *jobs.MemoryRegistry
	dispatcher *Dispatcher
	tracker    *Tracker
	config     RuntimeConfig
	scripts    *Scripts
	queues     []string
}

func NewWorker(name string, client *iredis.Client, registry *jobs.MemoryRegistry, dispatcher *Dispatcher, tracker *Tracker, cfg RuntimeConfig, scripts *Scripts) *Worker {
	return &Worker{
		name:       name,
		client:     client,
		registry:   registry,
		dispatcher: dispatcher,
		tracker:    tracker,
		config:     cfg,
		scripts:    scripts,
		queues:     []string{config.Cfg.Jobs.Download.Queue, config.Cfg.Jobs.Sync.Queue, config.Cfg.Jobs.Cron.Queue, cfg.DefaultQueue},
	}
}

func (w *Worker) Prepare(ctx context.Context) error {
	seen := make(map[string]struct{})
	for _, queueName := range w.queues {
		if queueName == "" {
			continue
		}
		if _, ok := seen[queueName]; ok {
			continue
		}
		seen[queueName] = struct{}{}
		keys := NewKeys(queueName)
		for _, priority := range []jobs.Priority{jobs.PriorityHigh, jobs.PriorityNormal, jobs.PriorityLow} {
			streamKey := keys.Stream(priority)
			err := w.client.Raw().XGroupCreateMkStream(ctx, streamKey, w.config.ConsumerGroup, "0").Err()
			if err != nil && !stringsContains(err.Error(), "BUSYGROUP") {
				return fmt.Errorf("create consumer group for %s: %w", streamKey, err)
			}
		}
	}

	return nil

}

func (w *Worker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		claimed := false
		for _, queueName := range w.queues {
			if queueName == "" {
				continue
			}
			processed, err := w.processQueue(ctx, queueName)
			if err != nil {
				log.Logger.Warn("queue worker iteration failed", zap.String("worker", w.name), zap.String("queue", queueName), zap.Error(err))
			}
			if processed > 0 {
				claimed = true
			}
		}

		if !claimed {
			select {
			case <-ctx.Done():
				return
			case <-time.After(200 * time.Millisecond):
			}
		}
	}
}

func (w *Worker) processQueue(ctx context.Context, queueName string) (int, error) {
	keys := NewKeys(queueName)
	streams := []string{keys.Stream(jobs.PriorityHigh), keys.Stream(jobs.PriorityNormal), keys.Stream(jobs.PriorityLow)}
	values := []string{">", ">", ">"}
	args := &goredis.XReadGroupArgs{
		Group:    w.config.ConsumerGroup,
		Consumer: w.name,
		Streams:  append(streams, values...),
		Count:    w.config.BatchSize,
		Block:    w.config.BlockTimeout,
	}
	results, err := w.client.Raw().XReadGroup(ctx, args).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return 0, w.reclaimPending(ctx, queueName)
		}
		metrics.NostrQueueRedisErrorsTotal.WithLabelValues(queueName, "xreadgroup").Inc()
		return 0, err
	}

	processed := 0
	for _, result := range results {
		priority := priorityFromStream(result.Stream)
		for _, message := range result.Messages {
			if err := w.handleMessage(ctx, queueName, priority, result.Stream, message); err != nil {
				log.Logger.Warn("queue job handling failed", zap.String("worker", w.name), zap.String("queue", queueName), zap.String("stream", result.Stream), zap.Error(err))
			}
			processed++
		}
	}

	if reclaimErr := w.reclaimPending(ctx, queueName); reclaimErr != nil {
		log.Logger.Warn("queue reclaim failed", zap.String("worker", w.name), zap.String("queue", queueName), zap.Error(reclaimErr))
	}

	return processed, nil
}

func (w *Worker) handleMessage(ctx context.Context, queueName string, priority jobs.Priority, streamKey string, message goredis.XMessage) error {
	rawID, ok := message.Values["i"]
	if !ok {
		return fmt.Errorf("stream message missing job id")
	}
	jobID, err := jobs.ParseJobID(fmt.Sprint(rawID))
	if err != nil {
		return err
	}
	keys := NewKeys(queueName)
	body, err := w.client.Get(ctx, keys.Body(jobID))
	if err != nil {
		return fmt.Errorf("load job body %s: %w", jobID.String(), err)
	}
	envelope, err := jobs.UnmarshalEnvelope([]byte(body))
	if err != nil {
		return err
	}
	handler, ok := w.registry.Lookup(envelope.Name)
	if !ok {
		return w.failDead(ctx, queueName, envelope.Name, jobID, streamKey, message.ID, fmt.Errorf("no handler registered for %s", envelope.Name), "")
	}

	jobPayload, err := handler.Decode(envelope.Payload)
	if err != nil {
		return w.failDead(ctx, queueName, envelope.Name, jobID, streamKey, message.ID, err, "")
	}
	if deferred, err := w.deferIfRemoteSyncAtCapacity(ctx, queueName, envelope.Name, priority, jobID, streamKey, message.ID, jobPayload); err != nil || deferred {
		return err
	}

	now := time.Now().UTC()
	attemptValue, err := w.scripts.start.Run(ctx, w.client.Raw(), []string{
		keys.State(),
		keys.Attempts(),
		keys.MetricsBucket(now),
		keys.Meta(jobID),
	}, jobID.String(), now.UnixMilli(), int(w.config.MetricsTTL.Seconds())).Result()
	if err != nil {
		metrics.NostrQueueLuaErrorsTotal.WithLabelValues(queueName, "start").Inc()
		return fmt.Errorf("mark job start: %w", err)
	}
	if raw, ok := attemptValue.(int64); ok && raw == -1 {
		return w.ackCanceled(ctx, queueName, streamKey, message.ID, jobID)
	}
	attempts, err := toUint64(attemptValue)
	if err != nil {
		return err
	}

	metrics.NostrQueueJobsStartedTotal.WithLabelValues(queueName, envelope.Name, w.name).Inc()
	metrics.NostrQueueJobLatencySeconds.WithLabelValues(queueName, envelope.Name).Observe(time.Since(time.UnixMilli(envelope.CreatedAtMS)).Seconds())
	workerBucket := keys.WorkersBucket(now)
	if err := w.client.Raw().PFAdd(ctx, workerBucket, w.name).Err(); err == nil {
		_ = w.client.Expire(ctx, workerBucket, w.config.WorkerWindow)
		metrics.NostrQueueActiveWorkers.WithLabelValues(queueName).Set(float64(1))
	}

	handlerCtx := ctx
	if envelope.TimeoutMS > 0 {
		var cancel context.CancelFunc
		handlerCtx, cancel = context.WithTimeout(ctx, time.Duration(envelope.TimeoutMS)*time.Millisecond)
		defer cancel()
	}
	executionState := &jobs.ExecutionState{ID: jobID, Queue: queueName, Name: envelope.Name}
	handlerCtx = jobs.WithExecutionState(handlerCtx, executionState)

	started := time.Now()
	releaseRemote, err := w.acquireRemoteSyncSlot(handlerCtx, queueName, envelope.Name, jobPayload)
	if err != nil {
		return w.failDead(ctx, queueName, envelope.Name, jobID, streamKey, message.ID, err, "")
	}
	if releaseRemote != nil {
		defer releaseRemote()
	}

	err = handler.Handle(handlerCtx, jobPayload)
	duration := time.Since(started)
	metrics.NostrQueueJobDurationSeconds.WithLabelValues(queueName, envelope.Name).Observe(duration.Seconds())
	resultJSON, resultErr := jobs.ResultJSON(handlerCtx)
	if resultErr != nil {
		log.Logger.Warn("queue job result marshal failed", zap.String("worker", w.name), zap.String("queue", queueName), zap.Error(resultErr))
	}

	if canceled, cancelErr := w.jobCanceled(ctx, queueName, jobID); cancelErr != nil {
		return cancelErr
	} else if canceled {
		return w.ackCanceled(ctx, queueName, streamKey, message.ID, jobID)
	}

	if err == nil {
		_, ackErr := w.scripts.ackSuccess.Run(ctx, w.client.Raw(), []string{
			streamKey,
			keys.Body(jobID),
			keys.State(),
			keys.MetricsBucket(time.Now().UTC()),
			keys.Meta(jobID),
			keys.Result(jobID),
		}, w.config.ConsumerGroup, message.ID, jobID.String(), duration.Milliseconds(), time.Now().UTC().UnixMilli(), resultJSON, int(w.config.ResultTTL.Seconds()), int(w.config.MetricsTTL.Seconds())).Result()
		if ackErr != nil {
			metrics.NostrQueueLuaErrorsTotal.WithLabelValues(queueName, "ack_success").Inc()
			return fmt.Errorf("ack success: %w", ackErr)
		}
		metrics.NostrQueueJobsSucceededTotal.WithLabelValues(queueName, envelope.Name).Inc()
		return nil
	}

	attemptCount := uint8(attempts)
	if attemptCount >= envelope.MaxAttempts {
		return w.failDead(ctx, queueName, envelope.Name, jobID, streamKey, message.ID, err, resultJSON)
	}

	backoff := w.dispatcher.Backoff(attemptCount)
	runAt := time.Now().UTC().Add(backoff)
	_, retryErr := w.scripts.retry.Run(ctx, w.client.Raw(), []string{
		streamKey,
		keys.Delayed(),
		keys.State(),
		keys.MetricsBucket(time.Now().UTC()),
		keys.Meta(jobID),
	}, w.config.ConsumerGroup, message.ID, jobID.String(), runAt.UnixMilli(), truncateError(err), time.Now().UTC().UnixMilli(), int(w.config.MetricsTTL.Seconds())).Result()
	if retryErr != nil {
		metrics.NostrQueueLuaErrorsTotal.WithLabelValues(queueName, "retry").Inc()
		return fmt.Errorf("schedule retry: %w", retryErr)
	}
	metrics.NostrQueueJobsRetriedTotal.WithLabelValues(queueName, envelope.Name).Inc()

	return nil
}

func (w *Worker) deferIfRemoteSyncAtCapacity(
	ctx context.Context,
	queueName string,
	jobName string,
	priority jobs.Priority,
	jobID jobs.JobID,
	streamKey string,
	messageID string,
	jobPayload jobs.Job,
) (bool, error) {
	if jobName != (syncjob.QueueJob{}).Name() || w.config.SyncMaxConcurrentPerRemote <= 0 {
		return false, nil
	}

	syncJob, ok := jobPayload.(syncjob.QueueJob)
	if !ok {
		return false, nil
	}
	remoteKey := strings.TrimSpace(strings.ToLower(syncJob.Remote))
	if remoteKey == "" {
		return false, nil
	}

	activeKey := syncRemoteActiveKey(queueName, remoteKey)
	active, err := w.client.Raw().Get(ctx, activeKey).Int()
	if err != nil && !errors.Is(err, goredis.Nil) {
		return false, fmt.Errorf("read sync remote concurrency: %w", err)
	}
	if active < w.config.SyncMaxConcurrentPerRemote {
		return false, nil
	}

	runAt := time.Now().UTC().Add(w.dispatcher.Backoff(1))
	keys := NewKeys(queueName)
	_, err = w.scripts.deferJob.Run(ctx, w.client.Raw(), []string{
		streamKey,
		keys.Delayed(),
		keys.State(),
		keys.MetricsBucket(time.Now().UTC()),
		keys.Meta(jobID),
	}, w.config.ConsumerGroup, messageID, jobID.String(), runAt.UnixMilli(), syncRemoteConcurrencyReason, time.Now().UTC().UnixMilli(), int(w.config.MetricsTTL.Seconds())).Result()
	if err != nil {
		metrics.NostrQueueLuaErrorsTotal.WithLabelValues(queueName, "defer_remote_cap").Inc()
		return false, fmt.Errorf("defer sync job %s: %w", jobID.String(), err)
	}
	return true, nil
}

func (w *Worker) acquireRemoteSyncSlot(ctx context.Context, queueName string, jobName string, jobPayload jobs.Job) (func(), error) {
	if jobName != (syncjob.QueueJob{}).Name() || w.config.SyncMaxConcurrentPerRemote <= 0 {
		return nil, nil
	}

	syncJob, ok := jobPayload.(syncjob.QueueJob)
	if !ok {
		return nil, nil
	}
	remoteKey := strings.TrimSpace(strings.ToLower(syncJob.Remote))
	if remoteKey == "" {
		return nil, nil
	}

	key := syncRemoteActiveKey(queueName, remoteKey)
	count, err := w.client.Raw().Incr(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("acquire sync remote slot: %w", err)
	}
	_ = w.client.Raw().Expire(ctx, key, w.config.DefaultTimeout+time.Minute).Err()
	if int(count) > w.config.SyncMaxConcurrentPerRemote {
		_, _ = w.client.Raw().Decr(ctx, key).Result()
		return nil, fmt.Errorf(syncRemoteConcurrencyReason)
	}

	return func() {
		_, _ = w.client.Raw().Decr(context.Background(), key).Result()
	}, nil
}

func (w *Worker) jobCanceled(ctx context.Context, queueName string, jobID jobs.JobID) (bool, error) {
	state, err := w.client.Raw().BitField(ctx, NewKeys(queueName).State(), "GET", "u3", "#"+jobID.String()).Result()
	if err != nil {
		return false, fmt.Errorf("read job state: %w", err)
	}
	return len(state) > 0 && jobs.Status(state[0]) == jobs.StatusCanceled, nil
}

func (w *Worker) ackCanceled(ctx context.Context, queueName, streamKey, messageID string, jobID jobs.JobID) error {
	if err := w.client.Raw().XAck(ctx, streamKey, w.config.ConsumerGroup, messageID).Err(); err != nil {
		return fmt.Errorf("ack canceled job %s: %w", jobID.String(), err)
	}
	if err := w.client.Raw().XDel(ctx, streamKey, messageID).Err(); err != nil {
		return fmt.Errorf("delete canceled job %s: %w", jobID.String(), err)
	}
	return nil
}

func syncRemoteActiveKey(queueName, remote string) string {
	return fmt.Sprintf("rq:{%s}:sync:remote:%s:active", queueName, remote)
}

func (w *Worker) failDead(ctx context.Context, queueName, jobName string, jobID jobs.JobID, streamKey, entryID string, cause error, resultJSON string) error {
	keys := NewKeys(queueName)
	_, err := w.scripts.moveDead.Run(ctx, w.client.Raw(), []string{
		streamKey,
		keys.Dead(),
		keys.State(),
		keys.MetricsBucket(time.Now().UTC()),
		keys.Meta(jobID),
		keys.Result(jobID),
	}, w.config.ConsumerGroup, entryID, jobID.String(), time.Now().UTC().UnixMilli(), truncateError(cause), resultJSON, int(w.config.ResultTTL.Seconds()), int(w.config.MetricsTTL.Seconds())).Result()
	if err != nil {
		metrics.NostrQueueLuaErrorsTotal.WithLabelValues(queueName, "move_dead").Inc()
		return fmt.Errorf("move dead job: %w", err)
	}
	metrics.NostrQueueJobsFailedTotal.WithLabelValues(queueName, jobName, jobs.StatusDead.String()).Inc()
	metrics.NostrQueueJobsDeadTotal.WithLabelValues(queueName, jobName).Inc()
	return cause
}

func (w *Worker) reclaimPending(ctx context.Context, queueName string) error {
	keys := NewKeys(queueName)
	for _, priority := range []jobs.Priority{jobs.PriorityHigh, jobs.PriorityNormal, jobs.PriorityLow} {
		streamKey := keys.Stream(priority)
		result, _, err := w.client.Raw().XAutoClaim(ctx, &goredis.XAutoClaimArgs{
			Stream:   streamKey,
			Group:    w.config.ConsumerGroup,
			Consumer: w.name,
			MinIdle:  w.config.ReclaimIdle,
			Start:    "0-0",
			Count:    w.config.ReclaimBatchSize,
		}).Result()
		if err != nil {
			if errors.Is(err, goredis.Nil) {
				continue
			}
			metrics.NostrQueueRedisErrorsTotal.WithLabelValues(queueName, "xautoclaim").Inc()
			return err
		}
		if len(result) == 0 {
			continue
		}
		metrics.NostrQueueReclaimsTotal.WithLabelValues(queueName).Add(float64(len(result)))
		for _, message := range result {
			if err := w.handleMessage(ctx, queueName, priority, streamKey, message); err != nil {
				log.Logger.Warn("reclaimed job failed", zap.String("worker", w.name), zap.String("queue", queueName), zap.Error(err))
			}
		}
	}

	return nil
}

func priorityFromStream(stream string) jobs.Priority {
	if stringsHasSuffix(stream, ":high") {
		return jobs.PriorityHigh
	}
	if stringsHasSuffix(stream, ":low") {
		return jobs.PriorityLow
	}
	return jobs.PriorityNormal
}

func truncateError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if len(msg) <= 255 {
		return msg
	}
	return msg[:255]
}

func stringsContains(source, needle string) bool {
	return strings.Contains(source, needle)
}

func stringsHasSuffix(source, suffix string) bool {
	if len(suffix) > len(source) {
		return false
	}
	return strings.HasSuffix(source, suffix)
}
