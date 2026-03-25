package http

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	dbmodel "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/infra/stream"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/pkg/nostrpool"
	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/prometheus/client_golang/prometheus"
	promdto "github.com/prometheus/client_model/go"
)

const (
	defaultAdminLimit = 100
	maxAdminLimit     = 250
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
	Pubkey string `json:"pubkey"`
	Count  int64  `json:"count"`
}

type adminTagAggregate struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

type adminEventAggregatesResponse struct {
	Total      int64                  `json:"total"`
	Kinds      []adminKindAggregate   `json:"kinds"`
	TopAuthors []adminAuthorAggregate `json:"top_authors"`
	TopTags    []adminTagAggregate    `json:"top_tags"`
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
	Persisted    bool                    `json:"persisted"`
	RelaysTried  int                     `json:"relays_tried"`
	RelayResults []adminFetchRelayResult `json:"relay_results"`
}

var errAdminEventNotFoundOnRelays = errors.New("event not found on provided relays")

var defaultAdminFetchRelays = []string{
	"wss://relay.damus.io",
	"wss://relay.primal.net",
	"wss://nos.lol",
	"wss://relay.nostr.band",
	"wss://nostr.mom",
}

func AdminTokenMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if cfg.AdminToken == "" {
			return c.Next()
		}
		if c.Get("X-Admin-Token") != cfg.AdminToken {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid admin token"})
		}
		return c.Next()
	}
}

func AdminIndex() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendString("Admin Interface")
	}
}

func AdminOverview() fiber.Handler {
	return func(c *fiber.Ctx) error {
		activeConnections := listener.ActiveConnections()
		authedConnections := listener.AuthedConnections()
		loggedUsers := aggregateLoggedUsers(authedConnections)

		bannedUsers, bannedTotal, err := db.DbQueries.ListBannedUsers(c.UserContext(), "", 1, 0)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return internalServerError(c, err)
		}
		_ = bannedUsers

		indexedEvents, err := db.DbQueries.CountAllEvents(c.UserContext())
		if err != nil {
			return internalServerError(c, err)
		}

		eventsPerMinute, err := db.DbQueries.CountEventsSince(c.UserContext(), time.Now().UTC().Add(-time.Minute).Unix())
		if err != nil {
			return internalServerError(c, err)
		}

		return c.JSON(adminOverviewResponse{
			ActiveConnections: len(activeConnections),
			AuthedConnections: len(authedConnections),
			LoggedUsers:       len(loggedUsers),
			BannedUsers:       int(bannedTotal),
			IndexedEvents:     indexedEvents,
			EventsPerMinute:   eventsPerMinute,
			RelayStatus:       "operational",
		})
	}
}

func StreamStatus() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(adminStreamStatusResponse{
			Config: adminStreamConfigResponse{
				StreamUp:   config.Cfg.Stream.StreamUp,
				StreamDown: config.Cfg.Stream.StreamDown,
				Relays:     append([]string{}, config.Cfg.Stream.Relays...),
			},
			Dispatcher: stream.Snapshot(),
			Pool:       nostrpool.Stats(),
			Counters: adminStreamCountersResponse{
				ForwardedEvents:   int64(counterValue(metrics.NostrRelayEventForwardedTotal)),
				ForwardedRequests: int64(counterValue(metrics.NostrRelayRequestForwardedTotal)),
				ForwardFailures:   int64(counterValue(metrics.NostrRelayEventForwardedFailuresTotal)),
			},
		})
	}
}

func ActiveConnections() fiber.Handler {
	return func(c *fiber.Ctx) error {
		connections := listener.ActiveConnections()
		limit := adminLimit(c)
		offset := adminOffset(c)

		return c.JSON(newAdminPage(
			mapConnectionsPage(connections, limit, offset),
			len(connections),
			limit,
			offset,
		))
	}
}

func AuthedConnections() fiber.Handler {
	return func(c *fiber.Ctx) error {
		connections := listener.AuthedConnections()
		limit := adminLimit(c)
		offset := adminOffset(c)

		return c.JSON(newAdminPage(
			mapConnectionsPage(connections, limit, offset),
			len(connections),
			limit,
			offset,
		))
	}
}

