package ws

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	fastws "github.com/fasthttp/websocket"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func TestReadLoopDispatchesMultipleDMImportEventsWithoutBlockingAuthOrder(t *testing.T) {
	metrics.NostrRequestCounter.Reset()
	metrics.NostrRequestProcessingDuration.Reset()
	t.Cleanup(func() {
		metrics.NostrRequestCounter.Reset()
		metrics.NostrRequestProcessingDuration.Reset()
	})
	previousConfig := config.Cfg
	previousAuth := wsMessageHandlers[dto.TypeAUTH]
	previousEvent := wsMessageHandlers[dto.TypeEVENT]
	config.Cfg = &config.Config{DMImport: config.DMImportConfig{Enabled: true}}
	t.Cleanup(func() {
		config.Cfg = previousConfig
		wsMessageHandlers[dto.TypeAUTH] = previousAuth
		wsMessageHandlers[dto.TypeEVENT] = previousEvent
	})

	authed := make(chan struct{}, 2)
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	wsMessageHandlers[dto.TypeAUTH] = func(server *dto.WsServer, _ dto.Data) string {
		server.SetAuth("owner", server.Ctx)
		authed <- struct{}{}
		return ""
	}
	wsMessageHandlers[dto.TypeEVENT] = func(_ *dto.WsServer, _ dto.Data) string {
		started <- struct{}{}
		<-release
		return ""
	}

	app := fiber.New()
	app.Get("/ws", websocket.New(func(conn *websocket.Conn) {
		HandleConnection(&dto.WsServer{
			Ctx:        context.Background(),
			Conn:       conn,
			RemoteIP:   conn.RemoteAddr().String(),
			ChanSender: make(chan any),
			ChanPing:   make(chan bool),
		})
	}))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
		_ = app.Shutdown()
	})
	go func() { _ = app.Listener(listener) }()

	conn, _, err := fastws.DefaultDialer.Dial("ws://"+listener.Addr().String()+"/ws", http.Header{})
	if err != nil {
		t.Fatalf("dial WebSocket: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteMessage(fastws.TextMessage, []byte(`["AUTH",{}]`)); err != nil {
		t.Fatalf("write AUTH: %v", err)
	}
	select {
	case <-authed:
	case <-time.After(time.Second):
		t.Fatal("AUTH was not processed synchronously before EVENT dispatch")
	}

	message := []byte(`["EVENT",{"id":"event","pubkey":"owner","created_at":1,"kind":4,"tags":[],"content":"cipher","sig":"sig"}]`)
	if err := conn.WriteMessage(fastws.TextMessage, message); err != nil {
		t.Fatalf("write first EVENT: %v", err)
	}
	if err := conn.WriteMessage(fastws.TextMessage, message); err != nil {
		t.Fatalf("write second EVENT: %v", err)
	}
	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("DM import EVENT did not reach a concurrent dispatcher worker")
		}
	}
	// ReAUTH mutates the live connection while the import workers retain their
	// immutable snapshots. This is exercised under -race in CI/local checks.
	if err := conn.WriteMessage(fastws.TextMessage, []byte(`["AUTH",{}]`)); err != nil {
		t.Fatalf("write reAUTH: %v", err)
	}
	select {
	case <-authed:
	case <-time.After(time.Second):
		t.Fatal("reAUTH was not processed")
	}
	close(release)
}
