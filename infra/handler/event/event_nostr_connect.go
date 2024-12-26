package event

import (
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip46"
	"go.uber.org/zap"
)

func handleNostrConnect(ws *dto.WsServer, evt *nostr.Event) string {
	sks := nip46.NewStaticKeySigner(ws.Challenge)
	_, _, eventResponse, err := sks.HandleRequest(ws.Ctx, evt)
	if err != nil {
		log.Logger.Error("failed to handle request", zap.Error(err))
		return ""
	}

	ws.ChanSender <- nostr.EventEnvelope{Event: eventResponse}
	return ""
}