func DisconnectConnection() fiber.Handler {
	return func(c *fiber.Ctx) error {
		wsID := c.Params("wsid")
		if wsID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing wsid"})
		}

		var req DisconnectRequest
		if len(c.Body()) > 0 {
			if err := c.BodyParser(&req); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
			}
		}
		_ = req

		if !listener.Disconnect(wsID) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "connection not found"})
		}

		return c.JSON(fiber.Map{"ws_id": wsID, "disconnected": true})
	}
}

func LoggedUsers() fiber.Handler {
	return func(c *fiber.Ctx) error {
		connections := listener.AuthedConnections()
		users := aggregateLoggedUsers(connections)
		limit := adminLimit(c)
		offset := adminOffset(c)
		window := paginate(users, limit, offset)

		pubkeys := make([]string, 0, len(window))
		for _, item := range window {
			pubkeys = append(pubkeys, item.Pubkey)
		}

		profiles, err := db.DbQueries.GetProfilesByPublicKeys(c.UserContext(), pubkeys)
		if err != nil {
			return internalServerError(c, err)
		}

		items := make([]adminLoggedUserResponse, 0, len(window))
		for _, item := range window {
			profile, ok := profiles[item.Pubkey]
			if !ok {
				profile = dbmodel.Profile{PublicKey: item.Pubkey, Name: item.Pubkey}
			}
			items = append(items, adminLoggedUserResponse{
				adminProfileResponse: profileToAdminProfile(profile, "online", "", nil, time.Time{}),
				ConnectionCount:      item.ConnectionCount,
				LastSeenAt:           formatUnix(item.LastSeenAt),
				ConnectionState:      item.ConnectionState,
			})
		}

		return c.JSON(newAdminPage(items, len(users), limit, offset))
	}
}

func BannedUsers() fiber.Handler {
	return func(c *fiber.Ctx) error {
		limit := adminLimit(c)
		offset := adminOffset(c)
		items, total, err := db.DbQueries.ListBannedUsers(c.UserContext(), c.Query("q"), limit, offset)
		if err != nil {
			return internalServerError(c, err)
		}

		response := make([]adminProfileResponse, 0, len(items))
		for _, item := range items {
			response = append(response, profileToAdminProfile(item.Profile, "banned", item.Reason, item.RelatedIDs, item.CreatedAt))
		}

		return c.JSON(newAdminPage(response, int(total), limit, offset))
	}
}

func SearchUsers() fiber.Handler {
	return func(c *fiber.Ctx) error {
		limit := adminLimit(c)
		offset := adminOffset(c)
		query := c.Query("q")
		if normalized, err := normalizeSearchQuery(query); err == nil {
			query = normalized
		}

		profiles, total, err := db.DbQueries.SearchProfiles(c.UserContext(), query, limit, offset)
		if err != nil {
			return internalServerError(c, err)
		}

		pubkeys := make([]string, 0, len(profiles))
		for _, profile := range profiles {
			pubkeys = append(pubkeys, profile.PublicKey)
		}
		banRecords, err := db.DbQueries.GetLatestBanRecordsByKeys(c.UserContext(), pubkeys)
		if err != nil {
			return internalServerError(c, err)
		}

		items := make([]adminProfileResponse, 0, len(profiles))
		for _, profile := range profiles {
			status := "active"
			reason := ""
			if record, ok := banRecords[profile.PublicKey]; ok {
				status = "banned"
				reason = record.Reason
			}
			items = append(items, profileToAdminProfile(profile, status, reason, nil, time.Time{}))
		}

		return c.JSON(newAdminPage(items, int(total), limit, offset))
	}
}

