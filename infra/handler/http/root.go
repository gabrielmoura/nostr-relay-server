package http

import (
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/net/privacy"
	"github.com/gabrielmoura/nostr-relay-server/infra/nip05"
	"github.com/gabrielmoura/nostr-relay-server/infra/ratelimit"
	"github.com/gabrielmoura/nostr-relay-server/infra/util"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	assets "github.com/gabrielmoura/nostr-relay-server/internal/embed"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/gabrielmoura/nostr-relay-server/internal/nip86"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func NostrIcon() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Type("png")
		c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
		return c.Send(assets.NostrPNG)
	}
}

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

// NIP11WithPrivacy returns the relay's NIP-11 document, augmented with the
// currently active privacy addresses (Tor onion, I2P b32, Yggdrasil) when the
// privacy subsystem is enabled.
func NIP11WithPrivacy(cfg *config.Config) any {
	doc := cfg.RelayInformation.PublicNIP11()
	addrs := privacy.GetActiveAddresses()
	raw, err := json.Marshal(doc)
	if err != nil {
		return doc
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return doc
	}
	if len(addrs) > 0 {
		m["privacy_addresses"] = addrs
	}
	if cfg.NIP29.Enabled {
		m["nip29"] = map[string]any{"subgroups": true}
		supported, _ := m["supported_nips"].([]any)
		hasNIP29 := false
		for _, nip := range supported {
			if isNIP29Number(nip) {
				hasNIP29 = true
				break
			}
		}
		if !hasNIP29 {
			m["supported_nips"] = append(supported, 29)
		}
	}
	if cfg.NIP70.Enabled {
		supported, _ := m["supported_nips"].([]any)
		hasNIP70 := false
		for _, nip := range supported {
			if isNIPNumber(nip, 70) {
				hasNIP70 = true
				break
			}
		}
		if !hasNIP70 {
			m["supported_nips"] = append(supported, 70)
		}
	}
	if cfg.API.NIPFE.Enabled {
		m["nip_fe"] = true
	}
	if cfg.NIP42Enabled() {
		trustedAuthURLs := cfg.RelayInformation.TrustedAuthRelayURLs()
		if len(trustedAuthURLs) > 0 {
			m["auth_relay_url"] = trustedAuthURLs[0]
			m["auth_relay_urls"] = trustedAuthURLs
		}
	}
	if cfg.RelaySync.Enabled && cfg.RelaySync.HTTPEnabled && cfg.RelaySync.SnapshotEnabled {
		extensions, _ := m["supported_nip_extensions"].([]any)
		found := false
		for _, extension := range extensions {
			if extension == "relay-sync-v2" {
				found = true
				break
			}
		}
		if !found {
			m["supported_nip_extensions"] = append(extensions, "relay-sync-v2")
		}
		m["relay_sync_v2"] = map[string]any{"http": true, "live": cfg.RelaySync.LiveEnabled}
	}
	m["public_endpoints"] = publicEndpoints(cfg)
	return m
}

type publicEndpoint struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

// publicEndpoints is a truthful, public-listener-only endpoint list consumed
// by the landing page. Each optional endpoint matches setupExternalRoutes.
func publicEndpoints(cfg *config.Config) []publicEndpoint {
	endpoints := []publicEndpoint{
		{Method: fiber.MethodGet, Path: "/", Description: "NIP-11 relay information"},
		{Method: fiber.MethodGet, Path: "/", Description: "Nostr WebSocket relay (Upgrade)"},
		{Method: fiber.MethodGet, Path: "/.well-known/nostr.json", Description: "NIP-05 identifiers"},
		{Method: fiber.MethodGet, Path: "/.well-known/nostr/nip96.json", Description: "NIP-96 file-server information"},
		{Method: fiber.MethodGet, Path: "/terms-of-service", Description: "Terms of service"},
	}
	if cfg.Store.Enabled {
		endpoints = append(endpoints,
			publicEndpoint{Method: fiber.MethodPost, Path: "/upload", Description: "Blossom file upload"},
			publicEndpoint{Method: fiber.MethodPut, Path: "/mirror", Description: "Blossom file mirror"},
			publicEndpoint{Method: fiber.MethodPut, Path: "/media", Description: "Blossom media upload"},
			publicEndpoint{Method: fiber.MethodGet, Path: "/blob/:id", Description: "Blossom blob download"},
			publicEndpoint{Method: fiber.MethodPut, Path: "/report", Description: "Blossom moderation report"},
			publicEndpoint{Method: fiber.MethodGet, Path: "/list/:id", Description: "Blossom media list"},
		)
	}
	if cfg.API.Enabled {
		endpoints = append(endpoints, publicEndpoint{Method: "API", Path: "/api/v1", Description: "Public relay API"})
	}
	if cfg.API.NIPFE.Enabled {
		endpoints = append(endpoints, publicEndpoint{Method: fiber.MethodPost, Path: "/", Description: "NIP-FE finite relay transport"})
	}
	return endpoints
}

