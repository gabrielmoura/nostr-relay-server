package stream

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/gabrielmoura/nostr-relay-server/pkg/nostrpool"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

// ForwardRequest forwards requests to the relays and processes the events.
func ForwardRequest(ws *dto.WsServer, filter nostr.Filter, id *string) {
	if !config.Cfg.StreamDown.Enabled {
		return
	}

	allEvents, err := nostrpool.Subscribe(nostr.Filters{filter})
	if err != nil {
		log.Logger.Warn("Erro ao coletar eventos do Relay Pool", zap.Error(err))
		return
	}

	for ev := range allEvents {
		metrics.NostrRelayRequestForwardedTotal.Inc()
		db.DbQueries.InsertEvent(ws.Ctx, ev)
		ws.ChanSender <- nostr.EventEnvelope{
			Event:          *ev,
			SubscriptionID: id,
		}
	}
}
