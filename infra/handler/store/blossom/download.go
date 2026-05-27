package blossom

import (
	"fmt"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

	filePath := filepath.Join(blobPath, id)

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
			if err := os.Remove(filePath); err != nil {
				log.Logger.Error("Failed to remove file", zap.Error(err), zap.String("filePath", filePath))
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

	file, err := os.Open(filePath)
	if err != nil {
		statusCode = fiber.StatusNotFound
		errorCategory = "not_found"
		return c.Status(fiber.StatusNotFound).SendString("File not found")
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		statusCode = fiber.StatusInternalServerError
		errorCategory = "internal"
		return c.Status(fiber.StatusInternalServerError).SendString("Unable to retrieve file info")
	}

	// Suporte a Range Requests
	r, err := c.Range(int(fileInfo.Size()))
	if err != nil && c.Get("Range") != "" {
		statusCode = fiber.StatusRequestedRangeNotSatisfiable
		errorCategory = "range_invalid"
		return c.Status(fiber.StatusRequestedRangeNotSatisfiable).SendString("Invalid range")
	}

	c.Set("Cache-Control", "public, max-age=31536000, immutable")
	c.Set("Content-Type", o.MimeType)
	c.Set("Accept-Ranges", "bytes")
	c.Set("Last-Modified", o.CreatedAt.Format(http.TimeFormat))
	c.Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	if c.Method() == fiber.MethodHead {
		return c.SendStatus(fiber.StatusOK)
	}

	if len(r.Ranges) == 0 {
		return c.Status(fiber.StatusOK).SendFile(filePath, false)
	}

	// Serve apenas o primeiro range, como no original
	start := r.Ranges[0].Start
	end := r.Ranges[0].End
	length := end - start + 1

	sectionReader := io.NewSectionReader(file, int64(start), int64(length))
	c.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileInfo.Size()))
	c.Set("Content-Length", fmt.Sprintf("%d", length))

	data := make([]byte, length)
	if _, err := sectionReader.Read(data); err != nil && err != io.EOF {
		statusCode = fiber.StatusInternalServerError
		errorCategory = "internal"
		return c.Status(fiber.StatusInternalServerError).SendString("Error reading file section")
	}

	return c.Status(fiber.StatusPartialContent).Send(data)
}
