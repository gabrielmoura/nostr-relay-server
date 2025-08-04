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

func HandleWS(wss *dto.WsServer) {
	ticker := time.NewTicker(pingPeriod)
	metrics.NostrUserAgentCounter.WithLabelValues(wss.Conn.Locals("ua").(string)).Inc()
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
						log.Logger.Debug("pong error", zap.Error(err))
						return
					}
				}
			case <-ticker.C:
				if err := wss.Conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
					log.Logger.Debug("ping error", zap.Error(err))
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
