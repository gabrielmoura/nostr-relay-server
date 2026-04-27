package net

import (
	"path/filepath"
	"strconv"

	"github.com/gabrielmoura/nostr-relay-server/config"
	httphandler "github.com/gabrielmoura/nostr-relay-server/infra/handler/http"
	httpblossom "github.com/gabrielmoura/nostr-relay-server/infra/handler/http/blossom"
	wshandler "github.com/gabrielmoura/nostr-relay-server/infra/handler/ws"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/net/middleware"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
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

	admin := app.Group("/admin", httphandler.AdminTokenMiddleware(r.Config))
	admin.Get("", httphandler.AdminIndex())
	admin.Get("/", httphandler.AdminIndex())
	admin.Get("/overview", httphandler.AdminOverview())
	admin.Get("/stream/status", httphandler.StreamStatus())
	admin.Get("/connections/active", httphandler.ActiveConnections())
	admin.Get("/connections/authed", httphandler.AuthedConnections())
	admin.Post("/connections/:wsid/disconnect", httphandler.DisconnectConnection())
	admin.Get("/users/logged", httphandler.LoggedUsers())
	admin.Get("/users/banned", httphandler.BannedUsers())
	admin.Get("/users/search", httphandler.SearchUsers())
	admin.Get("/users/:pubkey/profile", httphandler.UserProfile())
	admin.Get("/users/:pubkey/nip05", httphandler.UserNIP05())
	admin.Get("/users/:pubkey/ban", httphandler.BanStatus())
	admin.Post("/users/:pubkey/ban", httphandler.BanUser())
	admin.Delete("/users/:pubkey/ban", httphandler.UnbanUser())
	admin.Get("/nip05", httphandler.NIP05List())
	admin.Post("/nip05", httphandler.NIP05Upsert())
	admin.Delete("/nip05/:name", httphandler.NIP05Delete())
	admin.Get("/events/search", httphandler.SearchEvents())
	admin.Post("/events/import", httphandler.ImportEventsJSONL())
	admin.Get("/events/search/aggregates", httphandler.SearchEventsAggregates())
	admin.Get("/events/search/timeline", httphandler.SearchEventsTimeline())
	admin.Get("/events/reported", httphandler.ReportedEvents())
	admin.Post("/events/:id/fetch", httphandler.FetchEventFromRelays())
	admin.Get("/events/:id", httphandler.EventDetail())
	admin.Get("/events/:id/reports", httphandler.EventReports())
	admin.Get("/nip86/allowed-pubkeys", httphandler.NIP86AllowedPubKeys())
	admin.Post("/nip86/allowed-pubkeys/:pubkey", httphandler.NIP86CreateAllowedPubKey())
	admin.Delete("/nip86/allowed-pubkeys/:pubkey", httphandler.NIP86DeleteAllowedPubKey())
	admin.Get("/nip86/blocked-ips", httphandler.NIP86BlockedIPs())
	admin.Post("/nip86/blocked-ips/:ip", httphandler.NIP86CreateBlockedIP())
	admin.Delete("/nip86/blocked-ips/:ip", httphandler.NIP86DeleteBlockedIP())
	admin.Get("/nip86/banned-events", httphandler.NIP86BannedEvents())
	admin.Post("/nip86/banned-events/:id", httphandler.NIP86CreateBannedEvent())
	admin.Delete("/nip86/banned-events/:id", httphandler.NIP86DeleteBannedEvent())
	admin.Get("/nip86/relay-metadata", httphandler.NIP86RelayMetadata())
	admin.Post("/nip86/relay-metadata", httphandler.NIP86UpdateRelayMetadata())

	app.Get(httphandler.AdminUIBasePath(), httphandler.AdminUIIndex())
	app.Get(httphandler.AdminUIBasePath()+"/assets/*", httphandler.AdminUIAsset())
	app.Get(httphandler.AdminUIBasePath()+"/*", httphandler.AdminUISPAFallback())
}

// setupExternalRoutes configura as rotas públicas do Relay
func (r *RouterFactory) setupExternalRoutes(app *fiber.App) {
	// Middlewares Globais
	app.Use(cors.New(cors.Config{AllowOrigins: "*"}))
	app.Use(compress.New())

	// Arquivos Estáticos
	app.Static("/nostr.png", filepath.Join("nostr.png"))

	// Rotas Auxiliares
	app.Get("/terms-of-service", httphandler.TermsOfService(r.Config))

	// Well-Known Handlers (NIPs)
	r.setupWellKnownRoutes(app)

	// Blossom / Upload Handlers
	r.setupBlossomRoutes(app)

	// Rota Raiz: Lida com NIP-11 (Info) e Upgrade para WebSocket
	app.Use("/", httphandler.RootUpgrade(r.Config))
	app.Get("/", websocket.New(r.handleWebSocketConnection))
}

func (r *RouterFactory) setupWellKnownRoutes(app *fiber.App) {
	wellKnown := app.Group("/.well-known")

	// NIP-96
	wellKnown.Get("/nostr/nip96.json", httphandler.NIP96(r.Config))

	// NIP-05 / Nostr.json
	wellKnown.Get("/nostr.json", httphandler.NostrJSON(r.Config))
}

func (r *RouterFactory) setupBlossomRoutes(app *fiber.App) {
	// Middleware específico para bloquear se o Store estiver desabilitado
	// Assumindo que middleware.BlockIfStoreNotEnabled é um fiber.Handler
	mw := middleware.BlockIfStoreNotEnabled

	app.Post("/upload", mw, httpblossom.UploadHandler)
	app.Put("/upload", mw, httpblossom.UploadHandler)
	app.Get("/blob/:id", mw, httpblossom.BlobHandler)
	app.Head("/blob/:id", mw, httpblossom.BlobHandler)
	app.Get("/list/:id", mw, httpblossom.ListHandler)
}

// handleWebSocketConnection é o callback final do WebSocket
func (r *RouterFactory) handleWebSocketConnection(c *websocket.Conn) {
	wss, ok := c.Locals("wss").(*dto.WsServer)
	if !ok {
		// Logar erro caso o contexto não tenha sido setado corretamente
		return
	}
	wss.Conn = c
	wshandler.HandleConnection(wss)
}

// Router Função de entrada (Entrypoint) para manter compatibilidade com a chamada original
func Router() (internal, external *fiber.App) {
	factory := NewRouter()
	return factory.Build()
}
