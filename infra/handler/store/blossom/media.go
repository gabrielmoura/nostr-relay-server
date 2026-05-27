package blossom

import (
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	db2 "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	internalblossom "github.com/gabrielmoura/nostr-relay-server/internal/blossom"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	jobcore "github.com/gabrielmoura/nostr-relay-server/internal/jobs"
	"github.com/gabrielmoura/nostr-relay-server/pkg/magic"
	"github.com/gofiber/fiber/v2"
	"github.com/minio/sha256-simd"
	"github.com/tmthrgd/go-hex"
	"go.uber.org/zap"
	"os"
	"path/filepath"
)

type mediaPutResponse struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	URL        string `json:"url"`
	SHA256     string `json:"sha256"`
	Size       int64  `json:"size"`
	Type       string `json:"type"`
	Uploaded   int64  `json:"uploaded"`
	JobID      string `json:"job_id,omitempty"`
	Processing struct {
		Optimized bool `json:"optimized"`
		Thumbnail bool `json:"thumbnail"`
		Blurhash  bool `json:"blurhash"`
		Streaming bool `json:"streaming"`
	} `json:"processing"`
}

func MediaHandler(c *fiber.Ctx) error {
	startedAt := time.Now()
	statusCode := fiber.StatusOK
	errorCategory := ""
	defer func() {
		observeBlossomRequest("/media", c.Method(), startedAt)
		if statusCode >= 400 {
			observeBlossomError("/media", c.Method(), statusCode, errorCategory)
		}
	}()

	switch c.Method() {
	case fiber.MethodPut:
		responseErr := putMediaHandler(c)
		if responseErr != nil {
			if fiberErr, ok := responseErr.(*fiber.Error); ok {
				statusCode = fiberErr.Code
			} else {
				statusCode = fiber.StatusInternalServerError
			}
			errorCategory = blossomErrorCategory(responseErr, "internal")
		}
		return responseErr
	case fiber.MethodHead:
		responseErr := headMediaHandler(c)
		if responseErr != nil {
			if fiberErr, ok := responseErr.(*fiber.Error); ok {
				statusCode = fiberErr.Code
			} else {
				statusCode = fiber.StatusInternalServerError
			}
			errorCategory = blossomErrorCategory(responseErr, "internal")
		}
		return responseErr
	default:
		statusCode = fiber.StatusMethodNotAllowed
		errorCategory = "invalid_method"
		return c.Status(fiber.StatusMethodNotAllowed).SendString("Invalid request method")
	}
}