func UserProfile() fiber.Handler {
	return func(c *fiber.Ctx) error {
		pubkey, err := normalizePublicKey(c.Params("pubkey"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		profile, err := db.DbQueries.GetProfileByPublicKey(c.UserContext(), pubkey)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return internalServerError(c, err)
		}
		if errors.Is(err, sql.ErrNoRows) {
			profile = dbmodel.Profile{PublicKey: pubkey, Name: pubkey}
		}

		record, banned, err := db.DbQueries.GetLatestBanRecordByKey(c.UserContext(), pubkey)
		if err != nil {
			return internalServerError(c, err)
		}

		status := "active"
		reason := ""
		var relatedIDs []string
		var createdAt time.Time
		if banned {
			status = "banned"
			reason = record.Reason
			relatedIDs = record.RelatedIDs
			createdAt = record.CreatedAt
		}

		return c.JSON(profileToAdminProfile(profile, status, reason, relatedIDs, createdAt))
	}
}

func BanStatus() fiber.Handler {
	return func(c *fiber.Ctx) error {
		pubkey, err := normalizePublicKey(c.Params("pubkey"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		record, exists, err := db.DbQueries.GetLatestBanRecordByKey(c.UserContext(), pubkey)
		if err != nil {
			return internalServerError(c, err)
		}

		return c.JSON(fiber.Map{
			"pubkey":      pubkey,
			"banned":      exists,
			"reason":      record.Reason,
			"related_ids": record.RelatedIDs,
			"created_at":  record.CreatedAt,
		})
	}
}

func BanUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		pubkey, err := normalizePublicKey(c.Params("pubkey"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		var req BanRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		if req.Reason == "" {
			req.Reason = "admin ban"
		}

		if err := db.DbQueries.InsertUserProfile(c.UserContext(), &dbmodel.Profile{PublicKey: pubkey, Name: pubkey}); err != nil {
			return internalServerError(c, err)
		}
		if err := db.DbQueries.BanUserByPubKey(c.UserContext(), pubkey, req.Reason, req.RelatedIDs); err != nil {
			return internalServerError(c, err)
		}

		return c.JSON(fiber.Map{"pubkey": pubkey, "banned": true, "reason": req.Reason, "related_ids": req.RelatedIDs})
	}
}

func UnbanUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		pubkey, err := normalizePublicKey(c.Params("pubkey"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if err := db.DbQueries.UnbanUserByPubKey(c.UserContext(), pubkey); err != nil {
			return internalServerError(c, err)
		}
		return c.JSON(fiber.Map{"pubkey": pubkey, "banned": false})
	}
}

func SearchEvents() fiber.Handler {
	return func(c *fiber.Ctx) error {
		filter := buildAdminFilter(c)
		offset := adminOffset(c)
		events, total, err := db.DbQueries.QueryEventsWindow(c.UserContext(), filter, offset)
		if err != nil {
			return internalServerError(c, err)
		}

		return c.JSON(adminEventSearchResponse{
			Items:   events,
			Total:   total,
			Limit:   filter.Limit,
			Offset:  offset,
			HasMore: int64(offset+len(events)) < total,
		})
	}
}

func SearchEventsAggregates() fiber.Handler {
	return func(c *fiber.Ctx) error {
		filter := buildAdminFilter(c)
		filter.Limit = maxAdminLimit
		events, _, err := db.DbQueries.QueryEventsWindow(c.UserContext(), filter, 0)
		if err != nil {
			return internalServerError(c, err)
		}

		kindCounts := make(map[int]int64, 16)
		authorCounts := make(map[string]int64, 64)
		tagCounts := make(map[string]int64, 128)
		for _, evt := range events {
			kindCounts[evt.Kind]++
			authorCounts[evt.PubKey]++
			for _, tag := range extractHashtags(evt) {
				tagCounts[tag]++
			}
		}

		return c.JSON(adminEventAggregatesResponse{
			Total:      int64(len(events)),
			Kinds:      topKinds(kindCounts, 8),
			TopAuthors: topAuthors(authorCounts, 8),
			TopTags:    topTags(tagCounts, 12),
		})
	}
}

func SearchEventsTimeline() fiber.Handler {
	return func(c *fiber.Ctx) error {
		bucket := c.Query("bucket", "hour")
		if bucket != "hour" && bucket != "day" {
			bucket = "hour"
		}

		filter := buildAdminFilter(c)
		filter.Limit = maxAdminLimit
		events, _, err := db.DbQueries.QueryEventsWindow(c.UserContext(), filter, 0)
		if err != nil {
			return internalServerError(c, err)
		}

		points := make(map[int64]int64, len(events))
		var step int64 = 3600
		if bucket == "day" {
			step = 86400
		}
		for _, evt := range events {
			ts := int64(evt.CreatedAt)
			rounded := ts - (ts % step)
			points[rounded]++
		}

		keys := make([]int64, 0, len(points))
		for k := range points {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

		result := make([]adminTimelinePoint, 0, len(keys))
		for _, k := range keys {
			result = append(result, adminTimelinePoint{TS: k, Count: points[k]})
		}

		return c.JSON(adminTimelineResponse{Bucket: bucket, Points: result})
	}
}

func EventDetail() fiber.Handler {
	return func(c *fiber.Ctx) error {
		eventID := strings.TrimSpace(c.Params("id"))
		evt, err := db.DbQueries.GetEventByID(c.UserContext(), eventID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "event not found"})
			}
			return internalServerError(c, err)
		}

		authorProfile, err := db.DbQueries.GetProfileByPublicKey(c.UserContext(), evt.PubKey)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return internalServerError(c, err)
		}
		if errors.Is(err, sql.ErrNoRows) {
			authorProfile = dbmodel.Profile{PublicKey: evt.PubKey, Name: evt.PubKey}
		}

		return c.JSON(adminEventDetailResponse{
			Event: evt,
			Identifiers: adminEventIdentifiers{
				Npub: npubFromPublicKey(evt.PubKey),
			},
			Author: adminEventAuthor{
				Pubkey:      evt.PubKey,
				DisplayName: firstNonEmpty(authorProfile.DisplayName, authorProfile.Name, evt.PubKey),
				Picture:     authorProfile.Picture,
				NIP05:       authorProfile.Nip05,
			},
			Hashtags:  extractHashtags(evt),
			ImageURLs: extractImageURLs(evt),
		})
	}
}

