package http

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/down"
	"github.com/gabrielmoura/nostr-relay-server/internal/groups"
	jobcore "github.com/gabrielmoura/nostr-relay-server/internal/jobs"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/gabrielmoura/nostr-relay-server/internal/sync"
	"github.com/gabrielmoura/nostr-relay-server/internal/wot"
	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

type NegentropySyncRequest struct {
	Remote    string         `json:"remote"`
	Direction string         `json:"direction"` // "up", "down", "both"
	Filter    []nostr.Filter `json:"filter,omitempty"`
	Timeout   int            `json:"timeout,omitempty"`
}

type DownloadEventsRequest struct {
	Relays    []string     `json:"relays"`
	PublicKey string       `json:"public_key,omitempty"`
	Kinds     []int        `json:"kinds,omitempty"`
	Filter    nostr.Filter `json:"filter,omitempty"`
	Timeout   int          `json:"timeout,omitempty"`
}

type DownloadEventsResponse struct {
	Status  string   `json:"status"`
	JobID   string   `json:"job_id,omitempty"`
	Relays  []string `json:"relays"`
	Message string   `json:"message"`
}

type AdminGroupResponse struct {
	GroupID     string `json:"group_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
	Closed      bool   `json:"closed"`
	Hidden      bool   `json:"hidden"`
	MemberCount int    `json:"member_count"`
}

type AdminWoTSummaryResponse struct {
	TotalNodes     int      `json:"total_nodes"`
	TotalEdges     int      `json:"total_edges"`
	TrustedPubkeys []string `json:"trusted_pubkeys"`
	LastComputedAt string   `json:"last_computed_at,omitempty"`
}

type TrustedPubkeyRequest struct {
	Pubkey string `json:"pubkey"`
}

type AdminJob struct {
	ID          string          `json:"id"`
	Queue       string          `json:"queue"`
	Priority    string          `json:"priority"`
	JobName     string          `json:"job_name"`
	Status      string          `json:"status"`
	Attempts    uint8           `json:"attempts"`
	MaxAttempts uint8           `json:"max_attempts"`
	CreatedAt   string          `json:"created_at"`
	StartedAt   string          `json:"started_at,omitempty"`
	FinishedAt  string          `json:"finished_at,omitempty"`
	RunAt       string          `json:"run_at,omitempty"`
	LastError   string          `json:"last_error,omitempty"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	Result      json.RawMessage `json:"result,omitempty"`
}

type adminJobMutationRequest struct {
	Queue string `json:"queue"`
}

type adminJobDeleteResponse struct {
	Deleted int64 `json:"deleted"`
}

func NegentropySync() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req NegentropySyncRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		if req.Remote == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "remote relay URL is required"})
		}

		direction := sync.DirectionBoth
		switch strings.ToLower(req.Direction) {
		case "up":
			direction = sync.DirectionUp
		case "down":
			direction = sync.DirectionDown
		}

		timeout := 30 * time.Second
		if req.Timeout > 0 {
			timeout = time.Duration(req.Timeout) * time.Second
		}
		if config.Cfg != nil && config.Cfg.Jobs.Enabled && config.Cfg.Redis.Enabled && config.Cfg.Redis.Queue.Enabled {
			service := jobcore.Default()
			if service == nil || service.Dispatcher == nil {
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "sync queue runtime is not initialized"})
			}

			filterJSON := ""
			if len(req.Filter) > 0 {
				payload, err := json.Marshal(req.Filter)
				if err != nil {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid sync filter payload"})
				}
				filterJSON = string(payload)
			}

			id, err := service.Dispatcher.Dispatch(
				c.UserContext(),
				sync.QueueJob{
					Remote:     req.Remote,
					Direction:  string(direction),
					FilterJSON: filterJSON,
					TimeoutSec: int64(timeout / time.Second),
				},
				jobcore.WithQueue(config.Cfg.Jobs.Sync.Queue),
				jobcore.WithPriority(jobcore.Priority(config.Cfg.Jobs.Sync.Priority)),
			)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
			}

			return c.JSON(fiber.Map{
				"status":  "started",
				"remote":  req.Remote,
				"job_id":  id.String(),
				"message": "sync process started in background",
			})
		}

		go func() {
			cf := &sync.ConfSync{
				Remote:      req.Remote,
				Direction:   direction,
				LocalFilter: req.Filter,
				Timeout:     timeout,
			}
			sync.Sync(cf)
		}()

		return c.JSON(fiber.Map{
			"status":  "started",
			"remote":  req.Remote,
			"message": "sync process started in background",
		})
	}
}

