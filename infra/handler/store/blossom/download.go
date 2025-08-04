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
	if c.Method() != fiber.MethodHead && c.Method() != fiber.MethodGet {
		return c.Status(fiber.StatusMethodNotAllowed).SendString("Invalid request method")
	}

	id := c.Params("id") // Assumindo rota "/blob/:id"
	if id == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid file ID")
	}

	filePath := filepath.Join(blobPath, id)
	if c.Method() == fiber.MethodHead {
		if _, err := os.Stat(filePath); err != nil {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return c.SendStatus(fiber.StatusOK)
	}

	o, err := db.DbQueries.GetObjectByHash(c.UserContext(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("File not found")
	}
	if o.Blocked && o.BlockedByReason == "" {
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
		return c.Status(fiber.StatusGone).SendString("File has expired")
	}
	if o.BlockedByReason != "" {
		return c.Status(fiber.StatusUnavailableForLegalReasons).SendString(o.BlockedByReason)
	}
	metrics.DownloadCounter.Inc()

	file, err := os.Open(filePath)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("File not found")
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Unable to retrieve file info")
	}

	// Suporte a Range Requests
	r, err := c.Range(int(fileInfo.Size()))
	if err != nil && c.Get("Range") != "" {
		return c.Status(fiber.StatusRequestedRangeNotSatisfiable).SendString("Invalid range")
	}

	c.Set("Cache-Control", "public, max-age=31536000, immutable")
	//c.Type(o.MimeType)
	c.Set("Content-Type", o.MimeType)
	c.Set("Last-Modified", o.CreatedAt.Format(http.TimeFormat))

	if len(r.Ranges) == 0 {
		c.Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
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
		return c.Status(fiber.StatusInternalServerError).SendString("Error reading file section")
	}

	return c.Status(fiber.StatusPartialContent).Send(data)
}
