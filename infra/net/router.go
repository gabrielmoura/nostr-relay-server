package net

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/store"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/net/middleware"
	"github.com/gabrielmoura/nostr-relay-server/infra/util"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/goccy/go-json"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"go.uber.org/zap"
	"path/filepath"
	"strconv"
)

func hooks(a *fiber.App) func() {
	return func() {
		a.Hooks().OnListen(func(data fiber.ListenData) error {
			if data.Port == strconv.Itoa(config.Cfg.Port) {
				log.Logger.Info("Relay is listening on", zap.String("address", data.Port))
			} else {
				// Internal Server
				log.Logger.Info("Internal Server is listening on", zap.String("address", data.Port))
			}
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
	external.Use(cors.New(cors.Config{
		AllowOrigins: "*",
	}))

	external.Static("/nostr.png", filepath.Join("nostr.png"))

	external.Get("/terms-of-service", func(c *fiber.Ctx) error {
		return c.Redirect(config.Cfg.Store.APIPath + "/terms-of-service")
	})

	wellKnown := external.Group("/.well-known")
	wellKnown.Get("/nostr/nip96.json", func(c *fiber.Ctx) error {
		return c.JSON(config.FileServerConfig{
			APIURL:        config.Cfg.Store.APIPath,
			DownloadURL:   config.Cfg.Store.MediaPath,
			ContentTypes:  config.Cfg.Store.AcceptedMimetypes,
			SupportedNIPS: []int{1, 4, 5, 78, 94, 96, 98},
			TOSURL:        config.Cfg.RelayInformation.URL + "/terms-of-service",
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

	external.Post("/upload", store.UploadHandler).Use(middleware.BlockIfStoreNotEnabled)
	external.Put("/upload", store.UploadHandler).Use(middleware.BlockIfStoreNotEnabled)
	external.Get("/blob/:id", store.BlobHandler).Use(middleware.BlockIfStoreNotEnabled)
	external.Head("/blob/:id", store.BlobHandler).Use(middleware.BlockIfStoreNotEnabled)
	external.Get("/list/:id", store.ListHandler).Use(middleware.BlockIfStoreNotEnabled)

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
