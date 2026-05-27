package blossom

import (
	"context"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	dbmodel "github.com/gabrielmoura/nostr-relay-server/infra/db"
	storedb "github.com/gabrielmoura/nostr-relay-server/internal/db"
	jobcore "github.com/gabrielmoura/nostr-relay-server/internal/jobs"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/gabrielmoura/nostr-relay-server/pkg/magic"
	"github.com/minio/sha256-simd"
	"github.com/tmthrgd/go-hex"
)

const (
	mirrorJobName = "blossom.mirror"
	mediaJobName  = "blossom.media.optimize"
	blobPath      = "files"
)

type MirrorJob struct {
	SourceURL      string `json:"source_url"`
	ExpectedSHA256 string `json:"expected_sha256"`
	RequestedBy    string `json:"requested_by,omitempty"`
}

func (MirrorJob) Name() string { return mirrorJobName }

type MediaJob struct {
	Hash        string `json:"hash"`
	RequestedBy string `json:"requested_by,omitempty"`
}

func (MediaJob) Name() string { return mediaJobName }

type JobResult struct {
	Hash      string `json:"hash,omitempty"`
	SourceURL string `json:"source_url,omitempty"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}

func RegisterQueueHandlers(registry *jobcore.MemoryRegistry) error {
	if err := jobcore.RegisterTyped(registry, mirrorJobName, handleMirrorJob); err != nil {
		return err
	}
	return jobcore.RegisterTyped(registry, mediaJobName, handleMediaJob)
}

func handleMirrorJob(ctx context.Context, job MirrorJob) error {
	result := JobResult{SourceURL: job.SourceURL, Status: "failed"}
	hash, err := mirrorRemoteObject(ctx, job)
	if err != nil {
		result.Error = err.Error()
		_ = jobcore.SetResult(ctx, result)
		return err
	}
	result.Hash = hash
	result.Status = "succeeded"
	_ = jobcore.SetResult(ctx, result)
	return RecordAudit(ctx, job.RequestedBy, "mirror.create", "object", hash, map[string]string{"source_url": job.SourceURL})
}

func handleMediaJob(ctx context.Context, job MediaJob) error {
	result := JobResult{Hash: job.Hash, Status: "failed"}
	if err := optimizeObjectMetadata(ctx, job.Hash); err != nil {
		result.Error = err.Error()
		_ = jobcore.SetResult(ctx, result)
		return err
	}
	result.Status = "succeeded"
	_ = jobcore.SetResult(ctx, result)
	return RecordAudit(ctx, job.RequestedBy, "media.optimize", "object", job.Hash, map[string]string{"hash": job.Hash})
}

func mirrorRemoteObject(ctx context.Context, job MirrorJob) (string, error) {
	if err := validateMirrorJob(job); err != nil {
		return "", err
	}

	policy, quotaRef, usedBytes, err := loadMirrorUploadPolicy(ctx, job.RequestedBy)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, job.SourceURL, nil)
	if err != nil {
		return "", fmt.Errorf("build mirror request: %w", err)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download remote object: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("mirror request failed with status %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp(blobPath, "mirror-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create mirror temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
	}()

	hasher := sha256.New()
	size, err := io.Copy(io.MultiWriter(tmpFile, hasher), resp.Body)
	if err != nil {
		return "", fmt.Errorf("stream mirror body: %w", err)
	}
	hash := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(hash, strings.TrimSpace(job.ExpectedSHA256)) {
		return "", fmt.Errorf("mirror hash mismatch")
	}

	reviewState, blocked, blockedReason, policyErr := evaluateMirrorPolicy(policy, quotaRef, usedBytes, size)
	if policyErr != nil {
		return "", policyErr
	}

	mimeType, err := detectMirroredMIME(tmpFile, resp.Header.Get("Content-Type"))
	if err != nil {
		return "", fmt.Errorf("detect mirrored mime type: %w", err)
	}
	filePath := filepath.Join(blobPath, hash)
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("close mirror temp file: %w", err)
	}
	if err := persistMirroredFile(tmpPath, filePath); err != nil {
		return "", err
	}

	obj := &dbmodel.Object{
		Hash:            hash,
		CreatedAt:       time.Now().UTC(),
		MimeType:        mimeType,
		Size:            size,
		Blocked:         blocked,
		BlockedByReason: blockedReason,
		PublicKey:       normalizeActorPubkey(job.RequestedBy),
	}
	if err := storedb.DbQueries.InsertObject(ctx, obj); err != nil {
		return "", fmt.Errorf("insert mirrored object: %w", err)
	}

	if err := storedb.DbQueries.UpsertBlossomObjectAdmin(ctx, dbmodel.UpsertBlossomObjectAdminParams{
		Hash:         hash,
		Extension:    extensionForMIME(mimeType),
		IngressBytes: size,
		ReviewState:  reviewState,
		ExifStatus:   "pending",
		Mirrors:      mustMarshalStrings([]string{job.SourceURL}),
	}); err != nil {
		return "", fmt.Errorf("upsert mirrored object admin metadata: %w", err)
	}

	return hash, optimizeObjectMetadata(ctx, hash)
}

func optimizeObjectMetadata(ctx context.Context, hash string) error {
	if err := storedb.DbQueries.UpdateBlossomObjectProcessing(ctx, hash, "processing", ""); err != nil {
		return fmt.Errorf("mark object processing: %w", err)
	}

	object, err := storedb.DbQueries.GetObjectByHash(ctx, hash)
	if err != nil {
		_ = storedb.DbQueries.UpdateBlossomObjectProcessing(ctx, hash, "failed", err.Error())
		return fmt.Errorf("load object for optimization: %w", err)
	}
	if object.Hash == "" {
		_ = storedb.DbQueries.UpdateBlossomObjectProcessing(ctx, hash, "failed", "object not found")
		return fmt.Errorf("object %s not found", hash)
	}
	adminObject, ok, err := storedb.DbQueries.GetBlossomObject(ctx, hash)
	if err != nil {
		_ = storedb.DbQueries.UpdateBlossomObjectProcessing(ctx, hash, "failed", err.Error())
		return fmt.Errorf("load object admin metadata: %w", err)
	}
	if !ok {
		adminObject = dbmodel.BlossomObjectRow{}
	}
	filePath := filepath.Join(blobPath, hash)
	result, err := processMediaOptimization(ctx, object, filePath)
	if err != nil {
		_ = storedb.DbQueries.UpdateBlossomObjectProcessing(ctx, hash, "failed", err.Error())
		return fmt.Errorf("process media optimization: %w", err)
	}

	nip94Tags := buildNIP94Tags(
		object,
		result.width,
		result.height,
		result.durationMS,
		result.bitrateKbps,
		result.blurhash,
		result.thumbnailHash,
		result.optimizedHash,
		result.hlsManifestHash,
		decodeMirrorURLsFromPayload(adminObject.Mirrors),
	)

	if err := storedb.DbQueries.UpsertBlossomObjectAdmin(ctx, dbmodel.UpsertBlossomObjectAdminParams{
		Hash:             hash,
		Extension:        extensionForMIME(object.MimeType),
		Width:            result.width,
		Height:           result.height,
		DurationMS:       result.durationMS,
		BitrateKbps:      result.bitrateKbps,
		Blurhash:         result.blurhash,
		ThumbnailHash:    result.thumbnailHash,
		OptimizedHash:    result.optimizedHash,
		HLSManifestHash:  result.hlsManifestHash,
		ProcessingStatus: result.processingState,
		ProcessingError:  result.processingError,
		IngressBytes:     object.Size,
		NIP94Tags:        marshalNIP94Tags(nip94Tags),
	}); err != nil {
		_ = storedb.DbQueries.UpdateBlossomObjectProcessing(ctx, hash, "failed", err.Error())
		return fmt.Errorf("persist optimized metadata: %w", err)
	}
	return nil
}

func directURL(hash string) string {
	return fmt.Sprintf("%s/%s", strings.TrimRight(config.Cfg.Store.MediaPath, "/"), hash)
}

func extensionForMIME(mimeType string) string {
	extensions, err := mime.ExtensionsByType(mimeType)
	if err != nil || len(extensions) == 0 {
		return ""
	}
	return strings.ToLower(extensions[0])
}

func mustMarshalStrings(values []string) []byte {
	payload, err := json.Marshal(values)
	if err != nil {
		return []byte("[]")
	}
	return payload
}

func validateMirrorJob(job MirrorJob) error {
	parsed, err := url.Parse(strings.TrimSpace(job.SourceURL))
	if err != nil || parsed == nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("mirror source url must be absolute")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("mirror source url scheme must be http or https")
	}
	hash := strings.ToLower(strings.TrimSpace(job.ExpectedSHA256))
	if len(hash) != 64 {
		return fmt.Errorf("mirror expected sha256 must be 64 hex chars")
	}
	for _, r := range hash {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return fmt.Errorf("mirror expected sha256 must be lowercase hex")
		}
	}
	return nil
}

func loadMirrorUploadPolicy(ctx context.Context, requestedBy string) (dbmodel.BlossomServerPolicy, *dbmodel.BlossomPubkeyQuota, int64, error) {
	policy, ok, err := storedb.DbQueries.GetBlossomServerPolicy(ctx)
	if err != nil {
		return dbmodel.BlossomServerPolicy{}, nil, 0, fmt.Errorf("read blossom policy: %w", err)
	}
	if !ok {
		policy = dbmodel.BlossomServerPolicy{Mode: "free"}
	}

	pubkey := normalizeActorPubkey(requestedBy)
	quota, quotaOK, err := storedb.DbQueries.GetBlossomQuota(ctx, pubkey)
	if err != nil {
		return dbmodel.BlossomServerPolicy{}, nil, 0, fmt.Errorf("read blossom quota: %w", err)
	}
	usage, usageOK, err := storedb.DbQueries.GetBlossomUserUsage(ctx, pubkey)
	if err != nil {
		return dbmodel.BlossomServerPolicy{}, nil, 0, fmt.Errorf("read blossom usage: %w", err)
	}

	var quotaRef *dbmodel.BlossomPubkeyQuota
	if quotaOK {
		quotaRef = &quota
	}
	usedBytes := int64(0)
	if usageOK {
		usedBytes = usage.StorageUsedBytes
	}
	return policy, quotaRef, usedBytes, nil
}

func evaluateMirrorPolicy(policy dbmodel.BlossomServerPolicy, quota *dbmodel.BlossomPubkeyQuota, usedBytes int64, size int64) (string, bool, string, error) {
	mode := strings.TrimSpace(policy.Mode)
	if mode == "" {
		mode = "free"
	}
	switch mode {
	case "enabled_users":
		if quota == nil || !quota.Enabled {
			return "", false, "", fmt.Errorf("uploader is not enabled for Blossom mirrors")
		}
	case "mandatory_review", "free":
	default:
		return "", false, "", fmt.Errorf("invalid blossom policy mode")
	}
	limit := mirrorEffectiveStorageQuota(policy, quota)
	if limit != nil && usedBytes+size > *limit {
		return "", false, "", fmt.Errorf("storage quota exceeded")
	}
	if mode == "mandatory_review" {
		return "pending_review", true, "pending manual review", nil
	}
	return "ready", false, "", nil
}

func mirrorEffectiveStorageQuota(policy dbmodel.BlossomServerPolicy, quota *dbmodel.BlossomPubkeyQuota) *int64 {
	if quota != nil && quota.StorageQuotaBytes.Valid {
		value := quota.StorageQuotaBytes.Int64
		return &value
	}
	if strings.TrimSpace(policy.Mode) == "enabled_users" {
		if policy.EnabledUserDefaultStorageQuotaBytes.Valid {
			value := policy.EnabledUserDefaultStorageQuotaBytes.Int64
			return &value
		}
		return nil
	}
	if policy.DefaultStorageQuotaBytes.Valid {
		value := policy.DefaultStorageQuotaBytes.Int64
		return &value
	}
	return nil
}

func detectMirroredMIME(file *os.File, contentType string) (string, error) {
	trimmed := strings.TrimSpace(contentType)
	if trimmed != "" {
		if mimeType, _, err := mime.ParseMediaType(trimmed); err == nil && mimeType != "" {
			return strings.ToLower(mimeType), nil
		}
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("rewind temp file: %w", err)
	}
	buf := make([]byte, 8192)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read temp file header: %w", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("rewind temp file: %w", err)
	}
	mgl, err := magic.Lookup(buf[:n])
	if err != nil {
		return strings.ToLower(http.DetectContentType(buf[:n])), nil
	}
	return strings.ToLower(mgl.MIME), nil
}

func persistMirroredFile(tmpPath string, filePath string) error {
	if err := os.Rename(tmpPath, filePath); err == nil {
		return nil
	} else if !os.IsExist(err) {
		if _, statErr := os.Stat(filePath); statErr == nil {
			return nil
		}
		return fmt.Errorf("persist mirrored file: %w", err)
	}
	return nil
}

func buildNIP94Tags(
	object dbmodel.Object,
	width *int32,
	height *int32,
	durationMS *int64,
	bitrateKbps *int32,
	blurhash string,
	thumbnailHash string,
	optimizedHash string,
	hlsManifestHash string,
	mirrorURLs []string,
) [][]string {
	tags := make([][]string, 0, 12)
	tags = append(tags, []string{"url", directURL(object.Hash)})
	tags = append(tags, []string{"m", strings.ToLower(strings.TrimSpace(object.MimeType))})
	tags = append(tags, []string{"x", object.Hash})
	tags = append(tags, []string{"size", strconv.FormatInt(object.Size, 10)})
	tags = append(tags, []string{"service", "nip96"})
	if width != nil && height != nil {
		tags = append(tags, []string{"dim", fmt.Sprintf("%dx%d", *width, *height)})
	}
	if durationMS != nil {
		tags = append(tags, []string{"duration", strconv.FormatInt(*durationMS, 10)})
	}
	if bitrateKbps != nil {
		tags = append(tags, []string{"bitrate", strconv.FormatInt(int64(*bitrateKbps), 10)})
	}
	if strings.TrimSpace(blurhash) != "" {
		tags = append(tags, []string{"blurhash", strings.TrimSpace(blurhash)})
	}
	if strings.TrimSpace(thumbnailHash) != "" {
		tags = append(tags, []string{"thumb", directURL(strings.TrimSpace(thumbnailHash)), strings.TrimSpace(thumbnailHash)})
	}
	if strings.TrimSpace(optimizedHash) != "" {
		tags = append(tags, []string{"image", directURL(strings.TrimSpace(optimizedHash)), strings.TrimSpace(optimizedHash)})
		if optimizedHash != object.Hash {
			tags = append(tags, []string{"ox", object.Hash})
		}
	}
	if strings.TrimSpace(hlsManifestHash) != "" {
		tags = append(tags, []string{"fallback", directURL(strings.TrimSpace(hlsManifestHash))})
	}
	for _, mirrorURL := range uniqueStrings(mirrorURLs) {
		if strings.TrimSpace(mirrorURL) == "" {
			continue
		}
		tags = append(tags, []string{"fallback", strings.TrimSpace(mirrorURL)})
	}
	return tags
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	items := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		items = append(items, trimmed)
	}
	return items
}
