// Package fiber provides an optional adapter for Fiber v2. The protocol logic
// remains in relaysync and TLS/HTTP2 stay with the host listener.
package fiber

import (
	"net/http"

	relaysync "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

// Services groups the optional RelaySync HTTP services registered on a Fiber
// listener. Principal resolution remains an application concern.
type Services struct {
	Sessions  *relaysync.SessionService
	Manifests *relaysync.ManifestService
	Transfer  *relaysync.TransferService
	Reconcile *relaysync.ReconcileService
	Resolve   relaysync.PrincipalResolver
}

func Register(app *fiber.App, adapter *relaysync.HTTPAdapter) error {
	if app == nil || adapter == nil {
		return relaysync.ErrNotFound
	}
	mux := http.NewServeMux()
	adapter.Register(mux)
	handler := fasthttpadaptor.NewFastHTTPHandler(mux)
	app.Use(relaysync.CapabilitiesPath, func(ctx *fiber.Ctx) error { handler(ctx.Context()); return nil })
	return nil
}

// RegisterServices registers capabilities plus the configured session,
// manifest, transfer and reconcile endpoints on the existing Fiber listener.
// It does not create a server, TLS configuration or WebSocket upgrade route.
func RegisterServices(app *fiber.App, adapter *relaysync.HTTPAdapter, services Services) error {
	if app == nil || adapter == nil || services.Resolve == nil {
		return relaysync.ErrUnauthorized
	}
	mux := http.NewServeMux()
	adapter.Register(mux)
	if services.Sessions != nil {
		if err := adapter.RegisterSessionService(mux, services.Sessions, services.Resolve); err != nil {
			return err
		}
	}
	if services.Manifests != nil || services.Transfer != nil {
		if err := adapter.RegisterTransferServices(mux, services.Sessions, services.Manifests, services.Transfer, services.Resolve); err != nil {
			return err
		}
	}
	if services.Reconcile != nil {
		if err := adapter.RegisterReconcileService(mux, services.Reconcile, services.Resolve); err != nil {
			return err
		}
	}
	handler := fasthttpadaptor.NewFastHTTPHandler(mux)
	app.Use("/sync/v2", func(ctx *fiber.Ctx) error { handler(ctx.Context()); return nil })
	return nil
}
