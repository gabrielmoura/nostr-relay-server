package handler

import (
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/auth"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/count"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/event"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/req"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/goccy/go-json"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

func handleMessage(ws *dto.WsServer, message []byte) {

	var notice string
	defer func() {

		if notice != "" {
			log.Logger.Debug("Notice:", zap.String("notice", notice))
			ws.ChanSender <- nostr.NoticeEnvelope(notice)
		}
	}()

	var requestRaw dto.Data
	if err := json.Unmarshal(message, &requestRaw); err != nil {
		// stop silently
		return
	}

	if len(requestRaw) < 2 {
		notice = "request has less than 2 parameters"
		return
	}

	var typ string
	err := json.Unmarshal(requestRaw[0], &typ)
	if err != nil {
		notice = "failed to decode event type"
		log.Logger.Error("failed to decode event type", zap.Error(err))
		return
	}

	log.Logger.Debug("Event:", zap.Any("event", requestRaw))
	switch typ {
	case dto.TypeEVENT:
		notice = event.DoEVENT(ws, requestRaw)
	case dto.TypeCLOSE:
		notice = handleClose(ws, requestRaw)
	case dto.TypeREQ:
		notice = req.DoREQ(ws, requestRaw)
	case dto.TypeAUTH:
		notice = auth.DoAUTH(ws, requestRaw)
	case dto.TypeCOUNT:
		notice = count.DoCOUNT(ws, requestRaw)
	default:
		log.Logger.Error("Unknown event type", zap.String("type", typ))
		notice = "unknown event type " + typ
	}

}

func handleClose(ws *dto.WsServer, data dto.Data) string {
	var id string
	err := json.Unmarshal(data[1], &id)
	if err != nil {
		return ""
	}
	listener.RemoveListenerId(ws, id)

	return ""
}
