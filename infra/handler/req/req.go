package req

import (
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/auth"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/infra/stream"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/gabrielmoura/nostr-relay-server/internal/groups"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	policies2 "github.com/gabrielmoura/nostr-relay-server/internal/policies"
	"github.com/gabrielmoura/nostr-relay-server/internal/security"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
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

	normalizedFilters, reject, reason := policies2.P.ValidateReq(ws.Ctx, ws, filters)
	if reject {
		if config.Cfg.Ws.RequireAuthForReq() && ws.Authed == "" {
			auth.SendAuthChallenge(ws)
		}
		ws.ChanSender <- security.ClosedRejectReason(id, reason)
		return ""
	}

	maxSubscriptions := config.Cfg.RelayInformation.MaxSubscriptions()
	if maxSubscriptions > 0 && !listener.HasSubscription(ws, id) && listener.SubscriptionCount(ws) >= maxSubscriptions {
		ws.ChanSender <- security.ClosedReject(id, security.PrefixRateLimited, "max subscriptions reached")
		return ""
	}

	for _, filter := range normalizedFilters {

		events, handled, err := groups.QueryEvents(ws.Ctx, ws.Authed, filter, db.DbQueries.QueryEventsChan)
		if !handled {
			events, err = db.DbQueries.QueryEventsChan(ws.Ctx, filter)
		}
		if err != nil {
			log.Logger.Error("store", zap.Error(err))
			continue
		}

		i := 0
		if events != nil {
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

			if i == 0 {
				stream.ForwardRequest(ws, filter, &id)
			}

			// exhaust the channel (in case we broke out of it early) so it is closed by the storage
			for range events {
			}
		}
		countKindReqs(filter)
	}

	ws.ChanSender <- nostr.EOSEEnvelope(id)

	listener.SetListener(id, ws, normalizedFilters)
	metrics.NostrRequestDuration.WithLabelValues("REQ").Observe(time.Since(ws.StartTime).Seconds())
	return ""
}

func countKindReqs(filter nostr.Filter) {
	for _, kind := range filter.Kinds {
		metrics.NostrKindReqCounter.WithLabelValues(metrics.GetKindName(kind)).Inc()
	}
}
