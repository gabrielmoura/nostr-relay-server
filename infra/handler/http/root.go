package http

import (
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/util"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func TermsOfService(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Redirect(cfg.Store.APIPath + "/terms-of-service")
	}
}

func NIP96(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(config.FileServerConfig{
			APIURL:        cfg.Store.APIPath,
			DownloadURL:   cfg.Store.MediaPath,
			ContentTypes:  cfg.Store.AcceptedMimetypes,
			SupportedNIPS: []int{1, 4, 5, 78, 94, 96, 98},
			TOSURL:        cfg.RelayInformation.URL + "/terms-of-service",
		})
	}
}

func NostrJSON(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if name := c.Query("name"); name != "" {
			return c.JSON(fiber.Map{"names": fiber.Map{name: ""}})
		}
		return c.JSON(fiber.Map{
			"media": fiber.Map{
				"apiPath":           cfg.Store.APIPath,
				"mediaPath":         cfg.Store.MediaPath,
				"acceptedMimetypes": cfg.Store.AcceptedMimetypes,
				"contentPolicy": fiber.Map{
					"allowAdultContent":   cfg.Store.AllowAdultContent,
					"allowViolentContent": cfg.Store.AllowViolentContent,
				},
			},
			"names": cfg.Store.Names,
		})
	}
}

func RootUpgrade(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Get("Accept") == "application/nostr+json" {
			return c.JSON(cfg.RelayInformation)
		}
		if websocket.IsWebSocketUpgrade(c) {
			now := time.Now().UTC()
			userAgent := c.Get("User-Agent")
			c.Locals("allowed", true)
			c.Locals("ua", userAgent)
			c.Locals("wss", &dto.WsServer{
				Challenge:  util.GenChallenge(),
				Ctx:        c.Context(),
				ChanSender: make(chan any),
				ChanPing:   make(chan bool),
				StartTime:  now,
				LastSeen:   now,
				UserAgent:  userAgent,
			})
			return c.Next()
		}
		return c.Status(fiber.StatusUpgradeRequired).SendString("Please use a Nostr client to connect.")
	}
}
