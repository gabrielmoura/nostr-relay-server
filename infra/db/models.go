package db

import (
	"database/sql"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"time"
)

type BannedUser struct {
	User   Profile `json:"user"`
	Reason string  `json:"reason"`
	ID     int64   `json:"id"`
}

type Profile struct {
	PublicKey   string `json:"public_key"`
	Name        string `json:"name"`
	About       string `json:"about,omitempty"`
	Picture     string `json:"picture,omitempty"`
	Banner      string `json:"banner,omitempty"`
	Website     string `json:"website,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Lud16       string `json:"lud16,omitempty"`
	Pronouns    string `json:"pronouns,omitempty"`
	Nip05       string `json:"nip05,omitempty"`
	ID          int64  `json:"id"`
	Bot         bool   `json:"bot,omitempty"`
}

func (o *Object) ToJson() []byte {
	j, _ := json.Marshal(o)
	return j
}

type ObjectHash [32]byte

func StringToObjectHash(s string) ObjectHash {
	var h ObjectHash
	copy(h[:], s)
	return h
}

type Object struct {
	CreatedAt       time.Time `json:"created_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	Hash            string    `json:"hash"`
	MimeType        string    `json:"mime_type"`
	BlockedByReason string    `json:"blocked_by_reason,omitempty"`
	Size            int64     `json:"size"`
	Blocked         bool      `json:"blocked"`
	PublicKey       string    `json:"public_key"`
	Tags            []byte    `json:"tags,omitempty"`
}

type ObjectResponse struct {
	Hash      string `json:"hash"`
	Url       string `json:"url"`
	MimeType  string `json:"mime_type"`
	CreatedAt int64  `json:"created_at"`
}
type ObjectResponseData struct {
	Hash      string `json:"hash"`
	Link      string `json:"link"`
	MimeType  string `json:"mime_type"`
	CreatedAt int64  `json:"created_at"`
}

type BlossomObjectAdmin struct {
	Hash             string        `json:"hash"`
	Extension        string        `json:"extension"`
	Width            sql.NullInt32 `json:"width"`
	Height           sql.NullInt32 `json:"height"`
	DurationMS       sql.NullInt64 `json:"duration_ms"`
	BitrateKbps      sql.NullInt32 `json:"bitrate_kbps"`
	Blurhash         string        `json:"blurhash"`
	ThumbnailHash    string        `json:"thumbnail_hash"`
	OptimizedHash    string        `json:"optimized_hash"`
	HLSManifestHash  string        `json:"hls_manifest_hash"`
	ProcessingStatus string        `json:"processing_status"`
	ProcessingError  string        `json:"processing_error"`
	ExifStatus       string        `json:"exif_status"`
	GPSDetected      bool          `json:"gps_detected"`
	LastDownloadedAt sql.NullTime  `json:"last_downloaded_at"`
	DownloadCount    int64         `json:"download_count"`
	IngressBytes     int64         `json:"ingress_bytes"`
	EgressBytes      int64         `json:"egress_bytes"`
	ReviewState      string        `json:"review_state"`
	FlagReason       string        `json:"flag_reason"`
	NIP94Tags        []byte        `json:"nip94_tags"`
	Mirrors          []byte        `json:"mirrors"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

type BlossomPubkeyQuota struct {
	Pubkey            string        `json:"pubkey"`
	Enabled           bool          `json:"enabled"`
	StorageQuotaBytes sql.NullInt64 `json:"storage_quota_bytes"`
	EgressQuotaBytes  sql.NullInt64 `json:"egress_quota_bytes"`
	Notes             string        `json:"notes"`
	CreatedBy         string        `json:"created_by"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

type BlossomPlan struct {
	ID                string        `json:"id"`
	Name              string        `json:"name"`
	Scope             string        `json:"scope"`
	StorageQuotaBytes sql.NullInt64 `json:"storage_quota_bytes"`
	EgressQuotaBytes  sql.NullInt64 `json:"egress_quota_bytes"`
	Description       string        `json:"description"`
	IsDefault         bool          `json:"is_default"`
	UpdatedBy         string        `json:"updated_by"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

type BlossomAuditRecord struct {
	ID           int64     `json:"id"`
	ActorPubkey  string    `json:"actor_pubkey"`
	Action       string    `json:"action"`
	TargetType   string    `json:"target_type"`
	TargetID     string    `json:"target_id"`
	RequestID    string    `json:"request_id"`
	Payload      []byte    `json:"payload"`
	NostrEventID string    `json:"nostr_event_id"`
	CreatedAt    time.Time `json:"created_at"`
}
