package req

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/infra/stream"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	policies2 "github.com/gabrielmoura/nostr-relay-server/internal/policies"
	"github.com/goccy/go-json"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
	"time"
)

func DoREQ(ws *dto.WsServer, data dto.Data) string {
	var id string
	err := json.Unmarshal(data[1], &id)
	if err != nil {
		log.Logger.Warn("failed to decode REQ id", zap.Error(err))
		return ""
	}
	if id == "" {
		return "REQ has no <id>"
	}

	filters := make(nostr.Filters, len(data)-2)
	for i, filterReq := range data[2:] {
		if err := json.Unmarshal(
			filterReq,
			&filters[i],
		); err != nil {
			return "failed to decode filter"
		}
	}

	if ok, reason := policies2.P.RejectReqBannedUser(ws); !ok {
		ws.ChanSender <- nostr.ClosedEnvelope{
			Reason:         reason,
			SubscriptionID: id,
		}
		return ""
	}

	// if the user is not authenticated, only allow certain kinds
	if !policies2.P.AcceptReqs(filters, ws) {
		ws.ChanSender <- nostr.ClosedEnvelope{
			Reason:         "auth-required: REQ filters are not accepted",
			SubscriptionID: id,
		}
		return ""
		//	return "REQ filters are not accepted"
	}

	for _, filter := range filters {
		if reject, msg := policies2.P.NoEmptyFilters(ws.Ctx, filter); reject {
			log.Logger.Warn(
				"empty-filter",
				zap.String("reason", msg),
				zap.String("filter", filter.String()),
				zap.String("ip", ws.Conn.IP()),
			)
			ws.ChanSender <- nostr.ClosedEnvelope{
				Reason:         msg,
				SubscriptionID: id,
			}
			return ""
		}

		// caso não haja autenticação, não permitir baixar eventos sem autor.
		if reject, msg := policies2.P.AntiSyncBots(ws.Ctx, filter); reject {
			log.Logger.Warn("anti-sync-bot", zap.String("reason", msg))
			ws.ChanSender <- nostr.ClosedEnvelope{
				Reason:         msg,
				SubscriptionID: id,
			}
			return ""
		}

		if reject, msg := policies2.P.CheckKindsAuth(filter, ws); reject {
			ws.ChanSender <- nostr.ClosedEnvelope{
				Reason:         msg,
				SubscriptionID: id,
			}
			return ""
		}

		events, err := db.DbQueries.QueryEventsChan(ws.Ctx, filter)
		if err != nil {
			log.Logger.Error("store", zap.Error(err))
			continue
		}

		// caso o limite seja 0, ele será substituído pelo limite padrão
		if filter.Limit == 0 {
			filter.Limit = config.Cfg.Relay.FilterLimit
		}
		// caso o limite seja maior que o limite padrão, ele será substituído pelo limite padrão
		if filter.Limit > config.Cfg.Relay.FilterLimit {
			filter.Limit = config.Cfg.Relay.FilterLimit
		}

		i := 0
		if events != nil {
			if len(events) == 0 {
				go stream.ForwardRequest(ws, filter, &id)
			}

			for event := range events {
				// regra para filtrar eventos que não devem ser enviados
				if ws.SkipEventFunc(event) {
					continue
				}

				ws.ChanSender <- nostr.EventEnvelope{SubscriptionID: &id, Event: *event}

				i++
				if i > filter.Limit {
					break
				}
			}

			// exhaust the channel (in case we broke out of it early) so it is closed by the storage
			for range events {
			}
		}
		countKindReqs(filter)
	}

	ws.ChanSender <- nostr.EOSEEnvelope(id)

	listener.SetListener(id, ws, filters)
	metrics.NostrRequestDuration.WithLabelValues("REQ").Observe(time.Since(ws.StartTime).Seconds())
	return ""
}

func countKindReqs(filter nostr.Filter) {
	for _, kind := range filter.Kinds {
		metrics.NostrKindReqCounter.WithLabelValues(metrics.GetKindName(kind)).Inc()
	}
}
