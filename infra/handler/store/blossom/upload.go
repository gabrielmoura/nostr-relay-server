package blossom

import (
	"context"
	"github.com/gabrielmoura/nostr-relay-server/config"
	db2 "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	internalblossom "github.com/gabrielmoura/nostr-relay-server/internal/blossom"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	jobcore "github.com/gabrielmoura/nostr-relay-server/internal/jobs"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
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
	startedAt := time.Now()
	statusCode := fiber.StatusOK
	errorCategory := ""
	defer func() {
		observeBlossomRequest("/upload", c.Method(), startedAt)
		if statusCode >= 400 {
			observeBlossomError("/upload", c.Method(), statusCode, errorCategory)
		}
	}()

	if c.Method() != fiber.MethodPost && c.Method() != fiber.MethodPut {
		statusCode = fiber.StatusMethodNotAllowed
		errorCategory = "invalid_method"
		return c.Status(fiber.StatusMethodNotAllowed).SendString("Invalid request method")
	}

	tags, pubKey, err := processAuth(c)
	if err != nil {
		statusCode = fiber.StatusUnauthorized
		errorCategory = blossomErrorCategory(err, "auth_invalid")
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}
	hashToCheck := tags.GetFirst([]string{"x"}).Value()

	bodyBytes := c.Body()
	if len(bodyBytes) == 0 {
		statusCode = fiber.StatusBadRequest
		errorCategory = "empty_body"
		return c.Status(fiber.StatusBadRequest).SendString("Empty request body")
	}

	mgl, err := magic.Lookup(bodyBytes)
	if err != nil {
		log.Logger.Error("Magic lookup error", zap.Error(err))
		statusCode = fiber.StatusInternalServerError
		errorCategory = "internal"
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to detect file type")
	}

	mimeType := ternaryString(mgl.MIME, http.DetectContentType(bodyBytes))
	if !acceptMimeType(mimeType, config.Cfg.Store.AcceptedMimetypes) {
		statusCode = fiber.StatusBadRequest
		errorCategory = "mime_rejected"
		return c.Status(fiber.StatusBadRequest).SendString("Invalid file type")
	}

	hasher := sha256.New()
	hasher.Write(bodyBytes)
	hashBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)
	extension := dotExtension(mgl.Extension)

	if config.Cfg.Ws.Auth && hashToCheck != "" && hashToCheck != hashString {
		statusCode = fiber.StatusUnauthorized
		errorCategory = "hash_mismatch"
		return c.Status(fiber.StatusUnauthorized).SendString("Invalid file hash")
	}

	filePath := filepath.Join(blobPath, hashString)
	size := int64(len(bodyBytes))
	urlResponse := mediaURLWithExtension(hashString, extension)

	if _, err := os.Stat(filePath); err == nil {
		do, err := getFileExist(c.Context(), hashString)
		if err != nil {
			log.Logger.Error("File get error", zap.Error(err))
			statusCode = fiber.StatusInternalServerError
			errorCategory = "internal"
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to get file")
		}
		return c.Status(fiber.StatusOK).JSON(do)
	}

	policy, ok, err := db.DbQueries.GetBlossomServerPolicy(c.Context())
	if err != nil {
		log.Logger.Error("Blossom policy read error", zap.Error(err))
		statusCode = fiber.StatusInternalServerError
		errorCategory = "internal"
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read upload policy")
	}
	if !ok {
		policy = db2.BlossomServerPolicy{Mode: "free"}
	}
	quota, quotaOK, err := db.DbQueries.GetBlossomQuota(c.Context(), pubKey)
	if err != nil {
		log.Logger.Error("Blossom quota read error", zap.Error(err), zap.String("pubkey", pubKey))
		statusCode = fiber.StatusInternalServerError
		errorCategory = "internal"
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read upload quota")
	}
	usage, usageOK, err := db.DbQueries.GetBlossomUserUsage(c.Context(), pubKey)
	if err != nil {
		log.Logger.Error("Blossom usage read error", zap.Error(err), zap.String("pubkey", pubKey))
		statusCode = fiber.StatusInternalServerError
		errorCategory = "internal"
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read uploader usage")
	}
	var quotaRef *db2.BlossomPubkeyQuota
	if quotaOK {
		quotaRef = &quota
	}
	usedBytes := int64(0)
	if usageOK {
		usedBytes = usage.StorageUsedBytes
	}
	reviewState, blocked, blockedReason, policyErr := evaluateUploadPolicy(policy, quotaRef, usedBytes, size)
	if policyErr != nil {
		statusCode = policyErr.(*fiber.Error).Code
		errorCategory = blossomErrorCategory(policyErr, "policy_denied")
		return policyErr
	}

	outFile, err := os.Create(filePath)
	if err != nil {
		log.Logger.Error("File creation error", zap.Error(err))
		statusCode = fiber.StatusInternalServerError
		errorCategory = "internal"
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to create file on server")
	}
	defer outFile.Close()

	if _, err := outFile.Write(bodyBytes); err != nil {
		log.Logger.Error("File write error", zap.Error(err))
		statusCode = fiber.StatusInternalServerError
		errorCategory = "internal"
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to write file content")
	}

	rawTags, _ := json.Marshal(map[string]string{
		"x":   hashString,
		"m":   mimeType,
		"url": urlResponse,
	})

	obj := &db2.Object{
		Hash:            hashString,
		MimeType:        mimeType,
		Size:            size,
		CreatedAt:       time.Now(),
		Blocked:         blocked,
		BlockedByReason: blockedReason,
		Tags:            rawTags,
		PublicKey:       pubKey,
	}

	if err := db.DbQueries.InsertObject(c.Context(), obj); err != nil {
		log.Logger.Error("DB save error", zap.Error(err))
		statusCode = fiber.StatusInternalServerError
		errorCategory = "internal"
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to save file metadata")
	}
	if err := db.DbQueries.UpsertBlossomObjectAdmin(c.Context(), db2.UpsertBlossomObjectAdminParams{
		Hash:         hashString,
		Extension:    extension,
		IngressBytes: size,
		ExifStatus:   "pending",
		ReviewState:  reviewState,
		NIP94Tags:    rawTags,
	}); err != nil {
		log.Logger.Error("Blossom admin metadata save error", zap.Error(err))
	}
	if service := jobcore.Default(); service != nil && service.Dispatcher != nil {
		_, _ = service.Dispatcher.Dispatch(c.Context(), internalblossom.MediaJob{Hash: hashString, RequestedBy: pubKey}, jobcore.WithQueue(config.Cfg.Jobs.DefaultQueue))
	}

	response := &db2.ObjectResponse{
		Hash:      hashString,
		CreatedAt: obj.CreatedAt.Unix(),
		Url:       urlResponse,
		MimeType:  mimeType,
	}

	metrics.UploadCounter.Inc()

	return c.Status(fiber.StatusOK).JSON(response)
}

func getFileExist(ctx context.Context, hash string) (*db2.ObjectResponse, error) {
	obj, err := db.DbQueries.GetObjectByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	urlResponse := mediaURLWithExtension(hash, "")
	if adminObject, ok, err := db.DbQueries.GetBlossomObject(ctx, hash); err == nil && ok {
		urlResponse = mediaURLWithExtension(hash, adminObject.Extension)
	}
	response := &db2.ObjectResponse{
		Hash:      hash,
		CreatedAt: obj.CreatedAt.Unix(),
		Url:       urlResponse,
		MimeType:  obj.MimeType,
	}

	return response, nil
}

func dotExtension(value string) string {
	if value == "" {
		return ""
	}
	if value[0] == '.' {
		return value
	}
	return "." + value
}
