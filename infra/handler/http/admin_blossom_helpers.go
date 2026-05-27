package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/gabrielmoura/nostr-relay-server/config"
	dbmodel "github.com/gabrielmoura/nostr-relay-server/infra/db"
	internalblossom "github.com/gabrielmoura/nostr-relay-server/internal/blossom"
	jobcore "github.com/gabrielmoura/nostr-relay-server/internal/jobs"
	jsonx "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
)

func mapBlossomObjectResponse(item dbmodel.BlossomObjectRow) adminBlossomObjectResponse {
	response := adminBlossomObjectResponse{
		Hash:           item.Hash,
		UploaderPubkey: item.UploaderPubkey,
		MIMEType:       item.MIMEType,
		Extension:      item.Extension,
		Size:           item.Size,
		CreatedAt:      formatTime(item.CreatedAt),
		Blurhash:       item.Blurhash,
		DirectURL:      blossomDirectURL(item.Hash, item.Extension),
		BlossomID:      blossomObjectID(item.Hash, item.Extension, item.UploaderPubkey, item.Size),
		ReviewState:    item.ReviewState,
		ExifStatus:     item.ExifStatus,
		GPSDetected:    item.GPSDetected,
		DownloadCount:  item.DownloadCount,
		IngressBytes:   item.IngressBytes,
		EgressBytes:    item.EgressBytes,
		FlagReason:     item.FlagReason,
		NIP94Tags:      decodeBlossomNIP94Tags(item.NIP94Tags),
		Mirrors:        decodeBlossomJSONStrings(item.Mirrors),
	}
	if item.Width.Valid {
		value := item.Width.Int32
		response.Width = &value
	}
	if item.Height.Valid {
		value := item.Height.Int32
		response.Height = &value
	}
	if item.DurationMS.Valid {
		value := item.DurationMS.Int64
		response.DurationMS = &value
	}
	if item.BitrateKbps.Valid {
		value := item.BitrateKbps.Int32
		response.BitrateKbps = &value
	}
	if item.LastDownloadedAt.Valid {
		response.LastDownloadedAt = formatTime(item.LastDownloadedAt.Time)
	}
	if item.ThumbnailHash != "" {
		response.ThumbnailURL = blossomDirectURL(item.ThumbnailHash, "")
	}
	if item.OptimizedHash != "" {
		response.OptimizedURL = blossomDirectURL(item.OptimizedHash, "")
	}
	return response
}

func mapBlossomPolicyResponse(item dbmodel.BlossomServerPolicy) adminBlossomPolicyResponse {
	response := adminBlossomPolicyResponse{Mode: item.Mode, UpdatedAt: formatTime(item.UpdatedAt)}
	if item.DefaultStorageQuotaBytes.Valid {
		value := item.DefaultStorageQuotaBytes.Int64
		response.DefaultStorageQuotaBytes = &value
	}
	if item.DefaultEgressQuotaBytes.Valid {
		value := item.DefaultEgressQuotaBytes.Int64
		response.DefaultEgressQuotaBytes = &value
	}
	if item.EnabledUserDefaultStorageQuotaBytes.Valid {
		value := item.EnabledUserDefaultStorageQuotaBytes.Int64
		response.EnabledUserDefaultStorageQuotaBytes = &value
	}
	if item.EnabledUserDefaultEgressQuotaBytes.Valid {
		value := item.EnabledUserDefaultEgressQuotaBytes.Int64
		response.EnabledUserDefaultEgressQuotaBytes = &value
	}
	return response
}

func mapBlossomPlanResponse(item dbmodel.BlossomPlan) adminBlossomPlanResponse {
	response := adminBlossomPlanResponse{
		ID:          item.ID,
		Name:        item.Name,
		Scope:       item.Scope,
		Description: item.Description,
		IsDefault:   item.IsDefault,
		UpdatedAt:   formatTime(item.UpdatedAt),
	}
	if item.StorageQuotaBytes.Valid {
		value := item.StorageQuotaBytes.Int64
		response.StorageQuotaBytes = &value
	}
	if item.EgressQuotaBytes.Valid {
		value := item.EgressQuotaBytes.Int64
		response.EgressQuotaBytes = &value
	}
	return response
}

