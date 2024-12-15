package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/fasthttp/websocket"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/nbd-wtf/go-nostr"
	"github.com/rs/cors"
	"golang.org/x/time/rate"
	"log/slog"
	"net/http"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = pongWait / 2

	// Maximum message size allowed from peer.
	maxMessageSize = 512000
)

func serverHttpRelay(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	if r.Header.Get("Upgrade") == "websocket" {
		wsHandler(w, r, ctx)
	} else if r.Header.Get("Accept") == "application/nostr+json" {
		handleRelayInfo(w, r)
	} else {
		serverHttpRelay(w, r, ctx)
	}
}
func Init(ctx context.Context) *http.Server {
	limiter := rate.NewLimiter(config.Cfg.Ws.ReteLimit, config.Cfg.Ws.Burst)

	mux := http.NewServeMux()

	mux.HandleFunc("/relay", func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
		serverHttpRelay(w, r, ctx)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		js, _ := config.Cfg.RelayInformation.ToJson()
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	})

	// Adiciona suporte a CORS
	handler := cors.Default().Handler(mux)

	// Criar o servidor HTTP com suporte a desligamento gracioso
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Cfg.Port),
		Handler:      handler,
		WriteTimeout: 2 * time.Second,
		ReadTimeout:  2 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
	log.Logger.Info("Server started", slog.Int("port", config.Cfg.Port))

	return server

}

func wsHandler(w http.ResponseWriter, r *http.Request, ctx context.Context) {

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Logger.Error("Upgrade error:", err)
		return
	}

	wss := &dto.WsServer{
		Challenge:  genChallenge(),
		Conn:       conn,
		Request:    r,
		Ctx:        ctx,
		Response:   w,
		ChanSender: make(chan interface{}, 20),
	}
	wss.Lock()
	defer wss.Unlock()

	ctx, cancel := context.WithCancel(ctx)

	// reader
	go func() {
		defer func() {
			cancel()
			//ticker.Stop()
			//s.clientsMu.Lock()
			//wss.Lock()
			//if _, ok := s.clients[conn]; ok {
			conn.Close()
			//	delete(s.clients, conn)
			listener.RemoveListener(wss)
			//}
			//s.clientsMu.Unlock()
			//wss.Unlock()
			//s.Log.Infof("disconnected from %s", ip)
		}()
		conn.SetReadLimit(maxMessageSize)
		conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})

		if config.Cfg.Ws.Auth {
			//conn.WriteJSON(nostr.AuthEnvelope{Challenge: &wss.Challenge})
			wss.ChanSender <- nostr.AuthEnvelope{Challenge: &wss.Challenge}
		}

		for {
			typ, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(
					err,
					websocket.CloseGoingAway,        // 1001
					websocket.CloseNoStatusReceived, // 1005
					websocket.CloseAbnormalClosure,  // 1006
				) {
					log.Logger.WarnContext(
						ctx,
						"unexpected close error from ",
						slog.String("for", r.Header.Get("X-Forwarded-For")),
						slog.Any("error", err),
					)
				}
				break
			}
			if typ == websocket.PingMessage {
				conn.WriteMessage(websocket.PongMessage, nil)
				continue
			}
			go handleMessage(wss, message)
		}

	}()

	// writer
	go func() {
		for {
			select {
			case msg := <-wss.ChanSender:
				wss.Conn.WriteJSON(msg)
			case <-ctx.Done():
				return
			}
		}
	}()

}

func genChallenge() string {
	// NIP-42 challenge
	challenge := make([]byte, 8)
	rand.Read(challenge)

	// ponha no contexto

	return hex.EncodeToString(challenge)
}
