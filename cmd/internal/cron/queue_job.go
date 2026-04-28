package croncmd

import (
	"context"
	"fmt"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	jobcore "github.com/gabrielmoura/nostr-relay-server/internal/jobs"
)

const queueJobName = "cron.run"

type QueueJob struct {
	JobName string `json:"job_name"`
	Timeout int64  `json:"timeout_seconds"`
}

func (QueueJob) Name() string {
	return queueJobName
}

type QueueJobResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func RegisterQueueHandlers(registry *jobcore.MemoryRegistry) error {
	return jobcore.RegisterTyped(registry, queueJobName, func(ctx context.Context, job QueueJob) error {
		definitions := jobsFromConfig(config.Cfg)
		selected, err := filterJobs(definitions, []string{job.JobName})
		if err != nil {
			_ = jobcore.SetResult(ctx, QueueJobResult{Name: job.JobName, Status: "failed", Error: err.Error()})
			return err
		}
		if len(selected) != 1 {
			err := fmt.Errorf("cron job %q not found", job.JobName)
			_ = jobcore.SetResult(ctx, QueueJobResult{Name: job.JobName, Status: "failed", Error: err.Error()})
			return err
		}

		timeout := time.Duration(job.Timeout) * time.Second
		if timeout <= 0 {
			timeout = 30 * time.Minute
		}
		err = executeJob(ctx, selected[0], timeout)
		result := QueueJobResult{Name: job.JobName, Status: "succeeded"}
		if err != nil {
			result.Status = "failed"
			result.Error = err.Error()
		}
		_ = jobcore.SetResult(ctx, result)
		return err
	})
}

func dispatchJob(ctx context.Context, job jobDefinition, timeout time.Duration) error {
	service := jobcore.Default()
	if service == nil || service.Dispatcher == nil {
		return fmt.Errorf("cron queue runtime is not initialized")
	}
	queueName := config.Cfg.Jobs.Cron.Queue
	if queueName == "" {
		queueName = config.Cfg.Jobs.DefaultQueue
	}
	_, err := service.Dispatcher.Dispatch(
		ctx,
		QueueJob{JobName: job.Name, Timeout: int64(timeout / time.Second)},
		jobcore.WithQueue(queueName),
		jobcore.WithPriority(jobcore.Priority(config.Cfg.Jobs.Cron.Priority)),
		jobcore.WithTimeout(timeout),
	)
	return err
}

func useQueueExecution() bool {
	return config.Cfg != nil && config.Cfg.Jobs.Enabled && config.Cfg.Redis.Enabled && config.Cfg.Redis.Queue.Enabled
}
