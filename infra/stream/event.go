package stream

import (
	"context"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/infra/nostr-custom"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
	"slices"
	"time"
)

func ForwardEvent(event nostr.Event) {

	if config.Cfg.StreamUp.Enabled {
		metrics.NostrRelayEventForwardedTotal.Inc()
		kindsAccepted := []int{nostr.KindTextNote, nostr.KindDeletion, nostr.KindReaction, nostr.KindProfileMetadata, nostr.KindRepost, nostr_custom.KindEditContent}
		if slices.Contains(kindsAccepted, event.Kind) {
			for _, relay := range config.Cfg.StreamUp.Relays {
				go publishEvent(relay, event)
			}
		}
	}
}

func publishEvent(relay string, event nostr.Event) {
	// TODO: Da forma que é feita é aberta uma nova conexão para cada evento, isso pode ser otimizado, abrindo uma conexão (deixar no WsServer) e mantendo ela aberta para enviar vários eventos.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	rs, err := nostr.RelayConnect(ctx, relay)
	if err != nil {
		log.Logger.Error("failed to connect to relay", zap.Error(err))
	}
	defer func() {
		rs.Close()
		cancel()
	}()

	if err := rs.Publish(ctx, event); err != nil {
		metrics.NostrRelayEventForwardedFailuresTotal.Inc()
		log.Logger.Error(
			"failed to Forward event",
			zap.String("rs", relay),
			zap.String("ID", event.ID),
			zap.Error(err),
		)
		return
	}
	log.Logger.Debug(
		"Event forwarded",
		zap.String("ID", event.ID),
		zap.String("rs", relay),
	)
}