func mapBlossomUserResponse(item dbmodel.BlossomUserUsageRow) adminBlossomUserResponse {
	response := adminBlossomUserResponse{
		Pubkey:             item.Pubkey,
		ObjectCount:        item.ObjectCount,
		StorageUsedBytes:   item.StorageUsedBytes,
		MonthlyEgressBytes: item.MonthlyEgress,
		Enabled:            item.Enabled,
		Notes:              item.Notes,
	}
	if item.DisplayName.Valid && item.DisplayName.String != "" {
		response.DisplayName = item.DisplayName.String
	} else if item.Name.Valid && item.Name.String != "" {
		response.DisplayName = item.Name.String
	}
	if item.Picture.Valid && item.Picture.String != "" {
		response.Picture = item.Picture.String
	}
	if item.Pubkey != "" {
		response.Npub = npubFromPublicKey(item.Pubkey)
	}
	if item.StorageQuotaBytes.Valid {
		value := item.StorageQuotaBytes.Int64
		response.StorageQuotaBytes = &value
	}
	if item.EgressQuotaBytes.Valid {
		value := item.EgressQuotaBytes.Int64
		response.EgressQuotaBytes = &value
	}
	if item.LastUploadAt.Valid {
		response.LastUploadAt = formatTime(item.LastUploadAt.Time)
	}
	return response
}

func mapBlossomPlanAssignmentResponse(item dbmodel.BlossomPlanAssignment) adminBlossomPlanAssignmentResponse {
	response := adminBlossomPlanAssignmentResponse{
		PlanID:     item.PlanID,
		Pubkey:     item.Pubkey,
		AssignedBy: item.AssignedBy,
		AssignedAt: formatTime(item.AssignedAt),
		Npub:       npubFromPublicKey(item.Pubkey),
	}
	if item.DisplayName.Valid && item.DisplayName.String != "" {
		response.DisplayName = item.DisplayName.String
	} else if item.Name.Valid && item.Name.String != "" {
		response.DisplayName = item.Name.String
	}
	if item.Picture.Valid && item.Picture.String != "" {
		response.Picture = item.Picture.String
	}
	return response
}

func mapBlossomAuditResponse(item dbmodel.BlossomAuditRecord) adminBlossomAuditResponse {
	return adminBlossomAuditResponse{
		ID:           strconv.FormatInt(item.ID, 10),
		ActorPubkey:  item.ActorPubkey,
		Action:       item.Action,
		TargetType:   item.TargetType,
		TargetID:     item.TargetID,
		CreatedAt:    formatTime(item.CreatedAt),
		RequestID:    item.RequestID,
		NostrEventID: item.NostrEventID,
		Payload:      decodeBlossomJSONMap(item.Payload),
	}
}

func mapBlossomReportResponse(item dbmodel.BlossomReportRow) adminBlossomReportResponse {
	response := adminBlossomReportResponse{
		ID:             strconv.FormatInt(item.ID, 10),
		EventID:        item.EventID,
		ObjectHash:     item.ObjectHash,
		ReporterPubkey: item.ReporterPubkey,
		ReporterNpub:   npubFromPublicKey(item.ReporterPubkey),
		TargetEventID:  item.TargetEventID,
		TargetPubkey:   item.TargetPubkey,
		ReportType:     item.ReportType,
		Reason:         item.Reason,
		Status:         item.Status,
		ResolvedBy:     item.ResolvedBy,
		ResolvedNote:   item.ResolvedNote,
	}
	if item.CreatedAt.Valid {
		response.CreatedAt = formatTime(item.CreatedAt.Time)
	}
	if item.ResolvedAt.Valid {
		response.ResolvedAt = formatTime(item.ResolvedAt.Time)
	}
	return response
}

func blossomDirectURL(hash string, extension string) string {
	base := fmt.Sprintf("%s/%s", strings.TrimRight(config.Cfg.Store.MediaPath, "/"), hash)
	ext := normalizeBlossomExtension(extension)
	if ext == "" {
		return base
	}
	return base + ext
}

func blossomObjectID(hash string, extension string, uploaderPubkey string, size int64) string {
	ext := strings.TrimPrefix(normalizeBlossomExtension(extension), ".")
	if ext == "" {
		ext = "bin"
	}
	values := url.Values{}
	if xs := blossomServerHint(); xs != "" {
		values.Add("xs", xs)
	}
	if strings.TrimSpace(uploaderPubkey) != "" {
		values.Add("as", uploaderPubkey)
	}
	if size > 0 {
		values.Add("sz", strconv.FormatInt(size, 10))
	}
	if encoded := values.Encode(); encoded != "" {
		return fmt.Sprintf("blossom:%s.%s?%s", hash, ext, encoded)
	}
	return fmt.Sprintf("blossom:%s.%s", hash, ext)
}

func blossomServerHint() string {
	parsed, err := url.Parse(strings.TrimSpace(config.Cfg.Store.MediaPath))
	if err != nil {
		return ""
	}
	if parsed.Host != "" {
		return parsed.Host
	}
	return strings.Trim(parsed.Path, "/")
}