func EventReports() fiber.Handler {
	return func(c *fiber.Ctx) error {
		eventID := strings.TrimSpace(c.Params("id"))
		limit := adminLimit(c)
		offset := adminOffset(c)

		reports, total, err := db.DbQueries.GetReportsForEvent(c.UserContext(), eventID, limit, offset)
		if err != nil {
			return internalServerError(c, err)
		}

		reporterKeys := make([]string, 0, len(reports))
		for _, report := range reports {
			reporterKeys = append(reporterKeys, report.PubKey)
		}
		reporterProfiles, err := db.DbQueries.GetProfilesByPublicKeys(c.UserContext(), reporterKeys)
		if err != nil {
			return internalServerError(c, err)
		}

		items := make([]adminEventReportItem, 0, len(reports))
		for _, report := range reports {
			targetEventID, targetPubkey, reportType := extractReportCore(report)
			profile := reporterProfiles[report.PubKey]
			items = append(items, adminEventReportItem{
				ReportEventID:       report.ID,
				ReporterPubkey:      report.PubKey,
				ReporterNpub:        npubFromPublicKey(report.PubKey),
				ReporterDisplayName: firstNonEmpty(profile.DisplayName, profile.Name, report.PubKey),
				ReporterPicture:     profile.Picture,
				ReportedEventID:     targetEventID,
				ReportedPubkey:      targetPubkey,
				ReportType:          reportType,
				Content:             report.Content,
				CreatedAt:           int64(report.CreatedAt),
			})
		}

		return c.JSON(adminEventReportsResponse{Items: items, Total: total})
	}
}

