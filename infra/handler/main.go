package handler

import (
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/gofiber/contrib/websocket"
	"go.uber.org/zap"
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

//	func serverHttpRelay(w http.ResponseWriter, r *http.Request, ctx context.Context) {
//		if r.Header.Get("Upgrade") == "websocket" {
//			wsHandler(w, r, ctx)
//		} else {
//			if strings.Contains(r.Header.Get("Accept"), "application/nostr+json") {
//				handleRelayInfo(w, r)
//			} else {
//				http.Error(w, "Not a Nostr client", http.StatusNotAcceptable)
//			}
//		}
//
// }
//
//	func Init(ctx context.Context) *http.Server {
//		limiter := rate.NewLimiter(config.Cfg.Ws.ReteLimit, config.Cfg.Ws.Burst)
//
//		mux := http.NewServeMux()
//
//		mux.HandleFunc("/relay", func(w http.ResponseWriter, r *http.Request) {
//			if !limiter.Allow() {
//				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
//				return
//			}
//			serverHttpRelay(w, r, ctx)
//		})
//
//		mux.HandleFunc("/nostr.png", func(w http.ResponseWriter, r *http.Request) {
//			http.ServeFile(w, r, filepath.Join("nostr.png"))
//		})
//
//		mux.HandleFunc("/.well-known/nostr/nip96.json", store2.HandleWellKnownNip96)
//		mux.HandleFunc("/.well-known/nostr.json", store2.HandleWellKnown)
//		mux.HandleFunc("/blob/", store2.BlobHandler)
//		mux.HandleFunc("/upload", store2.UploadHandler)
//
//		mux.Handle("/metrics", promhttp.Handler())
//
//		mux.HandleFunc("/", handleRelayInfo)
//
//		// Adiciona suporte a CORS
//		handler := cors.Default().Handler(mux)
//
//		// Criar o servidor HTTP com suporte a desligamento gracioso
//		server := &http.Server{
//			Addr:         fmt.Sprintf(":%d", config.Cfg.Port),
//			Handler:      handler,
//			WriteTimeout: 2 * time.Second,
//			ReadTimeout:  2 * time.Second,
//			IdleTimeout:  30 * time.Second,
//		}
//		log.Logger.Info("Server started", zap.Int("port", config.Cfg.Port))
//
//		return server
//
// }
func HandleWS(wss *dto.WsServer) {
	ticker := time.NewTicker(pingPeriod)
	go func() {
		for {
			select {
			case msg := <-wss.ChanSender:
				metrics.NostrRelayWsMessagesSend.Inc()
				err := wss.Conn.WriteJSON(msg)
				if err != nil {
					log.Logger.Error("write error", zap.Error(err))
					return
				}
			case ping := <-wss.ChanPing:
				if ping {
					if err := wss.Conn.WriteMessage(websocket.PongMessage, nil); err != nil {
						log.Logger.Warn("pong error", zap.Error(err))
						return
					}
				}
			case <-ticker.C:
				if err := wss.Conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
					log.Logger.Warn("ping error", zap.Error(err))
					return
				}
				log.Logger.Debug("pinging for", zap.String("ip", wss.Conn.IP()))
			case <-wss.Ctx.Done():
				return
			}
		}
	}()
	for {
		typ, message, err := wss.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,        // 1001
				websocket.CloseNoStatusReceived, // 1005
				websocket.CloseAbnormalClosure,  // 1006
			) {
				log.Logger.Warn(
					"unexpected close error from ",
					zap.String("for", wss.Conn.IP()),
					zap.Error(err),
				)
			}
			break
		}
		if typ == websocket.PingMessage {
			wss.ChanPing <- true
			continue
		}
		go handleMessage(wss, message)
	}

}
