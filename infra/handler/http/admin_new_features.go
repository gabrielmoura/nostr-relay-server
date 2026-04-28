package http

import (
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/down"
	"github.com/gabrielmoura/nostr-relay-server/internal/groups"
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
