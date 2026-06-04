package net

import (
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gabrielmoura/nostr-relay-server/graph"
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
	graphqlHandler := fasthttpadaptor.NewFastHTTPHandler(graph.HTTPHandler())
	admin.Post("/graphql", func(c *fiber.Ctx) error {
		graphqlHandler(c.Context())
		return nil
	})
	schemaHandler := fasthttpadaptor.NewFastHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sdl, err := graph.SchemaSDL()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(sdl))
	}))
	admin.Get("/graphql/schema", func(c *fiber.Ctx) error {
		schemaHandler(c.Context())
		return nil
	})
	playgroundHandler := fasthttpadaptor.NewFastHTTPHandler(graph.PlaygroundHandler("/admin/graphql"))
	admin.Get("/graphql/playground", func(c *fiber.Ctx) error {
		playgroundHandler(c.Context())
		return nil
	})

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
	app.Put("/mirror", mw, httpblossom.MirrorHandler)
	app.Put("/media", mw, httpblossom.MediaHandler)
	app.Head("/media", mw, httpblossom.MediaHandler)
	app.Get("/blob/:id", mw, httpblossom.BlobHandler)
	app.Head("/blob/:id", mw, httpblossom.BlobHandler)
	app.Put("/report", mw, httpblossom.ReportHandler)
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
