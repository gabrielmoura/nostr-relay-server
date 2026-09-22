package ws

import (
	"sync"
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

type connectionLifecycle struct {
	done       chan struct{}
	writerDone chan struct{}
	stopOnce   sync.Once
}

func newConnectionLifecycle() *connectionLifecycle {
	return &connectionLifecycle{done: make(chan struct{}), writerDone: make(chan struct{})}
}

func (l *connectionLifecycle) stop() {
	l.stopOnce.Do(func() { close(l.done) })
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = pongWait / 2
	maxMessageSize = 1024 * 1024
)

func HandleConnection(wss *dto.WsServer) {
	metrics.NostrConnectionCounter.Inc()
	defer metrics.NostrConnectionCounter.Dec()

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

	if config.Cfg.Ws.NormalizedAuthMode() == "optional" {
		if err := auth.SendAuthChallengeNow(wss); err != nil {
			log.Logger.Warn("failed to send initial AUTH challenge", zap.Error(err))
			return
		}
	}

	lifecycle := newConnectionLifecycle()
	go writeLoop(wss, ticker, lifecycle)
	if config.Cfg.Ws.NormalizedAuthMode() != "optional" && config.Cfg.Ws.AuthEnabled() && wss.Challenge != "" {
		if !sendToWriter(wss.ChanSender, any([]any{"AUTH", wss.Challenge}), lifecycle.done) {
			log.Logger.Debug("discarded AUTH challenge because WebSocket writer stopped", zap.String("for", wss.RemoteIP))
		}
	}
	readLoop(wss, lifecycle)
	lifecycle.stop()
	<-lifecycle.writerDone
}

func writeLoop(wss *dto.WsServer, ticker *time.Ticker, lifecycle *connectionLifecycle) {
	defer close(lifecycle.writerDone)
	for {
		select {
		case msg := <-wss.ChanSender:
			metrics.NostrRelayWsMessagesSend.Inc()
			if err := wss.Conn.WriteJSON(msg); err != nil {
				log.Logger.Error("write error", zap.Error(err))
				stopWriterAndReader(wss, lifecycle)
				return
			}
		case ping := <-wss.ChanPing:
			if ping {
				if err := wss.Conn.WriteMessage(websocket.PongMessage, nil); err != nil {
					log.Logger.Debug("pong error", zap.Error(err))
					stopWriterAndReader(wss, lifecycle)
					return
				}
			}
		case <-ticker.C:
			if err := wss.Conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
				log.Logger.Debug("ping error", zap.Error(err))
				stopWriterAndReader(wss, lifecycle)
				return
			}
		case <-wss.Ctx.Done():
			return
		case <-lifecycle.done:
			return
		}
	}
}

func readLoop(wss *dto.WsServer, lifecycle *connectionLifecycle) {
	for {
		typ, message, err := wss.Conn.ReadMessage()
		limit := maxMessageSize
		if security.S != nil {
			limit = security.S.MaxMessageLength()
		}
		if len(message) > limit {
			log.Logger.Warn("message too large", zap.String("for", wss.RemoteIP), zap.Int("size", len(message)))
			metrics.NostrSecurityMessageRejectedTotal.WithLabelValues("max_message_length").Inc()
			if !sendToWriter(wss.ChanSender, any(nostr.NoticeEnvelope(security.Reason(security.PrefixRestricted, "message exceeds configured max_message_length"))), lifecycle.done) {
				log.Logger.Debug("discarded oversized-message NOTICE because WebSocket writer stopped", zap.String("for", wss.RemoteIP))
			}
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
			if !sendToWriter(wss.ChanPing, true, lifecycle.done) {
				return
			}
			continue
		}
		listener.Touch(wss)
		handleMessage(wss, message)
	}
}

func sendToWriter[T any](channel chan<- T, value T, done <-chan struct{}) bool {
	select {
	case channel <- value:
		return true
	case <-done:
		return false
	}
}

func stopWriterAndReader(wss *dto.WsServer, lifecycle *connectionLifecycle) {
	lifecycle.stop()
	if err := wss.Conn.Close(); err != nil {
		log.Logger.Debug("failed to close WebSocket after writer error", zap.Error(err))
	}
}
