package redisqueue

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	iredis "github.com/gabrielmoura/nostr-relay-server/infra/redis"
	"github.com/gabrielmoura/nostr-relay-server/internal/jobs"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	goredis "github.com/redis/go-redis/v9"
)

type Tracker struct {
	client  *iredis.Client
	runtime RuntimeConfig
	scripts *Scripts
}

func NewTracker(client *iredis.Client, runtime RuntimeConfig, scripts *Scripts) *Tracker {
	return &Tracker{client: client, runtime: runtime, scripts: scripts}
}

func (t *Tracker) Get(ctx context.Context, queue string, id jobs.JobID) (jobs.Snapshot, error) {
	if t.client == nil || !t.client.IsEnabled() {
		return jobs.Snapshot{}, errQueueDisabled
	}

	keys := NewKeys(queue)
	meta, err := t.client.HGetAll(ctx, keys.Meta(id))
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return jobs.Snapshot{}, fmt.Errorf("job %s not found", id.String())
		}
		return jobs.Snapshot{}, fmt.Errorf("load job meta: %w", err)
	}
	if len(meta) == 0 {
		return jobs.Snapshot{}, fmt.Errorf("job %s not found", id.String())
	}

	pipe := t.client.Raw().Pipeline()
	stateCmd := pipe.BitField(ctx, keys.State(), "GET", "u3", "#"+id.String())
	attemptCmd := pipe.BitField(ctx, keys.Attempts(), "GET", "u8", "#"+id.String())
	bodyCmd := pipe.Get(ctx, keys.Body(id))
	resultCmd := pipe.Get(ctx, keys.Result(id))
	_, err = pipe.Exec(ctx)
	if err != nil && !errors.Is(err, goredis.Nil) {
		return jobs.Snapshot{}, fmt.Errorf("load job snapshot: %w", err)
	}

	snapshot, err := buildSnapshot(id, meta, stateCmd.Val(), attemptCmd.Val(), bodyCmd.Val(), resultCmd.Val())
	if err != nil {
		return jobs.Snapshot{}, err
	}

	return snapshot, nil
}

func (t *Tracker) List(ctx context.Context, queue string, filter jobs.ListFilter) ([]jobs.Snapshot, error) {
	if t.client == nil || !t.client.IsEnabled() {
		return nil, errQueueDisabled
	}

	limit := filter.Limit
	if limit <= 0 || limit > t.runtime.RecentJobsLimit {
		limit = minInt64(t.runtime.RecentJobsLimit, 30)
	}

	keys := NewKeys(queue)
	ids, err := t.client.Raw().ZRevRange(ctx, keys.Jobs(), 0, limit-1).Result()
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	items := make([]jobs.Snapshot, 0, len(ids))
	for _, rawID := range ids {
		id, parseErr := jobs.ParseJobID(rawID)
		if parseErr != nil {
			continue
		}
		snapshot, getErr := t.Get(ctx, queue, id)
		if getErr != nil {
			continue
		}
		items = append(items, snapshot)
	}

	return items, nil
}

func (t *Tracker) Retry(ctx context.Context, queue string, id jobs.JobID) error {
	snapshot, err := t.Get(ctx, queue, id)
	if err != nil {
		return err
	}
	if snapshot.Status != jobs.StatusFailed && snapshot.Status != jobs.StatusDead && snapshot.Status != jobs.StatusSucceeded {
		return fmt.Errorf("job %s is not retryable from status %s", id.String(), snapshot.Status.String())
	}

	now := time.Now().UTC()
	keys := NewKeys(queue)
	pipe := t.client.Raw().Pipeline()
	pipe.ZRem(ctx, keys.Dead(), id.String())
	pipe.ZRem(ctx, keys.Delayed(), id.String())
	pipe.XAdd(ctx, &goredis.XAddArgs{Stream: keys.Stream(snapshot.Priority), MaxLen: t.runtime.MaxLenApprox, Approx: true, Values: map[string]any{"i": id.String()}})
	pipe.BitField(ctx, keys.State(), "SET", "u3", "#"+id.String(), int64(jobs.StatusQueued))
	pipe.HSet(ctx, keys.Meta(id), "e", "", "ra", "", "fa", "", "sa", "", "la", now.UnixMilli())
	_, err = pipe.Exec(ctx)
	if err != nil {
		metrics.NostrQueueRedisErrorsTotal.WithLabelValues(queue, "retry_manual").Inc()
		return fmt.Errorf("retry job %s: %w", id.String(), err)
	}

	return nil
}

func (t *Tracker) Cancel(ctx context.Context, queue string, id jobs.JobID) error {
	keys := NewKeys(queue)
	now := time.Now().UTC()
	_, err := t.scripts.cancel.Run(ctx, t.client.Raw(), []string{
		keys.State(),
		keys.Meta(id),
		keys.MetricsBucket(now),
		keys.Delayed(),
		keys.Dead(),
	}, id.String(), now.UnixMilli(), int(t.runtime.MetricsTTL.Seconds())).Result()
	if err != nil {
		metrics.NostrQueueLuaErrorsTotal.WithLabelValues(queue, "cancel").Inc()
		return fmt.Errorf("cancel job %s: %w", id.String(), err)
	}

	return nil
}

