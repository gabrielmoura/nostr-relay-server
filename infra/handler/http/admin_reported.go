package http

import (
	"strings"

	dbmodel "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
)

func ReportedEvents() fiber.Handler {
	return func(c *fiber.Ctx) error {
		limit := adminLimit(c)
		offset := adminOffset(c)
		filters, err := reportedEventsFiltersFromRequest(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if normalized, err := normalizeSearchQuery(filters.Query); err == nil {
			filters.Query = normalized
		}

		reported, total, err := db.DbQueries.GetReportedEvents(c.UserContext(), filters, limit, offset)
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
			evt, err := db.DbQueries.GetEventByID(c.UserContext(), item.TargetEventID)
			if err == nil {
				targetEvent = evt
				targetCreatedAt = int64(evt.CreatedAt)
			}

			targetPubkey := item.TargetPubkey
			if targetPubkey == "" && targetEvent != nil {
				targetPubkey = targetEvent.PubKey
			}

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
				TargetNevent:       neventFromEventID(item.TargetEventID, targetPubkey),
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

func ReportedEventsSummary() fiber.Handler {
	return func(c *fiber.Ctx) error {
		filters, err := reportedEventsFiltersFromRequest(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if normalized, err := normalizeSearchQuery(filters.Query); err == nil {
			filters.Query = normalized
		}

		summary, err := db.DbQueries.GetReportedEventsSummary(c.UserContext(), filters)
		if err != nil {
			return internalServerError(c, err)
		}

		authorKeys := make([]string, 0, len(summary.TopAuthors))
		for _, author := range summary.TopAuthors {
			if author.Pubkey != "" {
				authorKeys = append(authorKeys, author.Pubkey)
			}
		}

		authorProfiles, err := db.DbQueries.GetProfilesByPublicKeys(c.UserContext(), authorKeys)
		if err != nil {
			return internalServerError(c, err)
		}

		response := adminReportedEventsSummaryResponse{
			TotalEvents:         summary.TotalEvents,
			TotalReports:        summary.TotalReports,
			UniqueTargetAuthors: summary.UniqueTargetAuthors,
			Timeline:            make([]adminReportedTimelinePointResponse, 0, len(summary.Timeline)),
			ReportTypes:         make([]adminReportedTypeCountResponse, 0, len(summary.ReportTypes)),
			TopAuthors:          make([]adminReportedAuthorCountResponse, 0, len(summary.TopAuthors)),
			TopTargets:          make([]adminReportedTargetCountResponse, 0, len(summary.TopTargets)),
		}

		for _, item := range summary.Timeline {
			response.Timeline = append(response.Timeline, adminReportedTimelinePointResponse{Bucket: item.Bucket, Count: item.Count})
		}
		for _, item := range summary.ReportTypes {
			response.ReportTypes = append(response.ReportTypes, adminReportedTypeCountResponse{Name: item.Name, Count: item.Count})
		}
		for _, item := range summary.TopAuthors {
			displayName := item.Pubkey
			if profile, ok := authorProfiles[item.Pubkey]; ok {
				displayName = firstNonEmpty(profile.DisplayName, profile.Name, item.Pubkey)
			}
			response.TopAuthors = append(response.TopAuthors, adminReportedAuthorCountResponse{Pubkey: item.Pubkey, DisplayName: displayName, Count: item.Count})
		}
		for _, item := range summary.TopTargets {
			response.TopTargets = append(response.TopTargets, adminReportedTargetCountResponse{TargetEventID: item.TargetEventID, Count: item.Count})
		}

		return c.JSON(response)
	}
}

func reportedEventsFiltersFromRequest(c *fiber.Ctx) (dbmodel.ReportedEventsFilters, error) {
	filters := dbmodel.ReportedEventsFilters{
		Query:         strings.TrimSpace(c.Query("q")),
		ReportType:    strings.TrimSpace(c.Query("type")),
		TargetPubkey:  strings.TrimSpace(c.Query("target_pubkey")),
		TargetEventID: strings.TrimSpace(c.Query("target_event_id")),
		Since:         int64(c.QueryInt("since", 0)),
		Until:         int64(c.QueryInt("until", 0)),
	}

	if filters.TargetPubkey != "" {
		normalized, err := normalizePublicKey(filters.TargetPubkey)
		if err != nil {
			return dbmodel.ReportedEventsFilters{}, err
		}
		filters.TargetPubkey = normalized
	}
	if filters.TargetEventID != "" {
		normalized, err := normalizeEventID(filters.TargetEventID)
		if err != nil {
			return dbmodel.ReportedEventsFilters{}, err
		}
		filters.TargetEventID = normalized
	}

	return filters, nil
}
