package blossom

import (
	"errors"
	"strings"
	"time"

	storedb "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
)

func ReportHandler(c *fiber.Ctx) error {
	startedAt := time.Now()
	statusCode := fiber.StatusAccepted
	errorCategory := ""
	defer func() {
		observeBlossomRequest("/report", c.Method(), startedAt)
		if statusCode >= 400 {
			observeBlossomError("/report", c.Method(), statusCode, errorCategory)
		}
	}()

	if c.Method() != fiber.MethodPut {
		statusCode = fiber.StatusMethodNotAllowed
		errorCategory = "invalid_method"
		return c.Status(fiber.StatusMethodNotAllowed).SendString("Invalid request method")
	}

	var event nostr.Event
	if err := json.Unmarshal(c.Body(), &event); err != nil {
		statusCode = fiber.StatusBadRequest
		errorCategory = "invalid_request"
		return c.Status(fiber.StatusBadRequest).SendString("Invalid report payload")
	}
	if event.Kind != 1984 {
		statusCode = fiber.StatusBadRequest
		errorCategory = "auth_invalid"
		return c.Status(fiber.StatusBadRequest).SendString("Invalid report event kind")
	}
	if ok, err := event.CheckSignature(); !ok || err != nil {
		statusCode = fiber.StatusUnauthorized
		errorCategory = "auth_signature_invalid"
		return c.Status(fiber.StatusUnauthorized).SendString("Invalid report signature")
	}

	hashes := collectReportedBlobHashes(event.Tags)
	if len(hashes) == 0 {
		statusCode = fiber.StatusBadRequest
		errorCategory = "invalid_request"
		return c.Status(fiber.StatusBadRequest).SendString("Missing blob hash report tags")
	}

	if err := db.DbQueries.InsertEvent(c.UserContext(), &event); err != nil && !errors.Is(err, storedb.ErrDupEvent) {
		statusCode = fiber.StatusInternalServerError
		errorCategory = "internal"
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to store report event")
	}

	acceptedHashes := make([]string, 0, len(hashes))
	for _, hash := range hashes {
		obj, err := db.DbQueries.GetObjectByHash(c.UserContext(), hash)
		if err != nil {
			statusCode = fiber.StatusInternalServerError
			errorCategory = "internal"
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to inspect reported blob")
		}
		if obj.Hash == "" {
			continue
		}
		acceptedHashes = append(acceptedHashes, hash)
	}
	if len(acceptedHashes) == 0 {
		statusCode = fiber.StatusNotFound
		errorCategory = "not_found"
		return c.Status(fiber.StatusNotFound).SendString("No reported blob found on this server")
	}

	targetEventID, targetPubkey, reportType := extractReportTargets(event.Tags)
	if err := db.DbQueries.InsertBlossomReviewReports(c.UserContext(), event.ID, event.PubKey, targetEventID, targetPubkey, reportType, strings.TrimSpace(event.Content), acceptedHashes, timeFromUnix(event.CreatedAt)); err != nil {
		statusCode = fiber.StatusInternalServerError
		errorCategory = "internal"
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to persist blob report")
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"ok": true, "hashes": acceptedHashes})
}

func collectReportedBlobHashes(tags nostr.Tags) []string {
	seen := make(map[string]struct{}, len(tags))
	items := make([]string, 0, len(tags))
	for _, tag := range tags {
		if len(tag) < 2 || tag[0] != "x" {
			continue
		}
		hash := strings.TrimSpace(tag[1])
		if len(hash) != 64 {
			continue
		}
		if _, ok := seen[hash]; ok {
			continue
		}
		seen[hash] = struct{}{}
		items = append(items, hash)
	}
	return items
}

func extractReportTargets(tags nostr.Tags) (targetEventID string, targetPubkey string, reportType string) {
	for _, tag := range tags {
		if len(tag) < 2 {
			continue
		}
		switch tag[0] {
		case "e":
			if targetEventID == "" {
				targetEventID = strings.TrimSpace(tag[1])
			}
			if len(tag) > 2 && reportType == "" {
				reportType = strings.TrimSpace(tag[2])
			}
		case "p":
			if targetPubkey == "" {
				targetPubkey = strings.TrimSpace(tag[1])
			}
			if len(tag) > 2 && reportType == "" {
				reportType = strings.TrimSpace(tag[2])
			}
		case "x":
			if len(tag) > 2 && reportType == "" {
				reportType = strings.TrimSpace(tag[2])
			}
		}
	}
	return targetEventID, targetPubkey, reportType
}