func isNIP29Number(value any) bool {
	return isNIPNumber(value, 29)
}

func isNIPNumber(value any, nip int) bool {
	switch value := value.(type) {
	case int:
		return value == nip
	case float64:
		return value == float64(nip)
	default:
		return false
	}
}

func RootUpgrade(cfg *config.Config, limiters ...*ratelimit.PerIP) fiber.Handler {
	return rootUpgrade(cfg, nil, true, limiters...)
}

// RootUpgradeWithPublicUI keeps the relay protocol multiplexed at / while
// serving the isolated public UI only to ordinary HTML GET requests.
func RootUpgradeWithPublicUI(cfg *config.Config, landing fiber.Handler, limiters ...*ratelimit.PerIP) fiber.Handler {
	return rootUpgrade(cfg, landing, false, limiters...)
}

func rootUpgrade(cfg *config.Config, landing fiber.Handler, allowNIP86 bool, limiters ...*ratelimit.PerIP) fiber.Handler {
	var negentropyLimiter *ratelimit.PerIP
	var dmImportLimiter *ratelimit.PerIP
	var negentropyAllow func(string) bool
	var dmImportAllow func(string) bool
	if len(limiters) > 0 {
		negentropyLimiter = limiters[0]
	}
	if len(limiters) > 1 {
		dmImportLimiter = limiters[1]
	}
	if negentropyLimiter != nil {
		negentropyAllow = negentropyLimiter.Allow
	}
	if dmImportLimiter != nil {
		dmImportAllow = dmImportLimiter.Allow
	}
	return func(c *fiber.Ctx) error {
		if allowNIP86 {
			if handled, err := handleNIP86JSONRPC(c, cfg); handled {
				return err
			}
		}
		if strings.Contains(c.Get("Accept"), "application/nostr+json") {
			return serveNIP11(c, cfg)
		}
		if websocket.IsWebSocketUpgrade(c) {
			if nip86.S != nil && nip86.S.Enabled() {
				reason, blocked, err := nip86.S.IsIPBlocked(c.UserContext(), c.IP())
				if err != nil {
					return c.Status(fiber.StatusServiceUnavailable).SendString("ip moderation lookup failed")
				}
				if blocked {
					return c.Status(fiber.StatusForbidden).SendString(reason)
				}
			}
			now := time.Now().UTC()
			userAgent := c.Get("User-Agent")
			remoteIP := c.IP()
			c.Locals("allowed", true)
			c.Locals("ua", userAgent)
			c.Locals("wss", &dto.WsServer{
				Challenge:       util.GenChallenge(),
				Ctx:             c.Context(),
				ChanSender:      make(chan any),
				ChanPing:        make(chan bool),
				StartTime:       now,
				LastSeen:        now,
				UserAgent:       userAgent,
				RemoteIP:        remoteIP,
				NegentropyAllow: negentropyAllow,
				DMImportAllow:   dmImportAllow,
			})
			return c.Next()
		}
		if landing != nil && c.Method() == fiber.MethodGet && strings.Contains(c.Get(fiber.HeaderAccept), fiber.MIMETextHTML) {
			return landing(c)
		}
		return c.Status(fiber.StatusUpgradeRequired).SendString("Please use a Nostr client to connect.")
	}
}

func serveNIP11(c *fiber.Ctx, cfg *config.Config) error {
	payload, err := json.Marshal(NIP11WithPrivacy(cfg))
	if err != nil {
		return err
	}

	c.Set(fiber.HeaderContentType, "application/nostr+json")
	c.Set(fiber.HeaderAccessControlAllowOrigin, "*")
	c.Set(fiber.HeaderAccessControlAllowHeaders, "Accept, Content-Type")
	c.Set(fiber.HeaderAccessControlAllowMethods, fiber.MethodGet+", "+fiber.MethodOptions)
	return c.Send(payload)
}