func DownloadEvents() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req DownloadEventsRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		if len(req.Relays) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "at least one relay URL is required"})
		}

		job, err := down.StartJob(down.JobRequest(req))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(DownloadEventsResponse{Status: "started", JobID: job.ID, Relays: req.Relays, Message: "download process started in background"})
	}
}

func DownloadJobs() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"items": down.ListJobs()})
	}
}

func DownloadJobDetail() fiber.Handler {
	return func(c *fiber.Ctx) error {
		jobID := strings.TrimSpace(c.Params("jobId"))
		job, ok := down.GetJob(jobID)
		if !ok {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "download job not found"})
		}
		return c.JSON(job)
	}
}

func JobsList() fiber.Handler {
	return func(c *fiber.Ctx) error {
		service := jobcore.Default()
		if service == nil || service.Monitor == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "job runtime is not initialized"})
		}

		limit := adminLimit(c)
		offset := adminOffset(c)
		queueFilter := strings.TrimSpace(c.Query("queue"))
		jobNameFilter := strings.TrimSpace(c.Query("job_name"))
		statusFilter := strings.TrimSpace(c.Query("status"))

		snapshots, err := listAdminJobSnapshots(c.UserContext(), service.Monitor, queueFilter)
		if err != nil {
			return internalServerError(c, err)
		}

		items := make([]AdminJob, 0, len(snapshots))
		for _, snapshot := range snapshots {
			if jobNameFilter != "" && snapshot.Name != jobNameFilter {
				continue
			}
			if statusFilter != "" && snapshot.Status.String() != statusFilter {
				continue
			}
			items = append(items, adminJobFromSnapshot(snapshot))
		}

		total := len(items)
		return c.JSON(newAdminPage(paginate(items, limit, offset), total, limit, offset))
	}
}

func JobDetail() fiber.Handler {
	return func(c *fiber.Ctx) error {
		service := jobcore.Default()
		if service == nil || service.Monitor == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "job runtime is not initialized"})
		}

		jobID, err := jobcore.ParseJobID(c.Params("jobId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		snapshot, err := findAdminJobSnapshot(c.UserContext(), service.Monitor, strings.TrimSpace(c.Query("queue")), jobID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(adminJobFromSnapshot(snapshot))
	}
}

func RetryJob() fiber.Handler {
	return func(c *fiber.Ctx) error {
		service := jobcore.Default()
		if service == nil || service.Monitor == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "job runtime is not initialized"})
		}

		jobID, err := jobcore.ParseJobID(c.Params("jobId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		queueName, err := resolveAdminJobQueue(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		if err := service.Monitor.Retry(c.UserContext(), queueName, jobID); err != nil {
			return internalServerError(c, err)
		}

		return c.JSON(fiber.Map{"ok": true, "id": jobID.String(), "queue": queueName})
	}
}

func CancelJob() fiber.Handler {
	return func(c *fiber.Ctx) error {
		service := jobcore.Default()
		if service == nil || service.Monitor == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "job runtime is not initialized"})
		}

		jobID, err := jobcore.ParseJobID(c.Params("jobId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		queueName, err := resolveAdminJobQueue(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		if err := service.Monitor.Cancel(c.UserContext(), queueName, jobID); err != nil {
			return internalServerError(c, err)
		}

		return c.JSON(fiber.Map{"ok": true, "id": jobID.String(), "queue": queueName})
	}
}

