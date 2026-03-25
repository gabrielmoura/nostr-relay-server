package blossom

import (
	"context"
	"fmt"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/gabrielmoura/nostr-relay-server/config"
	db2 "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/pkg/magic"
	"github.com/gofiber/fiber/v2"
	"github.com/minio/sha256-simd"
	"github.com/tmthrgd/go-hex"
	"go.uber.org/zap"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// UploadHandler refatorado para Fiber
func UploadHandler(c *fiber.Ctx) error {
	if c.Method() != fiber.MethodPost && c.Method() != fiber.MethodPut {
		return c.Status(fiber.StatusMethodNotAllowed).SendString("Invalid request method")
	}

	startTime := time.Now()

	tags, pubKey, err := processAuth(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}
	hashToCheck := tags.GetFirst([]string{"x"}).Value()

	bodyBytes := c.Body()
	if len(bodyBytes) == 0 {
		return c.Status(fiber.StatusBadRequest).SendString("Empty request body")
	}

	mgl, err := magic.Lookup(bodyBytes)
	if err != nil {
		log.Logger.Error("Magic lookup error", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to detect file type")
	}

	mimeType := ternaryString(mgl.MIME, http.DetectContentType(bodyBytes))
	if !acceptMimeType(mimeType, config.Cfg.Store.AcceptedMimetypes) {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid file type")
	}

	hasher := sha256.New()
	hasher.Write(bodyBytes)
	hashBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)

	if config.Cfg.Ws.Auth && hashToCheck != "" && hashToCheck != hashString {
		return c.Status(fiber.StatusUnauthorized).SendString("Invalid file hash")
	}

	filePath := filepath.Join(blobPath, hashString)
	size := int64(len(bodyBytes))
	urlResponse := fmt.Sprintf("%s/%s", config.Cfg.Store.MediaPath, hashString)

	if _, err := os.Stat(filePath); err == nil {
		do, err := getFileExist(c.Context(), hashString)
		if err != nil {
			log.Logger.Error("File get error", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to get file")
		}
		return c.Status(fiber.StatusOK).JSON(do)
	}

	outFile, err := os.Create(filePath)
	if err != nil {
		log.Logger.Error("File creation error", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to create file on server")
	}
	defer outFile.Close()

	if _, err := outFile.Write(bodyBytes); err != nil {
		log.Logger.Error("File write error", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to write file content")
	}

	var rawTags []byte
	if len(tags) > 0 {
		json.Unmarshal(rawTags, &tags)
	}

	obj := &db2.Object{
		Hash:      hashString,
		MimeType:  mimeType,
		Size:      size,
		CreatedAt: time.Now(),
		Tags:      rawTags,
		PublicKey: pubKey,
	}

	if err := db.DbQueries.InsertObject(c.Context(), obj); err != nil {
		log.Logger.Error("DB save error", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to save file metadata")
	}

	response := &db2.ObjectResponse{
		Hash:      hashString,
		CreatedAt: obj.CreatedAt.Unix(),
		Url:       urlResponse,
		MimeType:  mimeType,
	}

	metrics.UploadCounter.Inc()
	metrics.HttpDuration.WithLabelValues(c.Path()).Observe(time.Since(startTime).Seconds())

	return c.Status(fiber.StatusOK).JSON(response)
}

func getFileExist(ctx context.Context, hash string) (*db2.ObjectResponse, error) {
	obj, err := db.DbQueries.GetObjectByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	urlResponse := fmt.Sprintf("%s/%s", config.Cfg.Store.MediaPath, hash)
	response := &db2.ObjectResponse{
		Hash:      hash,
		CreatedAt: obj.CreatedAt.Unix(),
		Url:       urlResponse,
		MimeType:  obj.MimeType,
	}

	return response, nil
}
