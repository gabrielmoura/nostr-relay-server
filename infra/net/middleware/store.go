package middleware

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// BlockIfStoreNotEnabled checks if the store is enabled and returns a 403 Forbidden response if not.
func BlockIfStoreNotEnabled(c *fiber.Ctx) error {
	if !config.Cfg.Store.Enabled {
		log.Logger.Warn("Store is not enabled",
			zap.String("remote_ip", c.IP()),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
		)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Store is not enabled",
		})
	}
	return c.Next()
}