func FetchEventFromRelays() fiber.Handler {
	return func(c *fiber.Ctx) error {
		eventID := strings.ToLower(strings.TrimSpace(c.Params("id")))
		if !eventIDPattern.MatchString(eventID) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid event id"})
		}

		var req adminFetchEventRequest
		if len(c.Body()) > 0 {
			if err := c.BodyParser(&req); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
			}
		}

		relays := mergeFetchRelayList(req.Relays, config.Cfg.Stream.Relays, defaultAdminFetchRelays)
		event, sourceRelay, tried, relayResults, err := fetchEventFromRelays(c.UserContext(), eventID, relays)
		if err != nil {
			if errors.Is(err, errAdminEventNotFoundOnRelays) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "event not found on provided relays", "relays_tried": tried, "relay_results": relayResults})
			}
			return internalServerError(c, err)
		}

		persisted := true
		if err := db.DbQueries.InsertEvent(c.UserContext(), event); err != nil {
			if errors.Is(err, dbmodel.ErrDupEvent) {
				persisted = false
			} else {
				return internalServerError(c, err)
			}
		}

		return c.JSON(adminFetchEventResponse{
			EventID:      event.ID,
			SourceRelay:  sourceRelay,
			Persisted:    persisted,
			RelaysTried:  tried,
			RelayResults: relayResults,
		})
	}
}

func ReportedEvents() fiber.Handler {
	return func(c *fiber.Ctx) error {
		limit := adminLimit(c)
		offset := adminOffset(c)
		q := strings.TrimSpace(c.Query("q"))
		reportType := strings.TrimSpace(c.Query("type"))

		if normalized, err := normalizeSearchQuery(q); err == nil {
			q = normalized
		}

		reported, total, err := db.DbQueries.GetReportedEvents(c.UserContext(), q, reportType, limit, offset)
		if err != nil {
			return internalServerError(c, err)
		}

		authorKeys := make([]string, 0, len(reported))
		for _, item := range reported {
			if item.TargetPubkey != "" {
				authorKeys = append(authorKeys, item.TargetPubkey)
			}
		}

		authorProfiles, err := db.DbQueries.GetProfilesByPublicKeys(c.UserContext(), authorKeys)
		if err != nil {
			return internalServerError(c, err)
		}

		items := make([]adminReportedEventResponse, 0, len(reported))
		for _, item := range reported {
			var targetEvent *nostr.Event
			targetCreatedAt := int64(0)
			targetNevent := ""
			evt, err := db.DbQueries.GetEventByID(c.UserContext(), item.TargetEventID)
			if err == nil {
				targetEvent = evt
				targetCreatedAt = int64(evt.CreatedAt)
			}

			targetPubkey := item.TargetPubkey
			if targetPubkey == "" && targetEvent != nil {
				targetPubkey = targetEvent.PubKey
			}

			targetNevent = neventFromEventID(item.TargetEventID, targetPubkey)

			profile, ok := authorProfiles[targetPubkey]
			if !ok {
				profile = dbmodel.Profile{PublicKey: targetPubkey, Name: targetPubkey}
			}

			displayName := firstNonEmpty(profile.DisplayName, profile.Name, targetPubkey)
			if displayName == "" {
				displayName = "autor desconhecido"
			}

			items = append(items, adminReportedEventResponse{
				TargetEventID:      item.TargetEventID,
				TargetPubkey:       targetPubkey,
				TargetNevent:       targetNevent,
				TargetCreatedAt:    targetCreatedAt,
				TargetCreatedAtISO: formatUnix(targetCreatedAt),
				TargetAuthor: adminEventAuthor{
					Pubkey:      targetPubkey,
					DisplayName: displayName,
					Picture:     profile.Picture,
					NIP05:       profile.Nip05,
				},
				ReportCount:    item.ReportCount,
				LastReported:   item.LastReported,
				LastReportedAt: formatUnix(item.LastReported),
				ReportTypes:    item.ReportTypes,
				TargetEvent:    targetEvent,
			})
		}

		return c.JSON(newAdminPage(items, int(total), limit, offset))
	}
}

