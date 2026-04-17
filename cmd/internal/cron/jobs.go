package croncmd

import (
	"context"
	"fmt"

	"github.com/gabrielmoura/nostr-relay-server/config"
	cronjob "github.com/gabrielmoura/nostr-relay-server/infra/cron"
)

type jobDefinition struct {
	Name     string
	Schedule string
	Enabled  bool
	Run      func(context.Context) error
}

func jobsFromConfig(cfg *config.Config) []jobDefinition {
	return []jobDefinition{
		{
			Name:     "db_optimization",
			Schedule: cfg.Cron.DBOptimization.Schedule,
			Enabled:  cfg.Cron.DBOptimization.Enabled,
			Run:      cronjob.RunDBOptimization,
		},
		{
			Name:     "reported_events_fetch",
			Schedule: cfg.Cron.ReportedEventsFetch.Schedule,
			Enabled:  cfg.Cron.ReportedEventsFetch.Enabled,
			Run:      cronjob.FetchReportedEvents,
		},
		{
			Name:     "delete_old_events",
			Schedule: cfg.Cron.DeleteOldEvents.Schedule,
			Enabled:  cfg.Cron.DeleteOldEvents.Enabled,
			Run:      cronjob.DeleteOldEvent,
		},
		{
			Name:     "nip40",
			Schedule: cfg.Cron.NIP40.Schedule,
			Enabled:  cfg.Cron.NIP40.Enabled,
			Run:      cronjob.RunNIP40ExpirationCleanup,
		},
	}
}

func filterJobs(all []jobDefinition, selected []string) ([]jobDefinition, error) {
	if len(selected) == 0 {
		return all, nil
	}

	jobMap := make(map[string]jobDefinition, len(all))
	for _, job := range all {
		jobMap[job.Name] = job
	}

	filtered := make([]jobDefinition, 0, len(selected))
	for _, name := range selected {
		job, ok := jobMap[name]
		if !ok {
			return nil, fmt.Errorf("unknown cron job %q (available: db_optimization, reported_events_fetch, delete_old_events, nip40)", name)
		}
		filtered = append(filtered, job)
	}

	return filtered, nil
}
