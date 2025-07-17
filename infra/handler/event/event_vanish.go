package event

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

// handleDeletionVanishEvent processes a vanish event (NIP-62) and deletes all events by the specified pubkey.
// It checks if the event is meant for the current relay by looking for a "relay" tag.
// https://github.com/nostr-protocol/nips/blob/master/62.md
func handleDeletionVanishEvent(ws *dto.WsServer, evt nostr.Event) string {
	if !isVanishEventForThisRelay(evt) {
		return ""
	}

	if err := db.DbQueries.DeleteAllEventsByPubkey(ws.Ctx, evt.PubKey); err != nil {
		log.Logger.Error("failed to delete events for pubkey on vanish event",
			zap.String("pubkey", evt.PubKey),
			zap.String("event_id", evt.ID),
			zap.Error(err),
		)
		ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: "Error: failed to delete events."}
		return ""
	}

	if err := db.DbQueries.InsertEvent(ws.Ctx, &evt); err != nil {
		log.Logger.Error("failed to insert vanish event after deleting user events",
			zap.String("pubkey", evt.PubKey),
			zap.String("event_id", evt.ID),
			zap.Error(err),
		)
		// Even if inserting the vanish event fails, the primary goal of deletion was successful.
		// We still inform the client of the success of the deletion.
		ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: true, Reason: "Notice: events deleted, but failed to store the vanish event."}
		return ""
	}

	log.Logger.Info("Successfully processed vanish event",
		zap.String("pubkey", evt.PubKey),
		zap.String("event_id", evt.ID),
	)
	ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: true}
	return ""
}

// isVanishEventForThisRelay checks if the vanish event is targeted to this relay.
func isVanishEventForThisRelay(evt nostr.Event) bool {
	if !config.Cfg.Relay.VanishEvent {
		return false
	}
	for _, tag := range evt.Tags {
		if len(tag) > 1 && tag[0] == "relay" {
			if tag[1] == config.Cfg.RelayInformation.CanonicalURL || tag[1] == "ALL_RELAYS" {
				return true
			}
		}
	}
	return false
}
