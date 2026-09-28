package blossom

import (
	"errors"
	"fmt"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/blobstore"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"net/http"
	"time"
)

func BlobHandler(c *fiber.Ctx) error {
	startedAt := time.Now()
	statusCode := fiber.StatusOK
	errorCategory := ""
	defer func() {
		observeBlossomRequest("/blob/:id", c.Method(), startedAt)
		if statusCode >= 400 {
			observeBlossomError("/blob/:id", c.Method(), statusCode, errorCategory)
		}
	}()

	if c.Method() != fiber.MethodHead && c.Method() != fiber.MethodGet {
		statusCode = fiber.StatusMethodNotAllowed
		errorCategory = "invalid_method"
		return c.Status(fiber.StatusMethodNotAllowed).SendString("Invalid request method")
	}

	id := normalizeBlobID(c.Params("id"))
	if id == "" {
		statusCode = fiber.StatusBadRequest
		errorCategory = "invalid_request"
		return c.Status(fiber.StatusBadRequest).SendString("Invalid file ID")
	}

	o, err := db.DbQueries.GetObjectByHash(c.UserContext(), id)
	if err != nil {
		statusCode = fiber.StatusNotFound
		errorCategory = "not_found"
		return c.Status(fiber.StatusNotFound).SendString("File not found")
	}
	if o.Hash == "" {
		statusCode = fiber.StatusNotFound
		errorCategory = "not_found"
		return c.Status(fiber.StatusNotFound).SendString("File not found")
	}
	if o.Blocked && o.BlockedByReason == "" {
		statusCode = fiber.StatusForbidden
		errorCategory = "policy_denied"
		return c.Status(fiber.StatusForbidden).SendString("File is blocked")
	}
	if !o.ExpiresAt.IsZero() && time.Now().After(o.ExpiresAt) {
		go func() {
			store, storeErr := currentStore()
			if storeErr != nil {
				log.Logger.Error("Failed to access blob store for expiration", zap.Error(storeErr), zap.String("id", id))
			} else if deleteErr := store.Delete(c.UserContext(), id); deleteErr != nil && !errors.Is(deleteErr, blobstore.ErrNotFound) {
				log.Logger.Error("Failed to remove expired blob", zap.Error(deleteErr), zap.String("id", id))
			}
			if err := db.DbQueries.RemoveObject(c.Context(), id); err != nil {
				log.Logger.Error("Failed to remove object", zap.Error(err), zap.String("id", id))
			}
		}()
		statusCode = fiber.StatusGone
		errorCategory = "not_found"
		return c.Status(fiber.StatusGone).SendString("File has expired")
	}
	if o.BlockedByReason != "" {
		statusCode = fiber.StatusUnavailableForLegalReasons
		errorCategory = "policy_denied"
		return c.Status(fiber.StatusUnavailableForLegalReasons).SendString(o.BlockedByReason)
	}
	metrics.DownloadCounter.Inc()
	_ = db.DbQueries.RecordBlossomDownload(c.UserContext(), id, o.Size, time.Now().UTC())

	store, err := currentStore()
	if err != nil {
		log.Logger.Error("Blossom blob store unavailable", zap.Error(err))
		statusCode = fiber.StatusInternalServerError
		errorCategory = "storage_unavailable"
		return c.Status(fiber.StatusInternalServerError).SendString("Blob storage is unavailable")
	}
	fileInfo, err := store.Stat(c.UserContext(), id)
	if err != nil {
		if errors.Is(err, blobstore.ErrNotFound) {
			statusCode = fiber.StatusNotFound
			errorCategory = "not_found"
			return c.Status(fiber.StatusNotFound).SendString("File not found")
		}
		log.Logger.Error("Failed to retrieve blob info", zap.Error(err), zap.String("id", id))
		statusCode = fiber.StatusInternalServerError
		errorCategory = "storage_error"
		return c.Status(fiber.StatusInternalServerError).SendString("Unable to retrieve file info")
	}

	// Suporte a Range Requests
	r, err := c.Range(int(fileInfo.Size))
	if err != nil && c.Get("Range") != "" {
		statusCode = fiber.StatusRequestedRangeNotSatisfiable
		errorCategory = "range_invalid"
		return c.Status(fiber.StatusRequestedRangeNotSatisfiable).SendString("Invalid range")
	}

	c.Set("Cache-Control", "public, max-age=31536000, immutable")
	c.Set("Content-Type", o.MimeType)
	c.Set("Accept-Ranges", "bytes")
	c.Set("Last-Modified", o.CreatedAt.Format(http.TimeFormat))
	c.Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size))

	if len(r.Ranges) == 0 {
		if c.Method() == fiber.MethodHead {
			return c.SendStatus(fiber.StatusOK)
		}
		reader, _, err := store.Get(c.UserContext(), id, blobstore.ByteRange{Offset: 0, Length: -1})
		if err != nil {
			return sendBlobReadError(c, err, &statusCode, &errorCategory)
		}
		return c.Status(fiber.StatusOK).SendStream(reader, int(fileInfo.Size))
	}

	// Serve apenas o primeiro range, como no original
	start := r.Ranges[0].Start
	end := r.Ranges[0].End
	length := int64(end - start + 1)
	c.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileInfo.Size))
	c.Set("Content-Length", fmt.Sprintf("%d", length))

	if c.Method() == fiber.MethodHead {
		return c.Status(fiber.StatusPartialContent).SendStatus(fiber.StatusPartialContent)
	}
	reader, _, err := store.Get(c.UserContext(), id, blobstore.ByteRange{Offset: int64(start), Length: length})
	if err != nil {
		return sendBlobReadError(c, err, &statusCode, &errorCategory)
	}

	return c.Status(fiber.StatusPartialContent).SendStream(reader, int(length))
}

func sendBlobReadError(c *fiber.Ctx, err error, statusCode *int, errorCategory *string) error {
	if errors.Is(err, blobstore.ErrNotFound) {
		*statusCode = fiber.StatusNotFound
		*errorCategory = "not_found"
		return c.Status(fiber.StatusNotFound).SendString("File not found")
	}
	log.Logger.Error("Failed to read blob", zap.Error(err))
	*statusCode = fiber.StatusInternalServerError
	*errorCategory = "storage_error"
	return c.Status(fiber.StatusInternalServerError).SendString("Unable to read file")
}
