package store

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/gabrielmoura/nostr-relay-server/config"
	db2 "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	errors2 "github.com/gabrielmoura/nostr-relay-server/internal/errors"
	"github.com/gabrielmoura/nostr-relay-server/pkg/magic"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// processAuth adapta para fiber.Ctx e retorna hash esperado ou erro
func processAuth(c *fiber.Ctx) (string, error) {
	if config.Cfg.Ws.Auth {
		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Nostr ") {
			return "", errors2.ErrorAuthHeaderRequired
		}
		token := strings.TrimPrefix(authHeader, "Nostr ")

		decodedBytes, err := base64.StdEncoding.DecodeString(token)
		if err != nil {
			log.Logger.Error("Decode error", zap.Error(err))
			return "", errors2.ErrorDecodeAuthorization
		}

		var event nostr.Event
		if err := json.Unmarshal(decodedBytes, &event); err != nil {
			log.Logger.Error("Unmarshal error", zap.Error(err))
			return "", errors2.ErrorUnmarshalAuthorization
		}

		if event.Kind != nostr.KindBlobs {
			return "", errors2.ErrorInvalidEventKind
		}

		if ok, err := event.CheckSignature(); !ok || err != nil {
			return "", errors2.ErrorInvalidSignature
		}

		// TODO: validar pubkey autorizado para upload

		return event.Tags.GetFirst([]string{"x"}).Value(), nil
	}
	return "", nil
}

// UploadHandler refatorado para Fiber
func UploadHandler(c *fiber.Ctx) error {
	if c.Method() != fiber.MethodPost && c.Method() != fiber.MethodPut {
		return c.Status(fiber.StatusMethodNotAllowed).SendString("Invalid request method")
	}

	startTime := time.Now()

	hashToCheck, err := processAuth(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

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

	obj := &db2.Object{
		Hash:      hashString,
		MimeType:  mimeType,
		Size:      size,
		CreatedAt: time.Now(),
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
func ternaryString(condition string, fallback string) string {
	if condition != "" {
		return condition
	}
	return fallback
}
