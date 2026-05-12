package http

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/gabrielmoura/nostr-relay-server/infra/nip05"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gofiber/fiber/v2"
)

func NIP05List() fiber.Handler {
	return func(c *fiber.Ctx) error {
		limit := adminLimit(c)
		offset := adminOffset(c)
		query := strings.TrimSpace(c.Query("q"))

		items, total, err := db.DbQueries.ListNIP05Identities(c.UserContext(), query, limit, offset)
		if err != nil {
			return internalServerError(c, err)
		}

		pubkeys := make([]string, 0, len(items))
		for _, item := range items {
			pubkeys = append(pubkeys, item.PublicKey)
		}

		relayHintsByPubkey, err := adminNIP05Service().RelayHintsByPubKeys(c.UserContext(), pubkeys)
		if err != nil {
			return internalServerError(c, err)
		}

		response := make([]adminNIP05IdentityResponse, 0, len(items))
		for _, item := range items {
			response = append(response, adminNIP05IdentityResponse{
				Name:        item.Name,
				Pubkey:      item.PublicKey,
				Npub:        npubFromPublicKey(item.PublicKey),
				DisplayName: firstNonEmpty(item.DisplayName, item.ProfileName, item.PublicKey),
				Picture:     item.Picture,
				RelayHints:  relayHintsByPubkey[strings.ToLower(item.PublicKey)],
				CreatedAt:   formatTime(item.CreatedAt),
				UpdatedAt:   formatTime(item.UpdatedAt),
			})
		}

		return c.JSON(newAdminPage(response, int(total), limit, offset))
	}
}

func NIP05Upsert() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req adminNIP05UpsertRequest
		if err := parseAdminJSONBody(c, &req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		pubkey, err := normalizePublicKey(req.Pubkey)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		normalizedName, err := adminNIP05Service().UpsertIdentity(c.UserContext(), req.Name, pubkey)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		_ = cache.Delete("profile:" + strings.ToLower(pubkey))

		identity, err := db.DbQueries.GetNIP05IdentityByPublicKey(c.UserContext(), strings.ToLower(pubkey))
		if err != nil {
			return internalServerError(c, err)
		}

		relayHintsByPubkey, err := adminNIP05Service().RelayHintsByPubKeys(c.UserContext(), []string{identity.PublicKey})
		if err != nil {
			return internalServerError(c, err)
		}

		return c.JSON(adminNIP05IdentityResponse{
			Name:        normalizedName,
			Pubkey:      identity.PublicKey,
			Npub:        npubFromPublicKey(identity.PublicKey),
			DisplayName: firstNonEmpty(identity.DisplayName, identity.ProfileName, identity.PublicKey),
			Picture:     identity.Picture,
			RelayHints:  relayHintsByPubkey[strings.ToLower(identity.PublicKey)],
			CreatedAt:   formatTime(identity.CreatedAt),
			UpdatedAt:   formatTime(identity.UpdatedAt),
		})
	}
}

func NIP05Delete() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")
		if err := adminNIP05Service().DeleteIdentityByName(c.UserContext(), name); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"name": strings.ToLower(strings.TrimSpace(name)), "deleted": true})
	}
}

func UserNIP05() fiber.Handler {
	return func(c *fiber.Ctx) error {
		pubkey, err := normalizePublicKey(c.Params("pubkey"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		identity, err := db.DbQueries.GetNIP05IdentityByPublicKey(c.UserContext(), strings.ToLower(pubkey))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.JSON(fiber.Map{"pubkey": strings.ToLower(pubkey), "exists": false})
			}
			return internalServerError(c, err)
		}

		relayHintsByPubkey, err := adminNIP05Service().RelayHintsByPubKeys(c.UserContext(), []string{identity.PublicKey})
		if err != nil {
			return internalServerError(c, err)
		}

		return c.JSON(fiber.Map{
			"pubkey":       identity.PublicKey,
			"exists":       true,
			"name":         identity.Name,
			"display_name": firstNonEmpty(identity.DisplayName, identity.ProfileName, identity.PublicKey),
			"picture":      identity.Picture,
			"relay_hints":  relayHintsByPubkey[strings.ToLower(identity.PublicKey)],
			"created_at":   formatTime(identity.CreatedAt),
			"updated_at":   formatTime(identity.UpdatedAt),
		})
	}
}

func adminNIP05Service() *nip05.Service {
	return nip05.NewService(db.DbQueries)
}
