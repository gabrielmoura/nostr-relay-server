// Package fiber provides an optional adapter for Fiber v2. The protocol logic
// remains in relaysync and TLS/HTTP2 stay with the host listener.
package fiber

import (
	"net/http"

	relaysync "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

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
