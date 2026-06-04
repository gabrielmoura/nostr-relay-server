package graph

import (
	stdjson "encoding/json"
	"strings"
	"unicode"
)

var restKeyOverrides = map[string]string{
	"ws_id":          "wsid",
	"target_event_id": "targetEventID",
	"direct_url":     "directURL",
	"optimized_url":  "optimizedURL",
	"thumbnail_url":  "thumbnailURL",
	"blossom_id":     "blossomID",
	"duration_ms":    "durationMS",
	"relay_url":      "relayUrl",
	"job_id":         "jobId",
	"event_id":       "eventID",
	"mime_type":      "mimeType",
	"gps_detected":   "gpsDetected",
	"bitrate_kbps":   "bitrateKbps",
	"image_urls":     "imageUrls",
	"nip94_tags":     "nip94Tags",
	"target_nevent":  "targetNevent",
	"target_pubkey":  "targetPubkey",
	"target_created_at": "targetCreatedAt",
	"target_created_at_iso": "targetCreatedAtIso",
	"report_event_id": "reportEventId",
	"reporter_pubkey": "reporterPubkey",
	"reporter_npub": "reporterNpub",
	"reporter_display_name": "reporterDisplayName",
	"reporter_picture": "reporterPicture",
	"reported_event_id": "reportedEventId",
	"reported_pubkey": "reportedPubkey",
	"report_type": "reportType",
	"last_reported_at": "lastReportedAt",
	"top_authors": "topAuthors",
	"top_tags": "topTags",
	"top_targets": "topTargets",
	"total_events": "totalEvents",
	"total_reports": "totalReports",
	"unique_target_authors": "uniqueTargetAuthors",
	"active_connections": "activeConnections",
	"authed_connections": "authedConnections",
	"logged_users": "loggedUsers",
	"banned_users": "bannedUsers",
	"indexed_events": "indexedEvents",
	"events_per_minute": "eventsPerMinute",
	"relay_status": "relayStatus",
	"source_relay": "sourceRelay",
	"relays_tried": "relaysTried",
	"relay_results": "relayResults",
	"target_types": "targetTypes",
	"object_hash": "objectHash",
	"resolved_by": "resolvedBy",
	"resolved_note": "resolvedNote",
	"created_by": "createdBy",
	"created_at": "createdAt",
	"updated_at": "updatedAt",
	"last_upload_at": "lastUploadAt",
	"storage_used_bytes": "storageUsedBytes",
	"storage_quota_bytes": "storageQuotaBytes",
	"monthly_egress_bytes": "monthlyEgressBytes",
	"egress_quota_bytes": "egressQuotaBytes",
	"object_count": "objectCount",
	"review_state": "reviewState",
	"exif_status": "exifStatus",
	"download_count": "downloadCount",
	"ingress_bytes": "ingressBytes",
	"egress_bytes": "egressBytes",
	"flag_reason": "flagReason",
	"uploader_pubkey": "uploaderPubkey",
	"display_name": "displayName",
	"last_seen_at": "lastSeenAt",
	"connection_count": "connectionCount",
	"connection_state": "connectionState",
	"subscription_count": "subscriptionCount",
	"connected_at": "connectedAt",
	"user_agent": "userAgent",
	"related_ids": "relatedIds",
	"relay_hints": "relayHints",
	"has_more": "hasMore",
	"page_info": "pageInfo",
	"member_count": "memberCount",
	"group_id": "groupId",
	"last_computed_at": "lastComputedAt",
	"trusted_pubkeys": "trustedPubkeys",
	"total_nodes": "totalNodes",
	"total_edges": "totalEdges",
	"job_name": "jobName",
	"max_attempts": "maxAttempts",
	"started_at": "startedAt",
	"finished_at": "finishedAt",
	"run_at": "runAt",
	"last_error": "lastError",
	"author_npub": "authorNpub",
	"target_author": "targetAuthor",
	"report_count": "reportCount",
	"report_types": "reportTypes",
	"target_type": "targetType",
	"relay_hint": "relayHint",
	"default_storage_quota_bytes": "defaultStorageQuotaBytes",
	"enabled_user_default_storage_quota_bytes": "enabledUserDefaultStorageQuotaBytes",
	"default_egress_quota_bytes": "defaultEgressQuotaBytes",
	"enabled_user_default_egress_quota_bytes": "enabledUserDefaultEgressQuotaBytes",
	"assigned_by": "assignedBy",
	"assigned_at": "assignedAt",
}

func normalizeRESTPayload(payload []byte) ([]byte, error) {
	var raw any
	if err := stdjson.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}
	normalized := normalizeRESTValue(raw)
	return stdjson.Marshal(normalized)
}

func normalizeRESTValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		mapped := make(map[string]any, len(typed))
		for key, nested := range typed {
			mapped[normalizeRESTKey(key)] = normalizeRESTValue(nested)
		}
		return mapped
	case []any:
		mapped := make([]any, len(typed))
		for i := range typed {
			mapped[i] = normalizeRESTValue(typed[i])
		}
		return mapped
	default:
		return value
	}
}

func normalizeRESTKey(key string) string {
	if mapped, ok := restKeyOverrides[key]; ok {
		return mapped
	}
	parts := strings.Split(key, "_")
	if len(parts) == 1 {
		return key
	}
	for i := 1; i < len(parts); i++ {
		parts[i] = upperFirst(parts[i])
	}
	return strings.Join(parts, "")
}

func upperFirst(value string) string {
	if value == "" {
		return value
	}
	runes := []rune(strings.ToLower(value))
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
