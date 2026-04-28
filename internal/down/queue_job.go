package down

import (
	"context"
	"fmt"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	jobcore "github.com/gabrielmoura/nostr-relay-server/internal/jobs"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
)

const queueJobName = "download.events"

type QueueDownloadJob struct {
	Request    JobRequest   `json:"request"`
	Filter     nostr.Filter `json:"filter"`
	FilterJSON string       `json:"filter_json"`
}

func (QueueDownloadJob) Name() string {
	return queueJobName
}

type QueueDownloadResult struct {
	Summary      DownloadSummary       `json:"summary"`
	RelayResults []RelayDownloadResult `json:"relay_results"`
	Error        string                `json:"error,omitempty"`
}

func RegisterQueueHandlers(registry *jobcore.MemoryRegistry) error {
	return jobcore.RegisterTyped(registry, queueJobName, func(ctx context.Context, job QueueDownloadJob) error {
		options, err := buildOptionsForQueueJob(job)
		if err != nil {
			return err
		}

		summary, results, runErr := DownloadRuntime(ctx, options)
		result := QueueDownloadResult{
			Summary:      summary,
			RelayResults: results,
		}
		if runErr != nil {
			result.Error = runErr.Error()
		}
		_ = jobcore.SetResult(ctx, result)

		return runErr
	})
}

func prepareQueueJob(req JobRequest) (QueueDownloadJob, *DownloadOptions, error) {
	options, err := buildJobOptions(req)
	if err != nil {
		return QueueDownloadJob{}, nil, err
	}

	filterJSON := "{}"
	if payload, err := json.Marshal(options.Filter); err == nil {
		filterJSON = string(payload)
	}

	job := QueueDownloadJob{
		Request:    req,
		Filter:     options.Filter,
		FilterJSON: filterJSON,
	}

	return job, options, nil
}

func buildOptionsForQueueJob(job QueueDownloadJob) (*DownloadOptions, error) {
	options, err := buildJobOptions(job.Request)
	if err != nil {
		return nil, err
	}
	options.Filter = job.Filter
	return options, nil
}

func snapshotToJob(snapshot jobcore.Snapshot) (*Job, error) {
	var queuedJob QueueDownloadJob
	if len(snapshot.Payload) > 0 {
		if err := json.Unmarshal(snapshot.Payload, &queuedJob); err != nil {
			return nil, fmt.Errorf("decode queued download payload: %w", err)
		}
	}

	item := &Job{
		ID:         formatPublicJobID(snapshot.ID),
		Status:     mapSnapshotStatus(snapshot.Status),
		Message:    messageForSnapshot(snapshot.Status),
		CreatedAt:  snapshot.CreatedAt,
		StartedAt:  snapshot.StartedAt,
		FinishedAt: snapshot.FinishedAt,
		Relays:     append([]string(nil), queuedJob.Request.Relays...),
		PublicKey:  queuedJob.Request.PublicKey,
		Kinds:      append([]int(nil), queuedJob.Request.Kinds...),
		Timeout:    queuedJob.Request.Timeout,
		Filter:     queuedJob.Filter,
		FilterJSON: queuedJob.FilterJSON,
		Error:      snapshot.LastError,
	}

	if len(snapshot.Result) > 0 {
		var result QueueDownloadResult
		if err := json.Unmarshal(snapshot.Result, &result); err == nil {
			item.Summary = result.Summary
			item.RelayResults = append([]RelayDownloadResult(nil), result.RelayResults...)
			if result.Error != "" {
				item.Error = result.Error
			}
		}
	}

	return item, nil
}

func mapSnapshotStatus(status jobcore.Status) JobStatus {
	switch status {
	case jobcore.StatusRunning:
		return JobRunning
	case jobcore.StatusSucceeded:
		return JobCompleted
	case jobcore.StatusFailed, jobcore.StatusDead, jobcore.StatusCanceled:
		return JobFailed
	default:
		return JobQueued
	}
}

func messageForSnapshot(status jobcore.Status) string {
	switch status {
	case jobcore.StatusRunning:
		return "download in progress"
	case jobcore.StatusSucceeded:
		return "download completed"
	case jobcore.StatusFailed, jobcore.StatusDead:
		return "download failed"
	case jobcore.StatusCanceled:
		return "download canceled"
	case jobcore.StatusDelayed:
		return "download delayed"
	default:
		return "download queued"
	}
}

func formatPublicJobID(id jobcore.JobID) string {
	return "dl_" + id.String()
}

func parsePublicJobID(raw string) (jobcore.JobID, error) {
	value := raw
	if len(value) >= 3 && value[:3] == "dl_" {
		value = value[3:]
	}
	return jobcore.ParseJobID(value)
}

func currentDownloadQueue() string {
	if config := configOrNil(); config != nil && config.Jobs.Download.Queue != "" {
		return config.Jobs.Download.Queue
	}
	return "admin"
}

func queueDownloadPriority() jobcore.Priority {
	if config := configOrNil(); config != nil {
		return jobcore.Priority(config.Jobs.Download.Priority).Normalize()
	}
	return jobcore.PriorityNormal
}

func configOrNil() *config.Config {
	return config.Cfg
}

func queuedJobToResponse(queuedJob QueueDownloadJob, snapshot jobcore.Snapshot) *Job {
	item := &Job{
		ID:         formatPublicJobID(snapshot.ID),
		Status:     JobQueued,
		Message:    messageForSnapshot(snapshot.Status),
		CreatedAt:  snapshot.CreatedAt,
		StartedAt:  snapshot.StartedAt,
		FinishedAt: snapshot.FinishedAt,
		Relays:     append([]string(nil), queuedJob.Request.Relays...),
		PublicKey:  queuedJob.Request.PublicKey,
		Kinds:      append([]int(nil), queuedJob.Request.Kinds...),
		Timeout:    queuedJob.Request.Timeout,
		Filter:     queuedJob.Filter,
		FilterJSON: queuedJob.FilterJSON,
	}
	return item
}

func buildQueuedJobResponse(id jobcore.JobID, queuedJob QueueDownloadJob) *Job {
	now := time.Now().UTC()
	return &Job{
		ID:         formatPublicJobID(id),
		Status:     JobQueued,
		Message:    "download queued",
		CreatedAt:  now,
		Relays:     append([]string(nil), queuedJob.Request.Relays...),
		PublicKey:  queuedJob.Request.PublicKey,
		Kinds:      append([]int(nil), queuedJob.Request.Kinds...),
		Timeout:    queuedJob.Request.Timeout,
		Filter:     queuedJob.Filter,
		FilterJSON: queuedJob.FilterJSON,
	}
}
