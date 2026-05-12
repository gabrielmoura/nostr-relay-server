package sync

import (
	"context"

	jobcore "github.com/gabrielmoura/nostr-relay-server/internal/jobs"
)

const queueJobName = "sync.negentropy"

type QueueJob struct {
	Remote     string `json:"remote"`
	Direction  string `json:"direction"`
	PublicKey  string `json:"public_key,omitempty"`
	FilterJSON string `json:"filter_json,omitempty"`
	TimeoutSec int64  `json:"timeout_seconds,omitempty"`
}

func (QueueJob) Name() string {
	return queueJobName
}

type QueueJobResult struct {
	Remote      string            `json:"remote"`
	Direction   string            `json:"direction"`
	Status      string            `json:"status"`
	Error       string            `json:"error,omitempty"`
	Filter      []any             `json:"filter,omitempty"`
	Rejections  []RejectionInfo `json:"rejections,omitempty"`
}

func RegisterQueueHandlers(registry *jobcore.MemoryRegistry) error {
	return jobcore.RegisterTyped(registry, queueJobName, func(ctx context.Context, job QueueJob) error {
		cfg, err := BuildConfig(CLIOptions{
			Remote:    job.Remote,
			Direction: job.Direction,
			Pk:        job.PublicKey,
			Filter:    job.FilterJSON,
			Timeout:   job.TimeoutSec,
		})
		if err != nil {
			_ = jobcore.SetResult(ctx, QueueJobResult{Remote: job.Remote, Direction: job.Direction, Status: "failed", Error: err.Error()})
			return err
		}

		var syncResult SyncResult
		executeSync(cfg, &syncResult)

		result := QueueJobResult{
			Remote:      job.Remote,
			Direction:   job.Direction,
			Status:      "succeeded",
			Filter:      syncResult.Filter,
			Rejections:  syncResult.Rejections,
		}
		if syncResult.Error != nil {
			result.Status = "failed"
			result.Error = syncResult.Error.Error()
		}
		_ = jobcore.SetResult(ctx, result)
		return syncResult.Error
	})
}
