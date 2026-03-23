package net

import (
	"path/filepath"
	"strconv"

	json "github.com/bytedance/sonic"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler"
	blossomStore "github.com/gabrielmoura/nostr-relay-server/infra/handler/store/blossom"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/net/middleware"
	"github.com/gabrielmoura/nostr-relay-server/infra/util"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"go.uber.org/zap"
)

// RouterFactory gerencia a criação das aplicações Fiber
type RouterFactory struct {
	Config *config.Config
	Logger *zap.Logger
}

// NewRouter cria uma nova factory.
// Idealmente, config e logger devem ser passados aqui, não acessados globalmente.
func NewRouter() *RouterFactory {
	return &RouterFactory{
		Config: config.Cfg,
		Logger: log.Logger,
	}
}

// Build inicializa e retorna as instâncias interna e externa
func (r *RouterFactory) Build() (internal, external *fiber.App) {
	// Configuração base do Fiber
	fiberConf := fiber.Config{
		JSONEncoder:           json.Marshal,
		JSONDecoder:           json.Unmarshal,
		AppName:               r.Config.RelayInformation.Name,
		DisableStartupMessage: true,
	}

	// 1. Setup Internal Server (Admin & Metrics)
	internal = fiber.New(fiberConf)
	r.setupLifecycleHooks(internal, "Internal Server")
	r.setupInternalRoutes(internal)

	// 2. Setup External Server (Public Relay)
	external = fiber.New(fiberConf)
	r.setupLifecycleHooks(external, "Relay Server")
	r.setupExternalRoutes(external)

	return internal, external
}

// setupLifecycleHooks configura logs de inicio e fim
func (r *RouterFactory) setupLifecycleHooks(app *fiber.App, serverName string) {
	app.Hooks().OnListen(func(data fiber.ListenData) error {
		// Verifica se é a porta principal ou interna para logar corretamente
		if data.Port == strconv.Itoa(r.Config.Port) {
			r.Logger.Info("Relay is listening", zap.String("address", data.Port))
		} else {
			r.Logger.Info(serverName+" is listening", zap.String("address", data.Port))
		}
		return nil
	})

	app.Hooks().OnShutdown(func() error {
		r.Logger.Info("Shutting down " + serverName)
		return nil
	})
}

// setupInternalRoutes configura rotas administrativas e métricas
func (r *RouterFactory) setupInternalRoutes(app *fiber.App) {
	p := fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler())

	app.Get("/metrics", func(c *fiber.Ctx) error {
		p(c.Context())
		return nil
	})

	app.Get("/admin", func(c *fiber.Ctx) error {
		return c.SendString("Admin Interface")
	})
}

// setupExternalRoutes configura as rotas públicas do Relay
func (r *RouterFactory) setupExternalRoutes(app *fiber.App) {
	// Middlewares Globais
	app.Use(cors.New(cors.Config{AllowOrigins: "*"}))
	app.Use(compress.New())

	// Arquivos Estáticos
	app.Static("/nostr.png", filepath.Join("nostr.png"))

	// Rotas Auxiliares
	app.Get("/terms-of-service", func(c *fiber.Ctx) error {
		return c.Redirect(r.Config.Store.APIPath + "/terms-of-service")
	})

	// Well-Known Handlers (NIPs)
	r.setupWellKnownRoutes(app)

	// Blossom / Upload Handlers
	r.setupBlossomRoutes(app)

	// Rota Raiz: Lida com NIP-11 (Info) e Upgrade para WebSocket
	app.Use("/", r.handleRootUpgrade)
	app.Get("/", websocket.New(r.handleWebSocketConnection))
}

func (r *RouterFactory) setupWellKnownRoutes(app *fiber.App) {
	wellKnown := app.Group("/.well-known")

	// NIP-96
	wellKnown.Get("/nostr/nip96.json", func(c *fiber.Ctx) error {
		return c.JSON(config.FileServerConfig{
			APIURL:        r.Config.Store.APIPath,
			DownloadURL:   r.Config.Store.MediaPath,
			ContentTypes:  r.Config.Store.AcceptedMimetypes,
			SupportedNIPS: []int{1, 4, 5, 78, 94, 96, 98},
			TOSURL:        r.Config.RelayInformation.URL + "/terms-of-service",
		})
	})

	// NIP-05 / Nostr.json
	wellKnown.Get("/nostr.json", func(c *fiber.Ctx) error {
		if name := c.Query("name"); name != "" {
			return c.JSON(fiber.Map{
				"names": fiber.Map{name: ""}, // Lógica placeholder do original
			})
		}
		return c.JSON(fiber.Map{
			"media": fiber.Map{
				"apiPath":           r.Config.Store.APIPath,
				"mediaPath":         r.Config.Store.MediaPath,
				"acceptedMimetypes": r.Config.Store.AcceptedMimetypes,
				"contentPolicy": fiber.Map{
					"allowAdultContent":   r.Config.Store.AllowAdultContent,
					"allowViolentContent": r.Config.Store.AllowViolentContent,
				},
			},
			"names": r.Config.Store.Names,
		})
	})
}

func (r *RouterFactory) setupBlossomRoutes(app *fiber.App) {
	// Middleware específico para bloquear se o Store estiver desabilitado
	// Assumindo que middleware.BlockIfStoreNotEnabled é um fiber.Handler
	mw := middleware.BlockIfStoreNotEnabled

	app.Post("/upload", mw, blossomStore.UploadHandler)
	app.Put("/upload", mw, blossomStore.UploadHandler)
	app.Get("/blob/:id", mw, blossomStore.BlobHandler)
	app.Head("/blob/:id", mw, blossomStore.BlobHandler)
	app.Get("/list/:id", mw, blossomStore.ListHandler)
}

// handleRootUpgrade atua como Middleware e Handler para a rota "/"
// Verifica NIP-11 (Accept Header) e prepara o contexto para WebSocket
func (r *RouterFactory) handleRootUpgrade(c *fiber.Ctx) error {
	// NIP-11: Relay Information Document
	if c.Get("Accept") == "application/nostr+json" {
		return c.JSON(r.Config.RelayInformation)
	}

	// WebSocket Upgrade Check
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
}

// handleWebSocketConnection é o callback final do WebSocket
func (r *RouterFactory) handleWebSocketConnection(c *websocket.Conn) {
	wss, ok := c.Locals("wss").(*dto.WsServer)
	if !ok {
		// Logar erro caso o contexto não tenha sido setado corretamente
		return
	}
	wss.Conn = c
	handler.HandleWS(wss)
}

// Router Função de entrada (Entrypoint) para manter compatibilidade com a chamada original
func Router() (internal, external *fiber.App) {
	factory := NewRouter()
	return factory.Build()
}
