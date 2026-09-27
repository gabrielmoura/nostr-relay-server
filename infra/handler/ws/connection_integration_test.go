//go:build integration

package ws

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	fastws "github.com/fasthttp/websocket"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func TestOversizedMessageOverRealWebSocketUpgrade(t *testing.T) {
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

	url := "ws://" + listener.Addr().String() + "/ws"
	conn, _, err := fastws.DefaultDialer.Dial(url, http.Header{})
	if err != nil {
		t.Fatalf("dial WebSocket: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteMessage(fastws.TextMessage, make([]byte, maxMessageSize+1)); err != nil {
		t.Fatalf("write oversized message: %v", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read NOTICE: %v", err)
	}
	if !strings.Contains(string(message), "NOTICE") || !strings.Contains(string(message), "max_message_length") {
		t.Fatalf("NOTICE = %q, want oversized-message notice", message)
	}
}
