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
	"log/slog"
)

func handleMessage(ws *dto.WsServer, message []byte) {
	//defer func() {
	//	log.Logger.Info("Closing connection")
	//	conn.Close()
	//}()

	var notice string
	defer func() {

		if notice != "" {
			log.Logger.Debug("Notice:", slog.Any("notice", notice))
			//ws.Conn.WriteJSON(nostr.NoticeEnvelope(notice))
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
	json.Unmarshal(requestRaw[0], &typ)

	log.Logger.DebugContext(ws.Ctx, "Event:", slog.Any("event", requestRaw))
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
		log.Logger.Error("Unknown event type:", typ)
		notice = "unknown event type " + typ
	}

}

func handleClose(ws *dto.WsServer, data dto.Data) string {
	//log.Logger.Info("Close", slog.String("remote_addr", req.RequestHttp.IP), slog.String("id", string(req.Data[1])))
	//req.Conn.WriteJSON(nostr.CloseEnvelope(req.Data[1]))
	//req.Conn.Close()
	var id string
	json.Unmarshal(data[1], &id)
	listener.RemoveListenerId(ws, id)

	return ""
}
