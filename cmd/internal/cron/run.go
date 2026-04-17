package croncmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

func Run(options *Options) error {
	if options == nil {
		return fmt.Errorf("cron options cannot be nil")
	}

	if err := config.LoadConfig(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	jobs := jobsFromConfig(config.Cfg)
	selectedJobs, err := filterJobs(jobs, options.Jobs)
	if err != nil {
		return err
	}

	if options.List {
		printJobs(selectedJobs)
		return nil
	}

	if !config.Cfg.Cron.Enabled {
		log.Init()
		log.Logger.Info("cron disabled by configuration", zap.Bool("cron.enabled", false))
		return nil
	}

	log.Init()

	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := db.Init(mainCtx); err != nil {
		return fmt.Errorf("init database: %w", err)
	}

	if options.RunOnce {
		return runOnce(mainCtx, selectedJobs, options.Timeout)
	}

	return runScheduler(mainCtx, cancel, selectedJobs, options.Timeout)
}

func printJobs(jobs []jobDefinition) {
	fmt.Println("Available cron jobs:")
	for _, job := range jobs {
		status := "disabled"
		if job.Enabled {
			status = "enabled"
		}
		fmt.Printf("- %s (status=%s, schedule=%q)\n", job.Name, status, job.Schedule)
	}
}

func runOnce(baseCtx context.Context, jobs []jobDefinition, timeout time.Duration) error {
	runnable := enabledJobs(jobs)
	if len(runnable) == 0 {
		return fmt.Errorf("no enabled cron jobs selected to run")
	}

	for _, job := range runnable {
		if err := executeJob(baseCtx, job, timeout); err != nil {
			return err
		}
	}

	return nil
}

func runScheduler(
	baseCtx context.Context,
	cancel context.CancelFunc,
	jobs []jobDefinition,
	timeout time.Duration,
) error {
	c := cron.New(cron.WithSeconds())
	registered := 0

	for _, job := range enabledJobs(jobs) {
		if job.Schedule == "" {
			log.Logger.Warn("cron job missing schedule and will be skipped", zap.String("job", job.Name))
			continue
		}

		jobCopy := job
		_, err := c.AddFunc(job.Schedule, func() {
			if err := executeJob(baseCtx, jobCopy, timeout); err != nil {
				log.Logger.Error("cron job failed", zap.String("job", jobCopy.Name), zap.Error(err))
			}
		})
		if err != nil {
			log.Logger.Error("failed to register cron job", zap.String("job", job.Name), zap.String("schedule", job.Schedule), zap.Error(err))
			continue
		}

		registered++
		log.Logger.Info("cron job registered", zap.String("job", job.Name), zap.String("schedule", job.Schedule))
	}

	if registered == 0 {
		return fmt.Errorf("no cron jobs registered")
	}

	c.Start()
	log.Logger.Info("cron scheduler started", zap.Int("jobs", registered))

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan

	log.Logger.Info("shutdown signal received for cron scheduler")
	cancel()
	stopCtx := c.Stop()
	<-stopCtx.Done()
	log.Logger.Info("cron scheduler stopped")

	return nil
}

func executeJob(baseCtx context.Context, job jobDefinition, timeout time.Duration) error {
	start := time.Now()
	jobCtx, cancel := context.WithTimeout(baseCtx, timeout)
	defer cancel()

	if err := job.Run(jobCtx); err != nil {
		return fmt.Errorf("run job %s: %w", job.Name, err)
	}

	log.Logger.Info("cron job finished", zap.String("job", job.Name), zap.Duration("duration", time.Since(start)))
	return nil
}

func enabledJobs(jobs []jobDefinition) []jobDefinition {
	enabled := make([]jobDefinition, 0, len(jobs))
	for _, job := range jobs {
		if !job.Enabled {
			continue
		}
		enabled = append(enabled, job)
	}
	return enabled
}