func normalizeBlossomExtension(extension string) string {
	trimmed := strings.TrimSpace(extension)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, ".") {
		return trimmed
	}
	return "." + trimmed
}

func defaultBlossomServerPolicy() dbmodel.BlossomServerPolicy {
	return dbmodel.BlossomServerPolicy{Mode: "free"}
}

func decodeBlossomJSONMap(payload []byte) map[string]string {
	if len(payload) == 0 {
		return map[string]string{}
	}
	values := make(map[string]string)
	if err := json.Unmarshal(payload, &values); err != nil {
		return map[string]string{}
	}
	return values
}

func decodeBlossomNIP94Tags(payload []byte) [][]string {
	if len(payload) == 0 {
		return [][]string{}
	}
	var tags [][]string
	if err := json.Unmarshal(payload, &tags); err == nil {
		return tags
	}
	legacy := decodeBlossomJSONMap(payload)
	if len(legacy) == 0 {
		return [][]string{}
	}
	tags = make([][]string, 0, len(legacy))
	for key, value := range legacy {
		tags = append(tags, []string{key, value})
	}
	return tags
}

func decodeBlossomJSONStrings(payload []byte) []string {
	if len(payload) == 0 {
		return []string{}
	}
	values := make([]string, 0)
	if err := json.Unmarshal(payload, &values); err != nil {
		return []string{}
	}
	return values
}

func blossomActorPubkey() string {
	if strings.TrimSpace(config.Cfg.AdminPubKey) != "" {
		return config.Cfg.AdminPubKey
	}
	return config.Cfg.RelayInformation.PubKey
}

func blossomFileUsage() (usedBytes int64, freeBytes int64, err error) {
	path := filepath.Clean("files")
	var stats syscall.Statfs_t
	if err := syscall.Statfs(path, &stats); err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	freeBytes = int64(stats.Bavail) * int64(stats.Bsize)
	usedBytes = int64(stats.Blocks-stats.Bfree) * int64(stats.Bsize)
	return usedBytes, freeBytes, nil
}

func blossomWorkerSummary() (running int, failed int, err error) {
	service := jobcore.Default()
	if service == nil || service.Monitor == nil {
		return 0, 0, nil
	}
	snapshots, err := listAdminJobSnapshots(
		contextTODO(),
		service.Monitor,
		"",
	)
	if err != nil {
		return 0, 0, err
	}
	for _, snapshot := range snapshots {
		if !strings.HasPrefix(snapshot.Name, "blossom.") {
			continue
		}
		if snapshot.Status == jobcore.StatusRunning || snapshot.Status == jobcore.StatusQueued || snapshot.Status == jobcore.StatusDelayed {
			running++
		}
		if snapshot.Status == jobcore.StatusFailed || snapshot.Status == jobcore.StatusDead {
			failed++
		}
	}
	return running, failed, nil
}

func blossomWorkerDetail(snapshot jobcore.Snapshot) adminBlossomWorkerResponse {
	response := adminBlossomWorkerResponse{
		JobID:     snapshot.ID.String(),
		JobType:   snapshot.Name,
		Status:    snapshot.Status.String(),
		CreatedAt: formatTime(snapshot.CreatedAt),
		UpdatedAt: formatTime(snapshot.CreatedAt),
		Detail:    snapshot.LastError,
	}
	if snapshot.FinishedAt != nil {
		response.UpdatedAt = formatTime(*snapshot.FinishedAt)
	}
	if snapshot.StartedAt != nil && response.UpdatedAt == response.CreatedAt {
		response.UpdatedAt = formatTime(*snapshot.StartedAt)
	}
	if response.Detail == "" {
		response.Detail = "job available"
	}
	if snapshot.Name == "blossom.mirror" {
		var payload internalblossom.MirrorJob
		if err := jsonx.Unmarshal(snapshot.Payload, &payload); err == nil {
			response.Detail = payload.SourceURL
		}
		var result internalblossom.JobResult
		if err := jsonx.Unmarshal(snapshot.Result, &result); err == nil {
			response.TargetHash = result.Hash
			if result.Error != "" {
				response.Detail = result.Error
			}
		}
	}
	if snapshot.Name == "blossom.media.optimize" {
		var payload internalblossom.MediaJob
		if err := jsonx.Unmarshal(snapshot.Payload, &payload); err == nil {
			response.TargetHash = payload.Hash
			response.Detail = payload.Hash
		}
	}
	return response
}

func contextTODO() context.Context {
	return context.Background()
}
