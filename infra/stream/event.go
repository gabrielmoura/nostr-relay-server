package stream

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/infra/nostr-custom"
	"github.com/gabrielmoura/nostr-relay-server/pkg/nostrpool"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
	"slices"
)

func ForwardEvent(event nostr.Event) {

	if config.Cfg.StreamUp.Enabled {
		metrics.NostrRelayEventForwardedTotal.Inc()
		kindsAccepted := []int{nostr.KindTextNote, nostr.KindDeletion, nostr.KindReaction, nostr.KindProfileMetadata, nostr.KindRepost, nostr_custom.KindEditContent}
		if slices.Contains(kindsAccepted, event.Kind) {

			if err := nostrpool.Publish(&event); err != nil {
				metrics.NostrRelayEventForwardedFailuresTotal.Inc()
				log.Logger.Warn("failed to publish event to relay pool", zap.Error(err), zap.String("event_id", event.ID))
			} else {
				metrics.NostrRelayEventForwardedTotal.Inc()
				log.Logger.Debug("Event forwarded to relay pool", zap.String("ID", event.ID))
			}
		}
	}
}
