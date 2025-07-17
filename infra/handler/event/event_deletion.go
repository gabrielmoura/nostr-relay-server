package event

import (
	"context"
	"fmt"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
	"time"
)

func handleDeletionEvent(ws *dto.WsServer, evt nostr.Event) string {
	// event deletion -- nip09
	for _, tag := range evt.Tags {
		if len(tag) >= 2 && tag[0] == "e" {
			ctx, cancel := context.WithTimeout(ws.Ctx, time.Millisecond*200)
			defer cancel()

			// fetch event to be deleted
			res, err := db.DbQueries.QueryEventsChan(ctx, nostr.Filter{IDs: []string{tag[1]}})
			if err != nil {
				//ws.Conn.WriteJSON(nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: "failed to query for target event"})
				ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: "failed to query for target event"}
				return ""
			}

			var target *nostr.Event
			exists := false
			select {
			case target, exists = <-res:
			case <-ctx.Done():
			}
			if !exists {
				// this will happen if event is not in the database
				// or when when the query is taking too long, so we just give up
				continue
			}

			// check if this can be deleted
			if target.PubKey != evt.PubKey {
				//ws.Conn.WriteJSON(nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: "insufficient permissions"})
				ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: "insufficient permissions"}
				return ""
			}

			//if advancedDeleter != nil {
			//	advancedDeleter.BeforeDelete(ctx, tag[1], evt.PubKey)
			//}

			if err := db.DbQueries.DeleteEvent(ctx, target.ID, evt.ID); err != nil {
				//ws.Conn.WriteJSON(nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: fmt.Sprintf("error: %s", err.Error())})
				log.Logger.Warn("failed to delete event",
					zap.String("event_id", target.ID),
					zap.String("deletion_event_id", evt.ID),
					zap.String("reason", err.Error()),
				)
				metrics.NostrRelayEventDeletionFailures.Inc()
				ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: false, Reason: fmt.Sprintf("error: %s", err.Error())}
				return ""
			}
			metrics.NostrRelayEventDeletionSuccessful.Inc()

			//if advancedDeleter != nil {
			//	advancedDeleter.AfterDelete(tag[1], evt.PubKey)
			//}
		}
	}

	listener.NotifyListeners(&evt)
	//ws.Conn.WriteJSON(nostr.OKEnvelope{EventID: evt.ID, OK: true})
	ws.ChanSender <- nostr.OKEnvelope{EventID: evt.ID, OK: true}
	return ""
}
