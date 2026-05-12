package http

import (
	"context"
	"database/sql"
	"errors"

	dbmodel "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
)

func SearchEvents() fiber.Handler {
	return func(c *fiber.Ctx) error {
		response, err := loadSearchEventsResponse(c.UserContext(), buildAdminFilter(c), adminOffset(c))
		if err != nil {
			return internalServerError(c, err)
		}
		return c.JSON(response)
	}
}

func SearchEventsAggregates() fiber.Handler {
	return func(c *fiber.Ctx) error {
		response, err := loadSearchEventsAggregatesResponse(c.UserContext(), buildAdminFilter(c))
		if err != nil {
			return internalServerError(c, err)
		}
		return c.JSON(response)
	}
}

func SearchEventsTimeline() fiber.Handler {
	return func(c *fiber.Ctx) error {
		response, err := loadSearchEventsTimelineResponse(
			c.UserContext(),
			buildAdminFilter(c),
			normalizeTimelineBucket(c.Query("bucket", "hour")),
		)
		if err != nil {
			return internalServerError(c, err)
		}
		return c.JSON(response)
	}
}

func EventDetail() fiber.Handler {
	return func(c *fiber.Ctx) error {
		eventID := c.Params("id")
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
		eventID := c.Params("id")
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

func loadSearchEventsResponse(ctx context.Context, filter nostr.Filter, offset int) (adminEventSearchResponse, error) {
	return loadAdminCachedPayload(adminSearchPageCacheKey(filter, offset), func() (adminEventSearchResponse, error) {
		items, total, err := db.DbQueries.QueryEventsWindow(ctx, filter, offset)
		if err != nil {
			return adminEventSearchResponse{}, err
		}

		return adminEventSearchResponse{
			Items:   items,
			Total:   total,
			Limit:   filter.Limit,
			Offset:  offset,
			HasMore: int64(offset+len(items)) < total,
		}, nil
	})
}

func loadSearchEventsAggregatesResponse(ctx context.Context, filter nostr.Filter) (adminEventAggregatesResponse, error) {
	return loadAdminCachedPayload(adminSearchAggregatesCacheKey(filter), func() (adminEventAggregatesResponse, error) {
		aggregates, err := db.DbQueries.GetEventAggregates(ctx, filter)
		if err != nil {
			return adminEventAggregatesResponse{}, err
		}

		response := adminEventAggregatesResponse{
			Total:      aggregates.Total,
			Kinds:      make([]adminKindAggregate, 0, len(aggregates.Kinds)),
			TopAuthors: make([]adminAuthorAggregate, 0, len(aggregates.TopAuthors)),
			TopTags:    make([]adminTagAggregate, 0, len(aggregates.TopTags)),
			Trends:     mapTrendAggregate(aggregates.Trends),
		}

		for _, item := range aggregates.Kinds {
			response.Kinds = append(response.Kinds, adminKindAggregate(item))
		}
		for _, item := range aggregates.TopAuthors {
			response.TopAuthors = append(response.TopAuthors, adminAuthorAggregate(item))
		}
		for _, item := range aggregates.TopTags {
			response.TopTags = append(response.TopTags, adminTagAggregate(item))
		}

		return response, nil
	})
}

func loadSearchEventsTimelineResponse(ctx context.Context, filter nostr.Filter, bucket string) (adminTimelineResponse, error) {
	return loadAdminCachedPayload(adminSearchTimelineCacheKey(filter, bucket), func() (adminTimelineResponse, error) {
		points, err := db.DbQueries.GetEventTimeline(ctx, filter, bucket)
		if err != nil {
			return adminTimelineResponse{}, err
		}

		response := adminTimelineResponse{Bucket: bucket, Points: make([]adminTimelinePoint, 0, len(points))}
		for _, item := range points {
			response.Points = append(response.Points, adminTimelinePoint(item))
		}

		return response, nil
	})
}

func mapTrendAggregate(trend dbmodel.EventTrendAggregate) adminEventTrendResponse {
	return adminEventTrendResponse(trend)
}
