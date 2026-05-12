package http

import (
	"github.com/gabrielmoura/nostr-relay-server/infra/stream"
	"github.com/gabrielmoura/nostr-relay-server/pkg/nostrpool"
	"github.com/nbd-wtf/go-nostr"
)

type BanRequest struct {
	Reason     string   `json:"reason"`
	RelatedIDs []string `json:"related_ids"`
}

type DisconnectRequest struct {
	Reason string `json:"reason"`
}

type adminPage[T any] struct {
	Items   []T  `json:"items"`
	Total   int  `json:"total"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
}

type adminOverviewResponse struct {
	ActiveConnections int    `json:"active_connections"`
	AuthedConnections int    `json:"authed_connections"`
	LoggedUsers       int    `json:"logged_users"`
	BannedUsers       int    `json:"banned_users"`
	IndexedEvents     int64  `json:"indexed_events"`
	EventsPerMinute   int64  `json:"events_per_minute"`
	RelayStatus       string `json:"relay_status"`
}

type adminStreamConfigResponse struct {
	StreamUp   bool     `json:"stream_up"`
	StreamDown bool     `json:"stream_down"`
	Relays     []string `json:"relays"`
}

type adminStreamCountersResponse struct {
	ForwardedEvents   int64 `json:"forwarded_events"`
	ForwardedRequests int64 `json:"forwarded_requests"`
	ForwardFailures   int64 `json:"forward_failures"`
}

type adminStreamStatusResponse struct {
	Config     adminStreamConfigResponse   `json:"config"`
	Dispatcher stream.DispatcherStats      `json:"dispatcher"`
	Pool       nostrpool.PoolStats         `json:"pool"`
	Counters   adminStreamCountersResponse `json:"counters"`
}

type adminProfileResponse struct {
	Pubkey      string   `json:"pubkey"`
	Npub        string   `json:"npub,omitempty"`
	DisplayName string   `json:"display_name"`
	Handle      string   `json:"handle,omitempty"`
	Picture     string   `json:"picture,omitempty"`
	NIP05       string   `json:"nip05,omitempty"`
	About       string   `json:"about,omitempty"`
	Website     string   `json:"website,omitempty"`
	Bot         bool     `json:"bot,omitempty"`
	Status      string   `json:"status,omitempty"`
	Reason      string   `json:"reason,omitempty"`
	RelatedIDs  []string `json:"related_ids,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty"`
}

type adminLoggedUserResponse struct {
	adminProfileResponse
	ConnectionCount int    `json:"connection_count"`
	LastSeenAt      string `json:"last_seen_at,omitempty"`
	ConnectionState string `json:"connection_state"`
}

type adminConnectionResponse struct {
	WSID              string `json:"ws_id"`
	IP                string `json:"ip"`
	Authed            string `json:"authed,omitempty"`
	SubscriptionCount int    `json:"subscription_count"`
	ConnectedAt       string `json:"connected_at,omitempty"`
	LastSeenAt        string `json:"last_seen_at,omitempty"`
	UserAgent         string `json:"user_agent,omitempty"`
}

type adminEventSearchResponse struct {
	Items   []*nostr.Event `json:"items"`
	Total   int64          `json:"total"`
	Limit   int            `json:"limit"`
	Offset  int            `json:"offset"`
	HasMore bool           `json:"has_more"`
}

type adminKindAggregate struct {
	Kind  int   `json:"kind"`
	Count int64 `json:"count"`
}

type adminAuthorAggregate struct {
	Pubkey      string `json:"pubkey"`
	DisplayName string `json:"display_name,omitempty"`
	Count       int64  `json:"count"`
}

type adminTagAggregate struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

type adminEventAggregatesResponse struct {
	Total      int64                   `json:"total"`
	Kinds      []adminKindAggregate    `json:"kinds"`
	TopAuthors []adminAuthorAggregate  `json:"top_authors"`
	TopTags    []adminTagAggregate     `json:"top_tags"`
	Trends     adminEventTrendResponse `json:"trends"`
}

type adminEventTrendResponse struct {
	TopTagMonth      string `json:"top_tag_month,omitempty"`
	TopTagMonthCount int64  `json:"top_tag_month_count,omitempty"`
	TopTagYear       string `json:"top_tag_year,omitempty"`
	TopTagYearCount  int64  `json:"top_tag_year_count,omitempty"`
	PeakMonth        string `json:"peak_month,omitempty"`
	PeakMonthCount   int64  `json:"peak_month_count,omitempty"`
	PeakYear         string `json:"peak_year,omitempty"`
	PeakYearCount    int64  `json:"peak_year_count,omitempty"`
}

type adminTimelinePoint struct {
	TS    int64 `json:"ts"`
	Count int64 `json:"count"`
}

type adminTimelineResponse struct {
	Bucket string               `json:"bucket"`
	Points []adminTimelinePoint `json:"points"`
}

type adminEventIdentifiers struct {
	Note     string `json:"note,omitempty"`
	Nevent   string `json:"nevent,omitempty"`
	Npub     string `json:"npub,omitempty"`
	Nprofile string `json:"nprofile,omitempty"`
}

type adminEventAuthor struct {
	Pubkey      string `json:"pubkey"`
	DisplayName string `json:"display_name"`
	Picture     string `json:"picture,omitempty"`
	NIP05       string `json:"nip05,omitempty"`
}

type adminEventDetailResponse struct {
	Event       *nostr.Event          `json:"event"`
	Identifiers adminEventIdentifiers `json:"identifiers"`
	Author      adminEventAuthor      `json:"author"`
	Hashtags    []string              `json:"hashtags"`
	ImageURLs   []string              `json:"image_urls"`
}

type adminEventReportItem struct {
	ReportEventID       string `json:"report_event_id"`
	ReporterPubkey      string `json:"reporter_pubkey"`
	ReporterNpub        string `json:"reporter_npub,omitempty"`
	ReporterDisplayName string `json:"reporter_display_name"`
	ReporterPicture     string `json:"reporter_picture,omitempty"`
	ReportedEventID     string `json:"reported_event_id,omitempty"`
	ReportedPubkey      string `json:"reported_pubkey,omitempty"`
	ReportType          string `json:"report_type,omitempty"`
	Content             string `json:"content,omitempty"`
	CreatedAt           int64  `json:"created_at"`
}

type adminEventReportsResponse struct {
	Items []adminEventReportItem `json:"items"`
	Total int64                  `json:"total"`
}

type adminReportedEventResponse struct {
	TargetEventID      string           `json:"target_event_id"`
	TargetPubkey       string           `json:"target_pubkey,omitempty"`
	TargetNevent       string           `json:"target_nevent,omitempty"`
	TargetCreatedAt    int64            `json:"target_created_at,omitempty"`
	TargetAuthor       adminEventAuthor `json:"target_author"`
	ReportCount        int64            `json:"report_count"`
	LastReported       int64            `json:"last_reported"`
	LastReportedAt     string           `json:"last_reported_at,omitempty"`
	TargetCreatedAtISO string           `json:"target_created_at_iso,omitempty"`
	ReportTypes        []string         `json:"report_types"`
	TargetEvent        *nostr.Event     `json:"target_event,omitempty"`
}

type adminReportedTimelinePointResponse struct {
	Bucket string `json:"bucket"`
	Count  int64  `json:"count"`
}

type adminReportedTypeCountResponse struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type adminReportedAuthorCountResponse struct {
	Pubkey      string `json:"pubkey"`
	DisplayName string `json:"display_name,omitempty"`
	Count       int64  `json:"count"`
}

type adminReportedTargetCountResponse struct {
	TargetEventID string `json:"target_event_id"`
	Count         int64  `json:"count"`
}

type adminReportedEventsSummaryResponse struct {
	TotalEvents         int64                                `json:"total_events"`
	TotalReports        int64                                `json:"total_reports"`
	UniqueTargetAuthors int64                                `json:"unique_target_authors"`
	Timeline            []adminReportedTimelinePointResponse `json:"timeline"`
	ReportTypes         []adminReportedTypeCountResponse     `json:"report_types"`
	TopAuthors          []adminReportedAuthorCountResponse   `json:"top_authors"`
	TopTargets          []adminReportedTargetCountResponse   `json:"top_targets"`
}

type adminFetchRelayResult struct {
	Relay  string `json:"relay"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type adminFetchEventRequest struct {
	Relays []string `json:"relays"`
}

type adminFetchEventResponse struct {
	EventID      string                  `json:"event_id,omitempty"`
	SourceRelay  string                  `json:"source_relay,omitempty"`
	Found        bool                    `json:"found"`
	Persisted    bool                    `json:"persisted"`
	RelaysTried  int                     `json:"relays_tried"`
	RelayResults []adminFetchRelayResult `json:"relay_results"`
	Message      string                  `json:"message,omitempty"`
}

type adminImportFileResult struct {
	Filename   string `json:"filename"`
	Total      int    `json:"total"`
	Inserted   int    `json:"inserted"`
	Duplicates int    `json:"duplicates"`
	Invalid    int    `json:"invalid"`
	Error      string `json:"error,omitempty"`
}

type adminImportEventsResponse struct {
	Files []adminImportFileResult `json:"files"`
}

type adminNIP05IdentityResponse struct {
	Name        string   `json:"name"`
	Pubkey      string   `json:"pubkey"`
	Npub        string   `json:"npub,omitempty"`
	DisplayName string   `json:"display_name,omitempty"`
	Picture     string   `json:"picture,omitempty"`
	RelayHints  []string `json:"relay_hints,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
}

type adminNIP05UpsertRequest struct {
	Name   string `json:"name"`
	Pubkey string `json:"pubkey"`
}

type loggedUserAggregate struct {
	Pubkey          string
	ConnectionCount int
	LastSeenAt      int64
	ConnectionState string
}

type adminCacheEnvelope struct {
	Payload string `json:"payload"`
}

type adminWarmupJob struct {
	Name string
	Run  func() error
}
