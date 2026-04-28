package down

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	jobcore "github.com/gabrielmoura/nostr-relay-server/internal/jobs"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
)

type JobStatus string

const (
	JobQueued    JobStatus = "queued"
	JobRunning   JobStatus = "running"
	JobCompleted JobStatus = "completed"
	JobFailed    JobStatus = "failed"
)

type JobRequest struct {
	Relays    []string     `json:"relays"`
	PublicKey string       `json:"public_key,omitempty"`
	Kinds     []int        `json:"kinds,omitempty"`
	Filter    nostr.Filter `json:"filter,omitempty"`
	Timeout   int          `json:"timeout,omitempty"`
}

type Job struct {
	ID           string                `json:"id"`
	Status       JobStatus             `json:"status"`
	Message      string                `json:"message,omitempty"`
	CreatedAt    time.Time             `json:"created_at"`
	StartedAt    *time.Time            `json:"started_at,omitempty"`
	FinishedAt   *time.Time            `json:"finished_at,omitempty"`
	Relays       []string              `json:"relays"`
	PublicKey    string                `json:"public_key,omitempty"`
	Kinds        []int                 `json:"kinds,omitempty"`
	Timeout      int                   `json:"timeout"`
	Filter       nostr.Filter          `json:"filter"`
	FilterJSON   string                `json:"filter_json"`
	Summary      DownloadSummary       `json:"summary"`
	RelayResults []RelayDownloadResult `json:"relay_results"`
	Error        string                `json:"error,omitempty"`
}

type jobStore struct {
	mu    sync.RWMutex
	jobs  map[string]*Job
	order []string
}

var downloadJobs = &jobStore{jobs: map[string]*Job{}, order: []string{}}

func StartJob(req JobRequest) (*Job, error) {
	if useQueuedJobs() {
		service := jobcore.Default()
		if service == nil || service.Dispatcher == nil || service.Monitor == nil {
			return nil, fmt.Errorf("download queue runtime is not initialized")
		}

		queuedJob, _, err := prepareQueueJob(req)
		if err != nil {
			return nil, err
		}
		id, err := service.Dispatcher.Dispatch(
			context.Background(),
			queuedJob,
			jobcore.WithQueue(currentDownloadQueue()),
			jobcore.WithPriority(queueDownloadPriority()),
		)
		if err != nil {
			return nil, err
		}

		return buildQueuedJobResponse(id, queuedJob), nil
	}

	options, filterJSON, err := buildJobOptionsWithFilterJSON(req)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	job := &Job{
		ID:         fmt.Sprintf("dl_%d", now.UnixNano()),
		Status:     JobQueued,
		Message:    "download queued",
		CreatedAt:  now,
		Relays:     append([]string(nil), options.RelayURLs...),
		PublicKey:  req.PublicKey,
		Kinds:      append([]int(nil), req.Kinds...),
		Timeout:    req.Timeout,
		Filter:     options.Filter,
		FilterJSON: filterJSON,
	}
	downloadJobs.put(job)

	go runJob(job.ID, options)
	return cloneJob(job), nil
}

func ListJobs() []*Job {
	if useQueuedJobs() {
		service := jobcore.Default()
		if service == nil || service.Monitor == nil {
			return nil
		}
		snapshots, err := service.Monitor.List(context.Background(), currentDownloadQueue(), jobcore.ListFilter{Limit: 30})
		if err != nil {
			return nil
		}
		items := make([]*Job, 0, len(snapshots))
		for _, snapshot := range snapshots {
			item, convErr := snapshotToJob(snapshot)
			if convErr != nil {
				continue
			}
			items = append(items, item)
		}
		return items
	}

	return downloadJobs.list()
}

func GetJob(id string) (*Job, bool) {
	if useQueuedJobs() {
		service := jobcore.Default()
		if service == nil || service.Monitor == nil {
			return nil, false
		}
		parsedID, err := parsePublicJobID(id)
		if err != nil {
			return nil, false
		}
		snapshot, err := service.Monitor.Get(context.Background(), currentDownloadQueue(), parsedID)
		if err != nil {
			return nil, false
		}
		item, err := snapshotToJob(snapshot)
		if err != nil {
			return nil, false
		}
		return item, true
	}

	return downloadJobs.get(id)
}

func buildJobOptions(req JobRequest) (*DownloadOptions, error) {
	options, err := BuildOptions(CLIOptions{
		PublicKey: req.PublicKey,
		RelayURL:  req.Relays,
		Kinds:     req.Kinds,
		Timeout:   req.Timeout,
		Merge:     string(MergeOverride),
	})
	if err != nil {
		return nil, err
	}
	if len(req.Filter.IDs) > 0 || len(req.Filter.Kinds) > 0 || len(req.Filter.Authors) > 0 || len(req.Filter.Tags) > 0 || req.Filter.Search != "" || req.Filter.Since != nil || req.Filter.Until != nil || req.Filter.Limit != 0 {
		options.Filter = req.Filter
	}
	return options, nil
}

func buildJobOptionsWithFilterJSON(req JobRequest) (*DownloadOptions, string, error) {
	options, err := buildJobOptions(req)
	if err != nil {
		return nil, "", err
	}

	filterJSON := "{}"
	if payload, marshalErr := json.Marshal(options.Filter); marshalErr == nil {
		filterJSON = string(payload)
	}

	return options, filterJSON, nil
}

func useQueuedJobs() bool {
	return config.Cfg != nil && config.Cfg.Jobs.Enabled && config.Cfg.Redis.Enabled && config.Cfg.Redis.Queue.Enabled
}

func runJob(jobID string, options *DownloadOptions) {
	start := time.Now().UTC()
	downloadJobs.update(jobID, func(job *Job) {
		job.Status = JobRunning
		job.StartedAt = &start
		job.Message = "download in progress"
	})
	summary, results, err := DownloadRuntime(context.Background(), options)
	finish := time.Now().UTC()
	downloadJobs.update(jobID, func(job *Job) {
		job.Summary = summary
		job.RelayResults = results
		job.FinishedAt = &finish
		if err != nil {
			job.Status = JobFailed
			job.Error = err.Error()
			job.Message = "download failed"
			return
		}
		job.Status = JobCompleted
		job.Message = "download completed"
	})
}

func (s *jobStore) put(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
	s.order = append([]string{job.ID}, s.order...)
	if len(s.order) > 30 {
		toDrop := s.order[30:]
		s.order = s.order[:30]
		for _, id := range toDrop {
			delete(s.jobs, id)
		}
	}
}

func (s *jobStore) update(id string, mutate func(job *Job)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return
	}
	mutate(job)
}

func (s *jobStore) get(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	if !ok {
		return nil, false
	}
	return cloneJob(job), true
}

func (s *jobStore) list() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]*Job, 0, len(s.order))
	for _, id := range s.order {
		job, ok := s.jobs[id]
		if ok {
			items = append(items, cloneJob(job))
		}
	}
	return items
}

func cloneJob(job *Job) *Job {
	if job == nil {
		return nil
	}
	copyJob := *job
	copyJob.Relays = append([]string(nil), job.Relays...)
	copyJob.Kinds = append([]int(nil), job.Kinds...)
	copyJob.RelayResults = append([]RelayDownloadResult(nil), job.RelayResults...)
	if job.Filter.Tags != nil {
		copyJob.Filter.Tags = make(nostr.TagMap, len(job.Filter.Tags))
		for key, values := range job.Filter.Tags {
			copyJob.Filter.Tags[key] = append([]string(nil), values...)
		}
	}
	return &copyJob
}