func (t *Tracker) Resume(ctx context.Context, queue string, id jobs.JobID) error {
	snapshot, err := t.Get(ctx, queue, id)
	if err != nil {
		return err
	}
	if snapshot.Status != jobs.StatusCanceled {
		return fmt.Errorf("job %s is not resumable from status %s", id.String(), snapshot.Status.String())
	}
	if snapshot.RunAt == nil && snapshot.StartedAt == nil {
		now := time.Now().UTC()
		keys := NewKeys(queue)
		_, err := t.client.Raw().TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
			pipe.BitField(ctx, keys.State(), "SET", "u3", "#"+id.String(), int64(jobs.StatusQueued))
			pipe.HSet(ctx, keys.Meta(id), "e", "", "fa", "", "la", now.UnixMilli())
			return nil
		})
		if err != nil {
			return fmt.Errorf("resume queued job %s: %w", id.String(), err)
		}
		return nil
	}
	return t.enqueueExisting(ctx, queue, id, snapshot.Priority)
}

func (t *Tracker) Delete(ctx context.Context, queue string, filter jobs.DeleteFilter) (int64, error) {
	snapshots, err := t.List(ctx, queue, jobs.ListFilter{Limit: t.runtime.RecentJobsLimit})
	if err != nil {
		return 0, err
	}

	keys := NewKeys(queue)
	var deleted int64
	for _, snapshot := range snapshots {
		if filter.JobName != "" && snapshot.Name != filter.JobName {
			continue
		}
		if len(filter.Statuses) > 0 && !slices.Contains(filter.Statuses, snapshot.Status) {
			continue
		}
		if !isTerminalJobStatus(snapshot.Status) {
			continue
		}

		_, err := t.client.Raw().TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
			pipe.ZRem(ctx, keys.Jobs(), snapshot.ID.String())
			pipe.ZRem(ctx, keys.Delayed(), snapshot.ID.String())
			pipe.ZRem(ctx, keys.Dead(), snapshot.ID.String())
			pipe.Del(ctx, keys.Body(snapshot.ID), keys.Meta(snapshot.ID), keys.Result(snapshot.ID))
			return nil
		})
		if err != nil {
			return deleted, fmt.Errorf("delete job %s: %w", snapshot.ID.String(), err)
		}
		deleted++
	}

	return deleted, nil
}

func (t *Tracker) enqueueExisting(ctx context.Context, queue string, id jobs.JobID, priority jobs.Priority) error {
	now := time.Now().UTC()
	keys := NewKeys(queue)
	streamKey := keys.Stream(priority)
	_, err := t.client.Raw().TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
		pipe.ZRem(ctx, keys.Dead(), id.String())
		pipe.ZRem(ctx, keys.Delayed(), id.String())
		pipe.XAdd(ctx, &goredis.XAddArgs{Stream: streamKey, MaxLen: t.runtime.MaxLenApprox, Approx: true, Values: map[string]any{"i": id.String()}})
		pipe.BitField(ctx, keys.State(), "SET", "u3", "#"+id.String(), int64(jobs.StatusQueued))
		pipe.HSet(ctx, keys.Meta(id), "e", "", "ra", "", "fa", "", "sa", "", "la", now.UnixMilli())
		return nil
	})
	if err != nil {
		metrics.NostrQueueRedisErrorsTotal.WithLabelValues(queue, "reenqueue_manual").Inc()
		return fmt.Errorf("reenqueue job %s: %w", id.String(), err)
	}
	return nil
}

func isTerminalJobStatus(status jobs.Status) bool {
	return status == jobs.StatusSucceeded || status == jobs.StatusFailed || status == jobs.StatusDead || status == jobs.StatusCanceled
}

func buildSnapshot(id jobs.JobID, meta map[string]string, stateVals, attemptVals []int64, bodyValue, resultValue string) (jobs.Snapshot, error) {
	status := jobs.StatusUnknown
	if len(stateVals) > 0 {
		status = jobs.Status(stateVals[0])
	}
	attempts := uint8(0)
	if len(attemptVals) > 0 {
		attempts = uint8(attemptVals[0])
	}
	createdAt, _ := parseUnixMilli(meta["ca"])
	startedAt, _ := parseOptionalUnixMilli(meta["sa"])
	finishedAt, _ := parseOptionalUnixMilli(meta["fa"])
	runAt, _ := parseOptionalUnixMilli(meta["ra"])
	maxAttempts, _ := strconv.Atoi(meta["ma"])

	snapshot := jobs.Snapshot{
		ID:          id,
		Queue:       meta["q"],
		Priority:    jobs.Priority(meta["p"]).Normalize(),
		Name:        meta["j"],
		Status:      status,
		Attempts:    attempts,
		MaxAttempts: uint8(maxAttempts),
		CreatedAt:   createdAt,
		StartedAt:   startedAt,
		FinishedAt:  finishedAt,
		RunAt:       runAt,
		LastError:   meta["e"],
		Payload:     decodePayload(bodyValue),
		Result:      decodeResult(resultValue),
	}

	return snapshot, nil
}

func decodePayload(bodyValue string) json.RawMessage {
	if bodyValue == "" {
		return nil
	}
	envelope, err := jobs.UnmarshalEnvelope([]byte(bodyValue))
	if err != nil {
		return nil
	}
	return envelope.Payload
}

func decodeResult(resultValue string) json.RawMessage {
	if resultValue == "" {
		return nil
	}
	return json.RawMessage(resultValue)
}

func parseUnixMilli(value string) (time.Time, error) {
	millis, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(millis).UTC(), nil
}

func parseOptionalUnixMilli(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := parseUnixMilli(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func minInt64(left, right int64) int64 {
	if left < right {
		return left
	}
	return right
}