func buildAdminFilter(c *fiber.Ctx) nostr.Filter {
	queryArgs := c.Request().URI().QueryArgs()
	tags := make(nostr.TagMap)
	for _, raw := range queryArgs.PeekMulti("tag") {
		part := string(raw)
		pieces := strings.SplitN(part, ":", 2)
		if len(pieces) != 2 {
			continue
		}
		key := strings.TrimPrefix(pieces[0], "#")
		tags[key] = append(tags[key], pieces[1])
	}

	if rawTags := strings.TrimSpace(c.Query("tags")); rawTags != "" {
		for _, tag := range strings.Split(rawTags, ",") {
			normalized := strings.TrimSpace(strings.TrimPrefix(tag, "#"))
			if normalized == "" {
				continue
			}
			tags["t"] = append(tags["t"], normalized)
		}
	}

	authors := queryValues(queryArgs.PeekMulti("author"))
	normalizedAuthors := make([]string, 0, len(authors))
	for _, author := range authors {
		if normalized, err := normalizePublicKey(author); err == nil {
			normalizedAuthors = append(normalizedAuthors, normalized)
		}
	}

	search := c.Query("q")
	if normalized, err := normalizeSearchQuery(search); err == nil {
		search = normalized
	}
	if eventIDPattern.MatchString(search) {
		normalizedAuthors = append(normalizedAuthors, search)
		search = ""
	}

	return nostr.Filter{
		Search:  search,
		Authors: normalizedAuthors,
		Kinds:   parseKinds(queryArgs.PeekMulti("kind")),
		Tags:    tags,
		Limit:   adminLimit(c),
	}
}

func queryValues(values [][]byte) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}

func parseKinds(values [][]byte) []int {
	result := make([]int, 0, len(values))
	for _, value := range values {
		kind, err := strconv.Atoi(string(value))
		if err == nil {
			result = append(result, kind)
		}
	}
	return result
}

func normalizeRelayURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false
	}

	scheme := strings.ToLower(parsed.Scheme)
	switch scheme {
	case "wss", "ws":
		return parsed.String(), true
	case "https":
		parsed.Scheme = "wss"
		return parsed.String(), true
	case "http":
		parsed.Scheme = "ws"
		return parsed.String(), true
	default:
		return "", false
	}
}

func mergeFetchRelayList(groups ...[]string) []string {
	seen := make(map[string]struct{}, 16)
	merged := make([]string, 0, 16)

	for _, relays := range groups {
		for _, relay := range relays {
			normalized, ok := normalizeRelayURL(relay)
			if !ok {
				continue
			}
			if _, exists := seen[normalized]; exists {
				continue
			}
			seen[normalized] = struct{}{}
			merged = append(merged, normalized)
		}
	}

	return merged
}

func fetchEventFromRelays(ctx context.Context, eventID string, relays []string) (*nostr.Event, string, int, []adminFetchRelayResult, error) {
	if len(relays) == 0 {
		return nil, "", 0, []adminFetchRelayResult{}, errAdminEventNotFoundOnRelays
	}

	results := make([]adminFetchRelayResult, 0, len(relays))

	for _, relayURL := range relays {
		relayCtx, cancelRelay := context.WithTimeout(ctx, 5*time.Second)
		relay, err := nostr.RelayConnect(relayCtx, relayURL)
		if err != nil {
			cancelRelay()
			results = append(results, adminFetchRelayResult{Relay: relayURL, Status: "connect_error", Error: err.Error()})
			continue
		}

		events, err := relay.QuerySync(relayCtx, nostr.Filter{IDs: []string{eventID}, Limit: 1})
		_ = relay.Close()
		cancelRelay()

		if err != nil {
			results = append(results, adminFetchRelayResult{Relay: relayURL, Status: "query_error", Error: err.Error()})
			continue
		}

		if len(events) == 0 {
			results = append(results, adminFetchRelayResult{Relay: relayURL, Status: "not_found"})
			continue
		}

		results = append(results, adminFetchRelayResult{Relay: relayURL, Status: "found"})
		return events[0], relayURL, len(results), results, nil
	}

	return nil, "", len(results), results, errAdminEventNotFoundOnRelays
}

func adminLimit(c *fiber.Ctx) int {
	limit := c.QueryInt("limit", defaultAdminLimit)
	if limit <= 0 {
		return defaultAdminLimit
	}
	if limit > maxAdminLimit {
		return maxAdminLimit
	}
	return limit
}

