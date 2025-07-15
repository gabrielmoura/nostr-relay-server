package net

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/store"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/util"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/goccy/go-json"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"go.uber.org/zap"
	"path/filepath"
)

func hooks(a *fiber.App) func() {
	return func() {
		a.Hooks().OnListen(func(data fiber.ListenData) error {
			log.Logger.Info("Server is listening on", zap.String("address", data.Port))
			return nil
		})
		a.Hooks().OnShutdown(func() error {
			log.Logger.Info("Shutting down server")
			return nil
		})
	}
}

func Router() (internal, external *fiber.App) {
	conf := fiber.Config{
		JSONEncoder:           json.Marshal,
		JSONDecoder:           json.Unmarshal,
		AppName:               config.Cfg.RelayInformation.Name,
		DisableStartupMessage: true,
	}

	internal = fiber.New(conf)

	p := fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler())
	internal.Get("/metrics", func(c *fiber.Ctx) error {
		p(c.Context())
		return nil
	})
	internal.Get("/admin", func(c *fiber.Ctx) error {
		return c.SendString("Admin Interface")
	})

	external = fiber.New(conf)

	external.Static("/nostr.png", filepath.Join("nostr.png"))

	wellKnown := external.Group("/.well-known")
	wellKnown.Get("/nostr/nip96.json", func(c *fiber.Ctx) error {
		return c.JSON(config.FileServerConfig{
			APIURL:      config.Cfg.Store.APIPath,
			DownloadURL: config.Cfg.Store.MediaPath,
		})
	})
	wellKnown.Get("/nostr.json", func(c *fiber.Ctx) error {
		//?name=
		if name := c.Query("name"); name != "" {
			return c.JSON(map[string]any{
				"names": map[string]string{
					name: "",
				},
			})
		}
		return c.JSON(map[string]any{
			"media": map[string]any{
				"apiPath":           config.Cfg.Store.APIPath,
				"mediaPath":         config.Cfg.Store.MediaPath,
				"acceptedMimetypes": config.Cfg.Store.AcceptedMimetypes,
				"contentPolicy": map[string]any{
					"allowAdultContent":   config.Cfg.Store.AllowAdultContent,
					"allowViolentContent": config.Cfg.Store.AllowViolentContent,
				},
			},
			"names": config.Cfg.Store.Names,
		})

	})

	external.Post("/upload", store.UploadHandler)
	external.Put("/upload", store.UploadHandler)
	external.Get("/blob/:id", store.BlobHandler)
	external.Head("/blob/:id", store.BlobHandler)

	external.Use("/", func(c *fiber.Ctx) error {
		// se estritamente /
		//if c.Path() == "/" {
		if c.Get("Accept") == "application/nostr+json" {
			return c.JSON(config.Cfg.RelayInformation)
		}

		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			c.Locals("ua", c.Get("User-Agent"))
			c.Locals("wss", &dto.WsServer{
				Challenge:  util.GenChallenge(),
				Ctx:        c.Context(),
				ChanSender: make(chan interface{}),
				ChanPing:   make(chan bool),
			})
			return c.Next()
		}
		return c.Status(fiber.StatusUpgradeRequired).SendString("Please use a Nostr client to connect.")
		//}
		//return c.Next()

	}, compress.New())
	external.Get("/", websocket.New(func(c *websocket.Conn) {
		wss := c.Locals("wss").(*dto.WsServer)
		wss.Conn = c
		handler.HandleWS(wss)
	}))

	hooks(internal)()
	hooks(external)()
	return
}
