package http

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/config"
	dbmodel "github.com/gabrielmoura/nostr-relay-server/infra/db"
	internalblossom "github.com/gabrielmoura/nostr-relay-server/internal/blossom"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	jobcore "github.com/gabrielmoura/nostr-relay-server/internal/jobs"
	"github.com/gofiber/fiber/v2"
)

func BlossomOverview() fiber.Handler {
	return func(c *fiber.Ctx) error {
		stats, err := db.DbQueries.GetBlossomOverviewStats(c.UserContext())
		if err != nil {
			return internalServerError(c, err)
		}
		usedBytes, freeBytes, err := blossomFileUsage()
		if err != nil {
			return internalServerError(c, err)
		}
		policy, ok, err := db.DbQueries.GetBlossomServerPolicy(c.UserContext())
		if err != nil {
			return internalServerError(c, err)
		}
		if !ok {
			policy = defaultBlossomServerPolicy()
		}
		runningWorkers, failedWorkers, err := blossomWorkerSummary()
		if err != nil {
			return internalServerError(c, err)
		}

		var response adminBlossomOverviewResponse
		response.Storage.UsedBytes = maxInt64(usedBytes, stats.UsedBytes)
		response.Storage.FreeBytes = freeBytes
		if response.Storage.UsedBytes+response.Storage.FreeBytes > 0 {
			response.Storage.UsedPercent = float64(response.Storage.UsedBytes) * 100 / float64(response.Storage.UsedBytes+response.Storage.FreeBytes)
		}
		response.Objects.Total = stats.TotalObjects
		response.Objects.Flagged = stats.FlaggedObjects
		response.Objects.PendingReview = stats.PendingReview
		response.Traffic.MonthlyIngressBytes = stats.MonthlyIngress
		response.Traffic.MonthlyEgressBytes = stats.MonthlyEgress
		response.Users.Active = stats.ActiveUsers
		response.Users.Whitelisted = stats.WhitelistedUsers
		response.Workers.Running = runningWorkers
		response.Workers.Failed = failedWorkers
		response.Policy = mapBlossomPolicyResponse(policy)
		if response.Storage.UsedPercent >= 90 {
			response.Alerts = append(response.Alerts, adminBlossomAlertResponse{Level: "warning", Code: "disk-usage", Message: "Disk 90% cheio"})
		}
		if failedWorkers > 0 {
			response.Alerts = append(response.Alerts, adminBlossomAlertResponse{Level: "danger", Code: "worker-failure", Message: "Falha em jobs Blossom de background"})
		}
		if stats.PendingReview > 0 {
			response.Alerts = append(response.Alerts, adminBlossomAlertResponse{Level: "info", Code: "review-queue", Message: "Arquivos aguardando revisão manual"})
		}
		return c.JSON(response)
	}
}

func BlossomObjects() fiber.Handler {
	return func(c *fiber.Ctx) error {
		items, total, err := db.DbQueries.ListBlossomObjects(c.UserContext(), dbmodel.BlossomObjectFilters{
			SHA256:        strings.TrimSpace(c.Query("sha256")),
			MIMEType:      strings.TrimSpace(c.Query("mime_type")),
			Extension:     strings.TrimSpace(c.Query("extension")),
			ReviewState:   strings.TrimSpace(c.Query("review_state")),
			Pubkey:        strings.TrimSpace(c.Query("pubkey")),
			UploaderQuery: strings.TrimSpace(c.Query("uploader_q")),
			Query:         strings.TrimSpace(c.Query("q")),
		}, adminLimit(c), adminOffset(c))
		if err != nil {
			return internalServerError(c, err)
		}
		response := make([]adminBlossomObjectResponse, 0, len(items))
		for _, item := range items {
			response = append(response, mapBlossomObjectResponse(item))
		}
		return c.JSON(newAdminPage(response, int(total), adminLimit(c), adminOffset(c)))
	}
}

func BlossomObjectDetail() fiber.Handler {
	return func(c *fiber.Ctx) error {
		item, ok, err := db.DbQueries.GetBlossomObject(c.UserContext(), strings.TrimSpace(c.Params("hash")))
		if err != nil {
			return internalServerError(c, err)
		}
		if !ok {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "blossom object not found"})
		}
		response := mapBlossomObjectResponse(item)
		reportCount, err := db.DbQueries.CountOpenBlossomReportsByHash(c.UserContext(), item.Hash)
		if err != nil {
			return internalServerError(c, err)
		}
		response.ReportCount = reportCount
		return c.JSON(response)
	}
}

func BlossomBulkReview() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req adminBlossomBulkReviewRequest
		if err := parseAdminJSONBody(c, &req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		if len(req.Hashes) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "hashes are required"})
		}
		switch strings.TrimSpace(req.Action) {
		case "approve":
			updated, err := db.DbQueries.UpdateBlossomObjectsReviewState(c.UserContext(), req.Hashes, "approved", "")
			if err != nil {
				return internalServerError(c, err)
			}
			if err := db.DbQueries.UpdateObjectsBlockedState(c.UserContext(), req.Hashes, false, ""); err != nil {
				return internalServerError(c, err)
			}
			for _, hash := range req.Hashes {
				_ = internalblossom.RecordAudit(c.UserContext(), blossomActorPubkey(), "object.approve", "object", hash, map[string]string{"reason": req.Reason})
			}
			return c.JSON(adminBlossomBulkReviewResponse{OK: true, Updated: updated})
		case "hard_delete":
			updated, err := hardDeleteBlossomObjects(c.UserContext(), req.Hashes, req.Reason)
			if err != nil {
				return internalServerError(c, err)
			}
			return c.JSON(adminBlossomBulkReviewResponse{OK: true, Updated: updated})
		case "requeue_optimization":
			updated, err := requeueBlossomOptimization(c.UserContext(), req.Hashes)
			if err != nil {
				return internalServerError(c, err)
			}
			return c.JSON(adminBlossomBulkReviewResponse{OK: true, Updated: updated})
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid bulk review action"})
		}
	}
}

func hardDeleteBlossomObjects(ctx context.Context, hashes []string, reason string) (int64, error) {
	for _, hash := range hashes {
		if err := os.Remove(filepath.Join("files", hash)); err != nil && !os.IsNotExist(err) {
			return 0, fmt.Errorf("remove file %s: %w", hash, err)
		}
	}
	deleted, err := db.DbQueries.DeleteBlossomObjects(ctx, hashes)
	if err != nil {
		return 0, err
	}
	for _, hash := range hashes {
		_ = internalblossom.RecordAudit(ctx, blossomActorPubkey(), "object.hard_delete", "object", hash, map[string]string{"reason": reason})
	}
	return deleted, nil
}

func requeueBlossomOptimization(ctx context.Context, hashes []string) (int64, error) {
	service := jobcore.Default()
	if service == nil || service.Dispatcher == nil {
		return 0, fmt.Errorf("job runtime is not initialized")
	}
	for _, hash := range hashes {
		if _, err := service.Dispatcher.Dispatch(ctx, internalblossom.MediaJob{Hash: hash, RequestedBy: blossomActorPubkey()}, jobcore.WithQueue(config.Cfg.Jobs.DefaultQueue)); err != nil {
			return 0, err
		}
		_ = internalblossom.RecordAudit(ctx, blossomActorPubkey(), "object.requeue_optimization", "object", hash, map[string]string{"hash": hash})
	}
	return int64(len(hashes)), nil
}

func maxInt64(values ...int64) int64 {
	var best int64
	for _, value := range values {
		if value > best {
			best = value
		}
	}
	return best
}
