package sync

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"go.uber.org/zap"
)

func TestCancellationInterruptsSilentWebsocket(t *testing.T) {
	previous := log.Logger
	log.Logger = zap.NewNop()
	t.Cleanup(func() { log.Logger = previous })
	ready := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetPongHandler(func(string) error { close(ready); return nil })
		if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(time.Second)); err != nil {
			return
		}
		_, _, _ = conn.ReadMessage()
	}))
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := closeOnCancellation(ctx, conn)
	defer stop()
	session := &SyncSession{Context: ctx, Conn: conn}
	done := make(chan error, 1)
	go func() { done <- session.listenLoop() }()
	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("websocket reader did not respond to ping")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("listenLoop error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancel did not stop websocket read")
	}
}
