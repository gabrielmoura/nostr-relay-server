package ws

import (
	"fmt"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/auth"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/count"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/event"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/req"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropy"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

type messageHandler func(*dto.WsServer, dto.Data) string

var wsMessageHandlers = map[string]messageHandler{
	dto.TypeEVENT:    event.DoEVENT,
	dto.TypeREQ:      req.DoREQ,
	dto.TypeAUTH:     auth.DoAUTH,
	dto.TypeCOUNT:    count.DoCOUNT,
	dto.TypeCLOSE:    handleClose,
	dto.TypeNegOpen:  handleNegOpen,
	dto.TypeNegMsg:   handleNegMsg,
	dto.TypeNegHave:  handleNegHave,
	dto.TypeNegNeed:  handleNegNeed,
	dto.TypeNegErr:   handleNegErr,
	dto.TypeNegClose: handleNegClose,
}

func handleMessage(ws *dto.WsServer, message []byte) {
	defer func() {
		if r := recover(); r != nil {
			log.Logger.Error("websocket message handler panic", zap.Any("panic", r), zap.ByteString("message", message))
			ws.ChanSender <- nostr.NoticeEnvelope("internal error")
		}
	}()

	typ, data, err := decodeMessage(message)
	if err != nil {
		log.Logger.Debug("invalid websocket message", zap.Error(err))
		ws.ChanSender <- nostr.NoticeEnvelope(err.Error())
		return
	}

	metrics.NostrRequestCounter.WithLabelValues(typ).Inc()
	handler, ok := wsMessageHandlers[typ]
	if !ok {
		log.Logger.Error("unknown event type", zap.String("type", typ))
		ws.ChanSender <- nostr.NoticeEnvelope("unknown event type " + typ)
		return
	}

	if notice := handler(ws, data); notice != "" {
		ws.ChanSender <- nostr.NoticeEnvelope(notice)
	}
}

func decodeMessage(message []byte) (string, dto.Data, error) {
	var requestRaw dto.Data
	if err := json.Unmarshal(message, &requestRaw); err != nil {
		return "", nil, fmt.Errorf("invalid JSON message")
	}
	if len(requestRaw) < 2 {
		return "", nil, fmt.Errorf("request has less than 2 parameters")
	}
	var typ string
	if err := json.Unmarshal(requestRaw[0], &typ); err != nil {
		return "", nil, fmt.Errorf("failed to decode event type")
	}
	return typ, requestRaw, nil
}

func handleNegOpen(ws *dto.WsServer, data dto.Data) string {
	metrics.NostrNegentropyCounter.WithLabelValues(dto.TypeNegOpen).Inc()
	if !config.Cfg.EnableNegentropy {
		return "Negentropy is not enabled"
	}
	if err := negentropy.HandleNegOpen(ws, data); err != nil {
		return err.Error()
	}
	return ""
}

func handleNegMsg(ws *dto.WsServer, data dto.Data) string {
	metrics.NostrNegentropyCounter.WithLabelValues(dto.TypeNegMsg).Inc()
	if !config.Cfg.EnableNegentropy {
		return "Negentropy is not enabled"
	}
	if err := negentropy.HandleNegMsg(ws, data); err != nil {
		return err.Error()
	}
	return ""
}

func handleNegHave(ws *dto.WsServer, data dto.Data) string {
	metrics.NostrNegentropyCounter.WithLabelValues(dto.TypeNegHave).Inc()
	if !config.Cfg.EnableNegentropy {
		return "Negentropy is not enabled"
	}
	if err := negentropy.HandleNegHave(ws, data); err != nil {
		return err.Error()
	}
	return ""
}

func handleNegNeed(ws *dto.WsServer, data dto.Data) string {
	metrics.NostrNegentropyCounter.WithLabelValues(dto.TypeNegNeed).Inc()
	if !config.Cfg.EnableNegentropy {
		return "Negentropy is not enabled"
	}
	if err := negentropy.HandleNegNeed(ws, data); err != nil {
		return err.Error()
	}
	return ""
}

func handleNegErr(_ *dto.WsServer, data dto.Data) string {
	metrics.NostrNegentropyCounter.WithLabelValues(dto.TypeNegErr).Inc()
	if len(data) > 1 {
		log.Logger.Info("Negentropy error", zap.Any("data", data[1]))
	}
	return ""
}

func handleNegClose(_ *dto.WsServer, data dto.Data) string {
	metrics.NostrNegentropyCounter.WithLabelValues(dto.TypeNegClose).Inc()
	if !config.Cfg.EnableNegentropy {
		return "Negentropy is not enabled"
	}
	if err := negentropy.HandleNegClose(data); err != nil {
		return err.Error()
	}
	if len(data) > 1 {
		log.Logger.Debug("Negentropy close", zap.Any("data", data[1]))
	}
	return ""
}

func handleClose(ws *dto.WsServer, data dto.Data) string {
	var id string
	if err := json.Unmarshal(data[1], &id); err != nil {
		return ""
	}
	listener.RemoveListenerId(ws, id)
	return ""
}
