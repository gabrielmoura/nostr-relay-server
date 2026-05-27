package http

type adminBlossomOverviewResponse struct {
	Storage struct {
		UsedBytes   int64   `json:"used_bytes"`
		FreeBytes   int64   `json:"free_bytes"`
		UsedPercent float64 `json:"used_percent"`
	} `json:"storage"`
	Objects struct {
		Total         int64 `json:"total"`
		Flagged       int64 `json:"flagged"`
		PendingReview int64 `json:"pending_review"`
	} `json:"objects"`
	Traffic struct {
		MonthlyIngressBytes int64 `json:"monthly_ingress_bytes"`
		MonthlyEgressBytes  int64 `json:"monthly_egress_bytes"`
	} `json:"traffic"`
	Users struct {
		Active      int64 `json:"active"`
		Whitelisted int64 `json:"whitelisted"`
	} `json:"users"`
	Workers struct {
		Running int `json:"running"`
		Failed  int `json:"failed"`
	} `json:"workers"`
	Policy adminBlossomPolicyResponse  `json:"policy"`
	Alerts []adminBlossomAlertResponse `json:"alerts"`
}

type adminBlossomAlertResponse struct {
	Level   string `json:"level"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type adminBlossomObjectResponse struct {
	Hash             string     `json:"hash"`
	UploaderPubkey   string     `json:"uploader_pubkey"`
	MIMEType         string     `json:"mime_type"`
	Extension        string     `json:"extension"`
	Size             int64      `json:"size"`
	CreatedAt        string     `json:"created_at"`
	Width            *int32     `json:"width,omitempty"`
	Height           *int32     `json:"height,omitempty"`
	DurationMS       *int64     `json:"duration_ms,omitempty"`
	BitrateKbps      *int32     `json:"bitrate_kbps,omitempty"`
	Blurhash         string     `json:"blurhash,omitempty"`
	ThumbnailURL     string     `json:"thumbnail_url,omitempty"`
	DirectURL        string     `json:"direct_url"`
	BlossomID        string     `json:"blossom_id,omitempty"`
	OptimizedURL     string     `json:"optimized_url,omitempty"`
	ReviewState      string     `json:"review_state"`
	ExifStatus       string     `json:"exif_status"`
	GPSDetected      bool       `json:"gps_detected"`
	DownloadCount    int64      `json:"download_count"`
	LastDownloadedAt string     `json:"last_downloaded_at,omitempty"`
	IngressBytes     int64      `json:"ingress_bytes,omitempty"`
	EgressBytes      int64      `json:"egress_bytes,omitempty"`
	FlagReason       string     `json:"flag_reason,omitempty"`
	NIP94Tags        [][]string `json:"nip94_tags,omitempty"`
	Mirrors          []string   `json:"mirrors,omitempty"`
	ReportCount      int64      `json:"report_count,omitempty"`
}

type adminBlossomPolicyResponse struct {
	Mode                                string `json:"mode"`
	DefaultStorageQuotaBytes            *int64 `json:"default_storage_quota_bytes,omitempty"`
	DefaultEgressQuotaBytes             *int64 `json:"default_egress_quota_bytes,omitempty"`
	EnabledUserDefaultStorageQuotaBytes *int64 `json:"enabled_user_default_storage_quota_bytes,omitempty"`
	EnabledUserDefaultEgressQuotaBytes  *int64 `json:"enabled_user_default_egress_quota_bytes,omitempty"`
	UpdatedAt                           string `json:"updated_at,omitempty"`
}

type adminBlossomPolicyRequest struct {
	Mode string `json:"mode"`
}

type adminBlossomPlanResponse struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Scope             string `json:"scope"`
	StorageQuotaBytes *int64 `json:"storage_quota_bytes,omitempty"`
	EgressQuotaBytes  *int64 `json:"egress_quota_bytes,omitempty"`
	Description       string `json:"description,omitempty"`
	IsDefault         bool   `json:"is_default"`
	UpdatedAt         string `json:"updated_at,omitempty"`
}

type adminBlossomPlanRequest struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Scope             string `json:"scope"`
	StorageQuotaBytes *int64 `json:"storage_quota_bytes"`
	EgressQuotaBytes  *int64 `json:"egress_quota_bytes"`
	Description       string `json:"description"`
	IsDefault         bool   `json:"is_default"`
}

type adminBlossomBulkReviewRequest struct {
	Hashes []string `json:"hashes"`
	Action string   `json:"action"`
	Reason string   `json:"reason"`
}

type adminBlossomBulkReviewResponse struct {
	OK      bool  `json:"ok"`
	Updated int64 `json:"updated"`
}

type adminBlossomWhitelistRequest struct {
	Pubkey            string `json:"pubkey"`
	Enabled           bool   `json:"enabled"`
	StorageQuotaBytes *int64 `json:"storage_quota_bytes"`
	EgressQuotaBytes  *int64 `json:"egress_quota_bytes"`
	Notes             string `json:"notes"`
}

type adminBlossomUserResponse struct {
	Pubkey             string `json:"pubkey"`
	DisplayName        string `json:"display_name,omitempty"`
	Picture            string `json:"picture,omitempty"`
	Npub               string `json:"npub,omitempty"`
	ObjectCount        int64  `json:"object_count"`
	StorageUsedBytes   int64  `json:"storage_used_bytes"`
	StorageQuotaBytes  *int64 `json:"storage_quota_bytes,omitempty"`
	MonthlyEgressBytes int64  `json:"monthly_egress_bytes"`
	EgressQuotaBytes   *int64 `json:"egress_quota_bytes,omitempty"`
	Enabled            bool   `json:"enabled"`
	LastUploadAt       string `json:"last_upload_at,omitempty"`
	Notes              string `json:"notes,omitempty"`
}

type adminBlossomUserDetailResponse struct {
	adminBlossomUserResponse
	Files []adminBlossomObjectResponse `json:"files"`
}

type adminBlossomMirrorRequest struct {
	SourceURL      string `json:"source_url"`
	ExpectedSHA256 string `json:"expected_sha256"`
}

type adminBlossomMirrorResponse struct {
	OK     bool   `json:"ok"`
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

type adminBlossomPlanAssignRequest struct {
	Pubkey string `json:"pubkey"`
}

type adminBlossomPlanAssignResponse struct {
	OK     bool   `json:"ok"`
	PlanID string `json:"plan_id"`
	Pubkey string `json:"pubkey"`
}

type adminBlossomPlanAssignmentResponse struct {
	PlanID      string `json:"plan_id"`
	Pubkey      string `json:"pubkey"`
	DisplayName string `json:"display_name,omitempty"`
	Picture     string `json:"picture,omitempty"`
	Npub        string `json:"npub,omitempty"`
	AssignedBy  string `json:"assigned_by"`
	AssignedAt  string `json:"assigned_at"`
}

type adminBlossomWorkerResponse struct {
	JobID         string `json:"job_id"`
	JobType       string `json:"job_type"`
	Status        string `json:"status"`
	TargetHash    string `json:"target_hash,omitempty"`
	Detail        string `json:"detail"`
	ProgressLabel string `json:"progress_label,omitempty"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type adminBlossomAuditResponse struct {
	ID           string            `json:"id"`
	ActorPubkey  string            `json:"actor_pubkey"`
	Action       string            `json:"action"`
	TargetType   string            `json:"target_type"`
	TargetID     string            `json:"target_id"`
	CreatedAt    string            `json:"created_at"`
	RequestID    string            `json:"request_id,omitempty"`
	NostrEventID string            `json:"nostr_event_id,omitempty"`
	Payload      map[string]string `json:"payload,omitempty"`
}

type adminBlossomReportResponse struct {
	ID             string `json:"id"`
	EventID        string `json:"event_id"`
	ObjectHash     string `json:"object_hash"`
	ReporterPubkey string `json:"reporter_pubkey"`
	ReporterNpub   string `json:"reporter_npub,omitempty"`
	TargetEventID  string `json:"target_event_id,omitempty"`
	TargetPubkey   string `json:"target_pubkey,omitempty"`
	ReportType     string `json:"report_type,omitempty"`
	Reason         string `json:"reason,omitempty"`
	Status         string `json:"status"`
	ResolvedBy     string `json:"resolved_by,omitempty"`
	ResolvedNote   string `json:"resolved_note,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
	ResolvedAt     string `json:"resolved_at,omitempty"`
}

type adminBlossomReportResolveRequest struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

type adminBlossomAnalyticsResponse struct {
	Reports struct {
		Total    int64               `json:"total"`
		Open     int64               `json:"open"`
		Resolved int64               `json:"resolved"`
		ByType   []adminCountByValue `json:"by_type"`
		ByStatus []adminCountByValue `json:"by_status"`
	} `json:"reports"`
	Objects struct {
		ByMime        []adminCountByValue `json:"by_mime"`
		ByReviewState []adminCountByValue `json:"by_review_state"`
	} `json:"objects"`
	Workers struct {
		ByStatus []adminCountByValue `json:"by_status"`
		ByType   []adminCountByValue `json:"by_type"`
	} `json:"workers"`
}

type adminCountByValue struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}
