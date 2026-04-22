package ws

import (
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/auth"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/gabrielmoura/nostr-relay-server/internal/security"
	"github.com/gofiber/contrib/websocket"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = pongWait / 2
	maxMessageSize = 1024 * 1024
)

func HandleConnection(wss *dto.WsServer) {
	if security.S != nil {
		allowed, reason := security.S.AcquireConnection(wss.RemoteIP)
		if !allowed {
			_ = wss.Conn.WriteJSON(nostr.NoticeEnvelope(reason))
			_ = wss.Conn.Close()
			return
		}
		defer security.S.ReleaseConnection(wss.RemoteIP)
	}

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	defer listener.RemoveListener(wss)
	listener.Touch(wss)
	metrics.NostrUserAgentCounter.WithLabelValues(wss.Conn.Locals("ua").(string)).Inc()

	if config.Cfg.Ws.NormalizedAuthMode() == "optional" {
		if err := auth.SendAuthChallengeNow(wss); err != nil {
			log.Logger.Warn("failed to send initial AUTH challenge", zap.Error(err))
			return
		}
	}

	go writeLoop(wss, ticker)
	if config.Cfg.Ws.NormalizedAuthMode() != "optional" {
		auth.SendAuthChallenge(wss)
	}
	readLoop(wss)
}

func writeLoop(wss *dto.WsServer, ticker *time.Ticker) {
	for {
		select {
		case msg := <-wss.ChanSender:
			metrics.NostrRelayWsMessagesSend.Inc()
			if err := wss.Conn.WriteJSON(msg); err != nil {
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
		case <-wss.Ctx.Done():
			return
		}
	}
}

func readLoop(wss *dto.WsServer) {
	for {
		typ, message, err := wss.Conn.ReadMessage()
		limit := maxMessageSize
		if security.S != nil {
			limit = security.S.MaxMessageLength()
		}
		if len(message) > limit {
			log.Logger.Warn("message too large", zap.String("for", wss.RemoteIP), zap.Int("size", len(message)))
			metrics.NostrSecurityMessageRejectedTotal.WithLabelValues("max_message_length").Inc()
			wss.ChanSender <- nostr.NoticeEnvelope(security.Reason(security.PrefixRestricted, "message exceeds configured max_message_length"))
			return
		}
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseNoStatusReceived,
				websocket.CloseAbnormalClosure,
			) {
				log.Logger.Warn("unexpected close error from", zap.String("for", wss.Conn.IP()), zap.Error(err))
			}
			return
		}
		if typ == websocket.PingMessage {
			listener.Touch(wss)
			wss.ChanPing <- true
			continue
		}
		listener.Touch(wss)
		handleMessage(wss, message)
	}
}
