package redisqueue

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	iredis "github.com/gabrielmoura/nostr-relay-server/infra/redis"
	"github.com/gabrielmoura/nostr-relay-server/internal/jobs"
)

var errQueueDisabled = errors.New("redis queue is disabled")

type Dispatcher struct {
	client  *iredis.Client
	runtime RuntimeConfig
	scripts *Scripts
	rng     *rand.Rand
}

func NewDispatcher(client *iredis.Client, runtime RuntimeConfig, scripts *Scripts) *Dispatcher {
	return &Dispatcher{
		client:  client,
		runtime: runtime,
		scripts: scripts,
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (d *Dispatcher) Dispatch(ctx context.Context, job jobs.Job, opts ...jobs.DispatchOption) (jobs.JobID, error) {
	return d.dispatch(ctx, job, time.Time{}, opts...)
}

func (d *Dispatcher) DispatchIn(ctx context.Context, job jobs.Job, delay time.Duration, opts ...jobs.DispatchOption) (jobs.JobID, error) {
	return d.dispatch(ctx, job, time.Now().UTC().Add(delay), opts...)
}

func (d *Dispatcher) DispatchAt(ctx context.Context, job jobs.Job, runAt time.Time, opts ...jobs.DispatchOption) (jobs.JobID, error) {
	return d.dispatch(ctx, job, runAt, opts...)
}

func (d *Dispatcher) dispatch(ctx context.Context, job jobs.Job, runAt time.Time, opts ...jobs.DispatchOption) (jobs.JobID, error) {
	if d.client == nil || !d.client.IsEnabled() {
		return 0, errQueueDisabled
	}

	resolved := jobs.DispatchConfig{
		Queue:       d.runtime.DefaultQueue,
		Priority:    jobs.PriorityNormal,
		Timeout:     d.runtime.DefaultTimeout,
		MaxAttempts: d.runtime.DefaultMaxAttempts,
	}
	for _, opt := range opts {
		opt(&resolved)
	}
	if !runAt.IsZero() {
		resolved.RunAt = runAt.UTC()
	}
	if resolved.Queue == "" {
		resolved.Queue = d.runtime.DefaultQueue
	}
	if resolved.Timeout <= 0 {
		resolved.Timeout = d.runtime.DefaultTimeout
	}
	if resolved.MaxAttempts == 0 {
		resolved.MaxAttempts = d.runtime.DefaultMaxAttempts
	}
	resolved.Priority = resolved.Priority.Normalize()

	now := time.Now().UTC()
	body, err := jobs.MarshalEnvelope(job, resolved, now)
	if err != nil {
		return 0, err
	}

	keys := NewKeys(resolved.Queue)
	metricKey := keys.MetricsBucket(now)
	args := []any{
		string(body),
		job.Name(),
		string(resolved.Priority),
		now.UnixMilli(),
		resolved.RunAt.UTC().UnixMilli(),
		int(d.runtime.BodyTTL.Seconds()),
		d.runtime.MaxLenApprox,
		resolved.Queue,
		resolved.MaxAttempts,
		int(d.runtime.MetricsTTL.Seconds()),
	}
	if resolved.RunAt.IsZero() {
		args[4] = int64(0)
	}

	result, err := d.scripts.enqueue.Run(ctx, d.client.Raw(), []string{
		keys.Seq(),
		keys.BodyPrefix(),
		keys.State(),
		keys.Attempts(),
		keys.Stream(jobs.PriorityHigh),
		keys.Stream(jobs.PriorityNormal),
		keys.Stream(jobs.PriorityLow),
		keys.Delayed(),
		metricKey,
		keys.MetaPrefix(),
		keys.Jobs(),
		keys.ResultPrefix(),
	}, args...).Result()
	if err != nil {
		metrics.NostrQueueLuaErrorsTotal.WithLabelValues(resolved.Queue, "enqueue").Inc()
		return 0, fmt.Errorf("enqueue job %s: %w", job.Name(), err)
	}

	idValue, err := toUint64(result)
	if err != nil {
		return 0, err
	}
	id := jobs.JobID(idValue)
	metrics.NostrQueueJobsEnqueuedTotal.WithLabelValues(resolved.Queue, job.Name(), string(resolved.Priority)).Inc()

	return id, nil
}

func (d *Dispatcher) Backoff(attempts uint8) time.Duration {
	if attempts <= 1 {
		return d.runtime.RetryPolicy.BaseDelay
	}

	delay := d.runtime.RetryPolicy.BaseDelay
	for i := uint8(1); i < attempts; i++ {
		delay *= 2
		if delay >= d.runtime.RetryPolicy.MaxDelay {
			delay = d.runtime.RetryPolicy.MaxDelay
			break
		}
	}
	if d.runtime.RetryPolicy.Jitter <= 0 {
		return delay
	}
	jitterRange := int64(d.runtime.RetryPolicy.Jitter)
	if jitterRange <= 0 {
		return delay
	}
	jitter := time.Duration(d.rng.Int63n(jitterRange + 1))
	if delay+jitter > d.runtime.RetryPolicy.MaxDelay {
		return d.runtime.RetryPolicy.MaxDelay
	}
	return delay + jitter
}

func toUint64(value any) (uint64, error) {
	switch typed := value.(type) {
	case int64:
		return uint64(typed), nil
	case string:
		parsed, err := strconv.ParseUint(typed, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse integer result %q: %w", typed, err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unexpected integer result type %T", value)
	}
}
