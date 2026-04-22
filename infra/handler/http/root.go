package http

import (
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/nip05"
	"github.com/gabrielmoura/nostr-relay-server/infra/util"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func TermsOfService(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if strings.TrimSpace(cfg.RelayInformation.TermsOfService) != "" {
			return c.Redirect(cfg.RelayInformation.TermsOfService)
		}
		return c.Redirect(cfg.Store.APIPath + "/terms-of-service")
	}
}

func NIP96(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tosURL := cfg.RelayInformation.URL + "/terms-of-service"
		if strings.TrimSpace(cfg.RelayInformation.TermsOfService) != "" {
			tosURL = cfg.RelayInformation.TermsOfService
		}
		return c.JSON(config.FileServerConfig{
			APIURL:        cfg.Store.APIPath,
			DownloadURL:   cfg.Store.MediaPath,
			ContentTypes:  cfg.Store.AcceptedMimetypes,
			SupportedNIPS: []int{1, 4, 5, 78, 94, 96, 98},
			TOSURL:        tosURL,
		})
	}
}

func NostrJSON(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		svc := nip05.NewService(db.DbQueries)
		doc, err := svc.BuildDocument(c.UserContext(), c.Query("name"))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		if c.Query("name") != "" {
			return c.JSON(doc)
		}

		response := fiber.Map{
			"media": fiber.Map{
				"apiPath":           cfg.Store.APIPath,
				"mediaPath":         cfg.Store.MediaPath,
				"acceptedMimetypes": cfg.Store.AcceptedMimetypes,
				"contentPolicy": fiber.Map{
					"allowAdultContent":   cfg.Store.AllowAdultContent,
					"allowViolentContent": cfg.Store.AllowViolentContent,
				},
			},
			"names": doc.Names,
		}

		if len(doc.Relays) > 0 {
			response["relays"] = doc.Relays
		}

		return c.JSON(response)
	}
}

func RootUpgrade(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if strings.Contains(c.Get("Accept"), "application/nostr+json") {
			return c.JSON(cfg.RelayInformation.PublicNIP11())
		}
		if websocket.IsWebSocketUpgrade(c) {
			now := time.Now().UTC()
			userAgent := c.Get("User-Agent")
			remoteIP := c.IP()
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
				RemoteIP:   remoteIP,
			})
			return c.Next()
		}
		return c.Status(fiber.StatusUpgradeRequired).SendString("Please use a Nostr client to connect.")
	}
}