func ResumeJob() fiber.Handler {
	return func(c *fiber.Ctx) error {
		service := jobcore.Default()
		if service == nil || service.Monitor == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "job runtime is not initialized"})
		}

		jobID, err := jobcore.ParseJobID(c.Params("jobId"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		queueName, err := resolveAdminJobQueue(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		if err := service.Monitor.Resume(c.UserContext(), queueName, jobID); err != nil {
			return internalServerError(c, err)
		}

		return c.JSON(fiber.Map{"ok": true, "id": jobID.String(), "queue": queueName})
	}
}

func DeleteJobsHistory() fiber.Handler {
	return func(c *fiber.Ctx) error {
		service := jobcore.Default()
		if service == nil || service.Monitor == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "job runtime is not initialized"})
		}

		jobName := strings.TrimSpace(c.Query("job_name"))
		if jobName == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "job_name is required"})
		}

		statuses := parseAdminJobStatuses(c.Request().URI().QueryArgs().PeekMulti("status"))
		if len(statuses) == 0 {
			statuses = []jobcore.Status{jobcore.StatusSucceeded, jobcore.StatusFailed, jobcore.StatusDead, jobcore.StatusCanceled}
		}

		var deleted int64
		for _, queueName := range adminJobQueues(strings.TrimSpace(c.Query("queue"))) {
			count, err := service.Monitor.Delete(c.UserContext(), queueName, jobcore.DeleteFilter{JobName: jobName, Statuses: statuses})
			if err != nil {
				return internalServerError(c, err)
			}
			deleted += count
		}

		return c.JSON(adminJobDeleteResponse{Deleted: deleted})
	}
}

func adminJobFromSnapshot(snapshot jobcore.Snapshot) AdminJob {
	item := AdminJob{
		ID:          snapshot.ID.String(),
		Queue:       snapshot.Queue,
		Priority:    string(snapshot.Priority),
		JobName:     snapshot.Name,
		Status:      snapshot.Status.String(),
		Attempts:    snapshot.Attempts,
		MaxAttempts: snapshot.MaxAttempts,
		CreatedAt:   formatTime(snapshot.CreatedAt),
		LastError:   snapshot.LastError,
		Payload:     snapshot.Payload,
		Result:      snapshot.Result,
	}
	if snapshot.StartedAt != nil {
		item.StartedAt = formatTime(*snapshot.StartedAt)
	}
	if snapshot.FinishedAt != nil {
		item.FinishedAt = formatTime(*snapshot.FinishedAt)
	}
	if snapshot.RunAt != nil {
		item.RunAt = formatTime(*snapshot.RunAt)
	}
	return item
}

