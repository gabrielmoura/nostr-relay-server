package http

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	dbmodel "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gofiber/fiber/v2"
)

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
			response = append(response, profileToAdminProfile(item.Profile, "banned", item.Reason, item.RelatedIDs, item.CreatedAt.Time))
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
			if record.CreatedAt.Valid {
				createdAt = record.CreatedAt.Time
			}
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

		createdAt := ""
		if record.CreatedAt.Valid {
			createdAt = formatTime(record.CreatedAt.Time)
		}

		return c.JSON(fiber.Map{
			"pubkey":      pubkey,
			"banned":      exists,
			"reason":      record.Reason,
			"related_ids": record.RelatedIDs,
			"created_at":  createdAt,
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
		if err := parseAdminJSONBody(c, &req); err != nil {
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