func adminOffset(c *fiber.Ctx) int {
	offset := c.QueryInt("offset", 0)
	if offset < 0 {
		return 0
	}
	return offset
}

func normalizePublicKey(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("missing public key")
	}
	if strings.HasPrefix(value, "npub") {
		prefix, decoded, err := nip19.Decode(value)
		if err != nil {
			return "", fmt.Errorf("invalid npub: %w", err)
		}
		if prefix != "npub" {
			return "", fmt.Errorf("invalid public key prefix")
		}
		pubkey, ok := decoded.(string)
		if !ok {
			return "", fmt.Errorf("invalid npub payload")
		}
		return pubkey, nil
	}
	return value, nil
}

func normalizeSearchQuery(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(value, "npub") {
		return normalizePublicKey(value)
	}
	return value, nil
}

func formatUnix(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.Unix(value, 0).UTC().Format(time.RFC3339)
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func npubFromPublicKey(pubkey string) string {
	npub, err := nip19.EncodePublicKey(pubkey)
	if err != nil {
		return ""
	}
	return npub
}

func neventFromEventID(eventID string, pubkey string) string {
	nevent, err := nip19.EncodeEvent(eventID, []string{}, pubkey)
	if err != nil {
		return ""
	}
	return nevent
}

func profileToAdminProfile(
	profile dbmodel.Profile,
	status string,
	reason string,
	relatedIDs []string,
	createdAt time.Time,
) adminProfileResponse {
	displayName := strings.TrimSpace(profile.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(profile.Name)
	}
	if displayName == "" {
		displayName = profile.PublicKey
	}

	handle := strings.TrimSpace(profile.Name)
	handle = strings.TrimPrefix(handle, "@")
	handle = strings.ReplaceAll(handle, " ", "_")

	return adminProfileResponse{
		Pubkey:      profile.PublicKey,
		Npub:        npubFromPublicKey(profile.PublicKey),
		DisplayName: displayName,
		Handle:      handle,
		Picture:     profile.Picture,
		NIP05:       profile.Nip05,
		About:       profile.About,
		Website:     profile.Website,
		Bot:         profile.Bot,
		Status:      status,
		Reason:      reason,
		RelatedIDs:  relatedIDs,
		CreatedAt:   formatTime(createdAt),
	}
}

func mapConnectionsPage(connections []listener.ConnectionInfo, limit int, offset int) []adminConnectionResponse {
	window := paginate(connections, limit, offset)
	items := make([]adminConnectionResponse, 0, len(window))
	for _, item := range window {
		items = append(items, adminConnectionResponse{
			WSID:              item.WSID,
			IP:                item.IP,
			Authed:            item.Authed,
			SubscriptionCount: item.SubscriptionCount,
			ConnectedAt:       formatUnix(item.ConnectedAt),
			LastSeenAt:        formatUnix(item.LastSeenAt),
			UserAgent:         item.UserAgent,
		})
	}
	return items
}

type loggedUserAggregate struct {
	Pubkey          string
	ConnectionCount int
	LastSeenAt      int64
	ConnectionState string
}

func aggregateLoggedUsers(connections []listener.ConnectionInfo) []loggedUserAggregate {
	grouped := make(map[string]loggedUserAggregate, len(connections))
	for _, connection := range connections {
		if connection.Authed == "" {
			continue
		}
		current := grouped[connection.Authed]
		current.Pubkey = connection.Authed
		current.ConnectionCount++
		if connection.LastSeenAt > current.LastSeenAt {
			current.LastSeenAt = connection.LastSeenAt
		}
		if connection.SubscriptionCount > 1 {
			current.ConnectionState = "stable"
		} else if current.ConnectionState == "" {
			current.ConnectionState = "attention"
		}
		grouped[connection.Authed] = current
	}

	users := make([]loggedUserAggregate, 0, len(grouped))
	for _, item := range grouped {
		if item.ConnectionState == "" {
			item.ConnectionState = "attention"
		}
		users = append(users, item)
	}

	sort.Slice(users, func(i, j int) bool {
		if users[i].ConnectionCount == users[j].ConnectionCount {
			if users[i].LastSeenAt == users[j].LastSeenAt {
				return users[i].Pubkey < users[j].Pubkey
			}
			return users[i].LastSeenAt > users[j].LastSeenAt
		}
		return users[i].ConnectionCount > users[j].ConnectionCount
	})

	return users
}

func newAdminPage[T any](items []T, total int, limit int, offset int) adminPage[T] {
	return adminPage[T]{
		Items:   items,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+len(items) < total,
	}
}

func paginate[T any](items []T, limit int, offset int) []T {
	if offset >= len(items) {
		return []T{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

func internalServerError(c *fiber.Ctx, err error) error {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func topKinds(counts map[int]int64, limit int) []adminKindAggregate {
	items := make([]adminKindAggregate, 0, len(counts))
	for kind, count := range counts {
		items = append(items, adminKindAggregate{Kind: kind, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Kind < items[j].Kind
		}
		return items[i].Count > items[j].Count
	})
	if len(items) > limit {
		return items[:limit]
	}
	return items
}

func topAuthors(counts map[string]int64, limit int) []adminAuthorAggregate {
	items := make([]adminAuthorAggregate, 0, len(counts))
	for pubkey, count := range counts {
		items = append(items, adminAuthorAggregate{Pubkey: pubkey, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Pubkey < items[j].Pubkey
		}
		return items[i].Count > items[j].Count
	})
	if len(items) > limit {
		return items[:limit]
	}
	return items
}

func topTags(counts map[string]int64, limit int) []adminTagAggregate {
	items := make([]adminTagAggregate, 0, len(counts))
	for tag, count := range counts {
		items = append(items, adminTagAggregate{Tag: tag, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Tag < items[j].Tag
		}
		return items[i].Count > items[j].Count
	})
	if len(items) > limit {
		return items[:limit]
	}
	return items
}

func extractHashtags(event *nostr.Event) []string {
	result := make([]string, 0, 8)
	seen := make(map[string]struct{}, 8)
	for _, tag := range event.Tags {
		if len(tag) > 1 && tag[0] == "t" {
			normalized := strings.ToLower(strings.TrimSpace(tag[1]))
			if normalized == "" {
				continue
			}
			if _, ok := seen[normalized]; ok {
				continue
			}
			seen[normalized] = struct{}{}
			result = append(result, normalized)
		}
	}
	return result
}

var imageURLPattern = regexp.MustCompile(`https?://[^\s"'<>]+\.(?:png|jpg|jpeg|gif|webp|avif)`)
var eventIDPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func extractImageURLs(event *nostr.Event) []string {
	matches := imageURLPattern.FindAllString(event.Content, -1)
	if len(matches) == 0 {
		return []string{}
	}
	unique := make(map[string]struct{}, len(matches))
	result := make([]string, 0, len(matches))
	for _, raw := range matches {
		if _, ok := unique[raw]; ok {
			continue
		}
		unique[raw] = struct{}{}
		result = append(result, raw)
	}
	return result
}

func extractReportCore(event *nostr.Event) (targetEventID string, targetPubkey string, reportType string) {
	for _, tag := range event.Tags {
		if len(tag) > 1 {
			switch tag[0] {
			case "e":
				if targetEventID == "" {
					targetEventID = tag[1]
				}
				if len(tag) > 2 && reportType == "" {
					reportType = tag[2]
				}
			case "p":
				if targetPubkey == "" {
					targetPubkey = tag[1]
				}
				if len(tag) > 2 && reportType == "" {
					reportType = tag[2]
				}
			case "x":
				if len(tag) > 2 && reportType == "" {
					reportType = tag[2]
				}
			}
		}
	}
	return targetEventID, targetPubkey, reportType
}

func counterValue(counter prometheus.Counter) float64 {
	metric := &promdto.Metric{}
	if err := counter.Write(metric); err != nil {
		return 0
	}
	if metric.Counter == nil {
		return 0
	}
	return metric.Counter.GetValue()
}