func listAdminJobSnapshots(ctx context.Context, monitor jobcore.Monitor, queueFilter string) ([]jobcore.Snapshot, error) {
	queues := adminJobQueues(queueFilter)
	items := make([]jobcore.Snapshot, 0)
	seen := make(map[string]struct{})
	for _, queueName := range queues {
		snapshots, err := monitor.List(ctx, queueName, jobcore.ListFilter{Limit: int64(maxAdminLimit)})
		if err != nil {
			return nil, fmt.Errorf("list jobs for queue %s: %w", queueName, err)
		}
		for _, snapshot := range snapshots {
			key := snapshot.Queue + ":" + snapshot.ID.String()
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			items = append(items, snapshot)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

func findAdminJobSnapshot(ctx context.Context, monitor jobcore.Monitor, queueFilter string, jobID jobcore.JobID) (jobcore.Snapshot, error) {
	for _, queueName := range adminJobQueues(queueFilter) {
		snapshot, err := monitor.Get(ctx, queueName, jobID)
		if err == nil {
			return snapshot, nil
		}
	}
	return jobcore.Snapshot{}, fmt.Errorf("job %s not found", jobID.String())
}

func adminJobQueues(queueFilter string) []string {
	if queueFilter != "" {
		return []string{queueFilter}
	}
	if config.Cfg == nil {
		return nil
	}
	values := []string{config.Cfg.Jobs.Download.Queue, config.Cfg.Jobs.Sync.Queue, config.Cfg.Jobs.Cron.Queue, config.Cfg.Jobs.DefaultQueue}
	seen := make(map[string]struct{}, len(values))
	queues := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		queues = append(queues, value)
	}
	return queues
}

func resolveAdminJobQueue(c *fiber.Ctx) (string, error) {
	queueName := strings.TrimSpace(c.Query("queue"))
	if queueName != "" {
		return queueName, nil
	}
	var body adminJobMutationRequest
	if err := parseAdminJSONBody(c, &body); err != nil {
		return "", err
	}
	queueName = strings.TrimSpace(body.Queue)
	if queueName == "" {
		return "", fmt.Errorf("queue is required")
	}
	return queueName, nil
}

func parseAdminJobStatuses(values [][]byte) []jobcore.Status {
	statuses := make([]jobcore.Status, 0, len(values))
	for _, raw := range values {
		switch strings.TrimSpace(string(raw)) {
		case jobcore.StatusSucceeded.String():
			statuses = append(statuses, jobcore.StatusSucceeded)
		case jobcore.StatusFailed.String():
			statuses = append(statuses, jobcore.StatusFailed)
		case jobcore.StatusDead.String():
			statuses = append(statuses, jobcore.StatusDead)
		case jobcore.StatusCanceled.String():
			statuses = append(statuses, jobcore.StatusCanceled)
		}
	}
	return statuses
}

func ListGroups() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !groups.Enabled() {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "NIP-29 groups module is disabled"})
		}

		ctx := c.UserContext()
		limit := adminLimit(c)
		offset := adminOffset(c)

		scope := groups.M.GetRelayScope()

		dbGroups, total, err := db.DbQueries.ListNIP29Groups(ctx, scope, int32(limit), int32(offset))
		if err != nil {
			return internalServerError(c, err)
		}

		items := make([]AdminGroupResponse, 0, len(dbGroups))
		for _, g := range dbGroups {
			items = append(items, AdminGroupResponse{
				GroupID:     g.GroupID,
				Name:        g.Name,
				Description: g.About,
				Private:     g.Private,
				Closed:      g.Closed,
				Hidden:      g.Hidden,
				MemberCount: int(g.MemberCount),
			})
		}

		return c.JSON(newAdminPage(items, int(total), limit, offset))
	}
}

func WoTSummary() fiber.Handler {
	return func(c *fiber.Ctx) error {
		summary := wot.GetSummary()

		trusted := config.Cfg.WoT.TrustedPubkeys
		if trusted == nil {
			trusted = []string{}
		}

		return c.JSON(AdminWoTSummaryResponse{
			TotalNodes:     summary.Nodes,
			TotalEdges:     summary.Edges,
			TrustedPubkeys: trusted,
			LastComputedAt: summary.LastComputed.Format(time.RFC3339),
		})
	}
}

func AddTrustedPubkey() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req TrustedPubkeyRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		pubkey, err := normalizePublicKey(req.Pubkey)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid pubkey"})
		}

		found := false
		for _, p := range config.Cfg.WoT.TrustedPubkeys {
			if p == pubkey {
				found = true
				break
			}
		}

		if !found {
			config.Cfg.WoT.TrustedPubkeys = append(config.Cfg.WoT.TrustedPubkeys, pubkey)
			log.Logger.Info("added trusted pubkey to WoT config", zap.String("pubkey", pubkey))
		}

		wot.ScheduleRecompute()

		return c.JSON(fiber.Map{"pubkey": pubkey, "added": true})
	}
}

func RemoveTrustedPubkey() fiber.Handler {
	return func(c *fiber.Ctx) error {
		pubkey, err := normalizePublicKey(c.Params("pubkey"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid pubkey"})
		}

		newTrusted := make([]string, 0)
		removed := false
		for _, p := range config.Cfg.WoT.TrustedPubkeys {
			if p != pubkey {
				newTrusted = append(newTrusted, p)
			} else {
				removed = true
			}
		}

		if removed {
			config.Cfg.WoT.TrustedPubkeys = newTrusted
			log.Logger.Info("removed trusted pubkey from WoT config", zap.String("pubkey", pubkey))
			wot.ScheduleRecompute()
		}

		return c.JSON(fiber.Map{"pubkey": pubkey, "removed": removed})
	}
}
