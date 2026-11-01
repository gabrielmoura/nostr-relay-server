package ws

import (
	"context"
	"sync"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/auth"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/event"
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

	dmImportDispatchWorkers   = 4
	dmImportDispatchQueueSize = 64
)

type dmImportDispatcher struct {
	queue  chan dmImportJob
	stopCh chan struct{}
	wg     sync.WaitGroup
}

type dmImportJob struct {
	ws      *dto.WsServer
	message []byte
}

func newDMImportDispatcher(handle func(*dto.WsServer, []byte)) *dmImportDispatcher {
	d := &dmImportDispatcher{
		queue:  make(chan dmImportJob, dmImportDispatchQueueSize),
		stopCh: make(chan struct{}),
	}
	for range dmImportDispatchWorkers {
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			for {
				select {
				case job := <-d.queue:
					handle(job.ws, job.message)
				case <-d.stopCh:
					return
				}
			}
		}()
	}
	return d
}

func (d *dmImportDispatcher) dispatch(ws *dto.WsServer, message []byte) bool {
	select {
	case d.queue <- dmImportJob{ws: ws, message: message}:
		return true
	case <-d.stopCh:
		return false
	default:
		return false
	}
}

func (d *dmImportDispatcher) stop() {
	close(d.stopCh)
	d.wg.Wait()
}

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

	connectionCtx, cancelConnection := context.WithCancel(wss.Ctx)
	defer cancelConnection()
	wss.Ctx = connectionCtx
	lifecycle := newConnectionLifecycle()
	wss.Done = lifecycle.done
	go writeLoop(wss, ticker, lifecycle)
	needsImportAuthChallenge := config.Cfg.DMImport.Enabled
	needsGlobalAuthChallenge := config.Cfg.Ws.NormalizedAuthMode() != "optional" && config.Cfg.Ws.AuthEnabled()
	if (needsImportAuthChallenge || needsGlobalAuthChallenge) && wss.Challenge != "" {
		if !sendToWriter(wss.ChanSender, any([]any{"AUTH", wss.Challenge}), lifecycle.done) {
			log.Logger.Debug("discarded AUTH challenge because WebSocket writer stopped", zap.String("for", wss.RemoteIP))
		}
	}
	readLoop(wss, lifecycle, cancelConnection)
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
		case <-lifecycle.done:
			return
		}
	}
}

func readLoop(wss *dto.WsServer, lifecycle *connectionLifecycle, cancelConnection context.CancelFunc) {
	dispatcher := newDMImportDispatcher(func(snapshot *dto.WsServer, message []byte) { handleMessage(snapshot, message) })
	defer func() {
		cancelConnection()
		dispatcher.stop()
	}()
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
		authed, _ := wss.AuthSnapshot()
		if config.Cfg.DMImport.Enabled && authed != "" && isDMImportMessage(message) {
			if !dispatcher.dispatch(wss.ImportSnapshot(), message) {
				if !sendToWriter(wss.ChanSender, any(nostr.NoticeEnvelope("rate-limited: DM import dispatch queue is full")), lifecycle.done) {
					return
				}
			}
			continue
		}
		handleMessage(wss, message)
	}
}

func isDMImportMessage(message []byte) bool {
	typ, data, err := decodeMessage(message)
	return err == nil && typ == dto.TypeEVENT && event.IsDMImportData(data)
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
