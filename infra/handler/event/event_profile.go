package event

import (
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/gabrielmoura/nostr-relay-server/internal/profile"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

func handleProfile(ws *dto.WsServer, evt *nostr.Event) {

	if evt.Kind != nostr.KindProfileMetadata {
		return
	}

	if err := profile.Upsert(ws.Ctx, evt); err != nil {
		log.Logger.Error("failed to insert profile", zap.Error(err))
	}
}