func putMediaHandler(c *fiber.Ctx) error {
	expectedHash := normalizeMirrorHash(c.Get("X-SHA-256"))
	pubKey, err := processBlossomActionAuth(c, "media", expectedHash)
	if err != nil {
		status := fiber.StatusUnauthorized
		if fiberErr, ok := err.(*fiber.Error); ok {
			status = fiberErr.Code
		}
		return c.Status(status).SendString(err.Error())
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
	mimeType := ternaryString(strings.ToLower(strings.TrimSpace(mgl.MIME)), strings.ToLower(strings.TrimSpace(c.Get("Content-Type"))))
	if !acceptMimeType(mimeType, config.Cfg.Store.AcceptedMimetypes) {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid file type")
	}

	hasher := sha256.New()
	_, _ = hasher.Write(bodyBytes)
	hashString := hex.EncodeToString(hasher.Sum(nil))
	if expectedHash != "" && expectedHash != hashString {
		return c.Status(fiber.StatusUnauthorized).SendString("Invalid file hash")
	}

	policy, ok, err := db.DbQueries.GetBlossomServerPolicy(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read upload policy")
	}
	if !ok {
		policy = db2.BlossomServerPolicy{Mode: "free"}
	}
	quota, quotaOK, err := db.DbQueries.GetBlossomQuota(c.Context(), pubKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read upload quota")
	}
	usage, usageOK, err := db.DbQueries.GetBlossomUserUsage(c.Context(), pubKey)
	if err != nil {
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
	reviewState, blocked, blockedReason, policyErr := evaluateUploadPolicy(policy, quotaRef, usedBytes, int64(len(bodyBytes)))
	if policyErr != nil {
		return policyErr
	}

	filePath := filepath.Join(blobPath, hashString)
	if _, statErr := os.Stat(filePath); statErr != nil {
		if err := os.WriteFile(filePath, bodyBytes, 0o644); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to persist media file")
		}
	}

	obj := &db2.Object{
		Hash:            hashString,
		MimeType:        mimeType,
		Size:            int64(len(bodyBytes)),
		CreatedAt:       time.Now().UTC(),
		Blocked:         blocked,
		BlockedByReason: blockedReason,
		PublicKey:       pubKey,
	}
	if err := db.DbQueries.InsertObject(c.Context(), obj); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to save media metadata")
	}
	if err := db.DbQueries.UpsertBlossomObjectAdmin(c.Context(), db2.UpsertBlossomObjectAdminParams{
		Hash:             hashString,
		Extension:        dotExtension(mgl.Extension),
		IngressBytes:     int64(len(bodyBytes)),
		ExifStatus:       "pending",
		ReviewState:      reviewState,
		ProcessingStatus: initialMediaProcessingStatus(),
		ProcessingError:  "",
	}); err != nil {
		log.Logger.Error("Blossom media admin metadata save error", zap.Error(err))
	}

	jobID := ""
	status := "processing"
	message := "media optimization queued"
	if config.Cfg.Store.MediaProcessing.Enabled {
		service := jobcore.Default()
		if service == nil || service.Dispatcher == nil {
			return c.Status(fiber.StatusServiceUnavailable).SendString("Job runtime is not initialized")
		}
		id, dispatchErr := service.Dispatcher.Dispatch(c.Context(), internalblossom.MediaJob{Hash: hashString, RequestedBy: pubKey}, jobcore.WithQueue(config.Cfg.Jobs.DefaultQueue))
		if dispatchErr != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to enqueue media optimization")
		}
		jobID = id.String()
	} else {
		status = "ready"
		message = "media optimization disabled"
	}

	metrics.UploadCounter.Inc()

	response := mediaPutResponse{
		Status:   status,
		Message:  message,
		URL:      mediaURLWithExtension(hashString, dotExtension(mgl.Extension)),
		SHA256:   hashString,
		Size:     int64(len(bodyBytes)),
		Type:     mimeType,
		Uploaded: obj.CreatedAt.Unix(),
		JobID:    jobID,
	}
	return c.Status(fiber.StatusAccepted).JSON(response)
}

func headMediaHandler(c *fiber.Ctx) error {
	hash := normalizeMirrorHash(c.Get("X-SHA-256"))
	if hash == "" {
		return c.Status(fiber.StatusBadRequest).SendString("X-SHA-256 header is required")
	}
	pubKey, err := processBlossomActionAuth(c, "media", hash)
	if err != nil {
		status := fiber.StatusUnauthorized
		if fiberErr, ok := err.(*fiber.Error); ok {
			status = fiberErr.Code
		}
		return c.Status(status).SendString(err.Error())
	}
	_ = pubKey

	obj, err := db.DbQueries.GetObjectByHash(c.UserContext(), hash)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to inspect media object")
	}
	if obj.Hash == "" {
		return c.Status(fiber.StatusNotFound).SendString("Media object not found")
	}
	adminObject, ok, err := db.DbQueries.GetBlossomObject(c.UserContext(), hash)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to inspect media processing status")
	}

	status := "pending"
	optimized := false
	thumbnail := false
	blurhashReady := false
	streaming := false
	if ok {
		status = adminObject.ProcessingStatus
		optimized = strings.TrimSpace(adminObject.OptimizedHash) != ""
		thumbnail = strings.TrimSpace(adminObject.ThumbnailHash) != ""
		blurhashReady = strings.TrimSpace(adminObject.Blurhash) != ""
		streaming = strings.TrimSpace(adminObject.HLSManifestHash) != ""
	}

	c.Set("X-SHA-256", hash)
	c.Set("X-Media-Status", status)
	c.Set("X-Optimized-Available", strconvBool(optimized))
	c.Set("X-Thumbnail-Available", strconvBool(thumbnail))
	c.Set("X-Blurhash-Available", strconvBool(blurhashReady))
	c.Set("X-Streaming-Available", strconvBool(streaming))
	if ok && strings.TrimSpace(adminObject.ProcessingError) != "" {
		c.Set("X-Processing-Error", adminObject.ProcessingError)
	}
	return c.SendStatus(fiber.StatusOK)
}

func strconvBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func initialMediaProcessingStatus() string {
	if config.Cfg.Store.MediaProcessing.Enabled {
		return "pending"
	}
	return "ready"
}
