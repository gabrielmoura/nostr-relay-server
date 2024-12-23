package req

import (
	"fmt"
	"github.com/fiatjaf/khatru/policies"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/infra/stream"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/goccy/go-json"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
	"slices"
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

	// if the user is not authenticated, only allow certain kinds
	if !ws.AcceptReqs(filters) {
		ws.ChanSender <- nostr.ClosedEnvelope{
			Reason:         "auth-required: REQ filters are not accepted",
			SubscriptionID: id,
		}
		return ""
		//	return "REQ filters are not accepted"
	}

	for _, filter := range filters {
		//if ok, err := policies.NoEmptyFilters(ws.Ctx, filter); !ok {
		//	log.Logger.Warn("empty-filter", zap.String("reason", err))
		//	ws.ChanSender <- nostr.ClosedEnvelope{
		//		Reason:         err,
		//		SubscriptionID: id,
		//	}
		//	return ""
		//}

		// caso não haja autenticação, não permitir baixar eventos sem autor.
		if ok, err := policies.AntiSyncBots(ws.Ctx, filter); ok {
			if ws.Authed == "" {
				log.Logger.Warn("anti-sync-bot", zap.String("reason", err))
				ws.ChanSender <- nostr.ClosedEnvelope{
					Reason:         fmt.Sprintf("auth-required: %s", err),
					SubscriptionID: id,
				}
				return ""
			}
		}

		// prevent kind-4 events from being returned to unauthed users,
		//   only when authentication is a thing
		if slices.Contains(filter.Kinds, nostr.KindEncryptedDirectMessage) {
			senders := filter.Authors
			receivers, _ := filter.Tags["p"]
			switch {
			case ws.Authed == "":
				// not authenticated
				//return "auth-required: this relay does not serve kind-4 to unauthenticated users, does your client implement NIP-42?"
				ws.ChanSender <- nostr.ClosedEnvelope{
					Reason:         "auth-required: this relay does not serve kind-4 to unauthenticated users, does your client implement NIP-42?",
					SubscriptionID: id,
				}
				return ""
			case len(senders) == 1 && len(receivers) < 2 && (senders[0] == ws.Authed):
				// allowed filter:ws.Authed is sole sender (filter specifies one or all receivers)
			case len(receivers) == 1 && len(senders) < 2 && (receivers[0] == ws.Authed):
				// allowed filter: req.authed is sole receiver (filter specifies one or all senders)
			default:
				// restricted filter: do not return any events,
				//   even if other elements in filters array were not restricted).
				//   client should know better.
				//return "auth-required: authenticated user does not have authorization for requested filters."
				ws.ChanSender <- nostr.ClosedEnvelope{
					Reason:         "auth-required: authenticated user does not have authorization for requested filters.",
					SubscriptionID: id,
				}
				return ""
			}
		}

		if slices.Contains(filter.Kinds, nostr.KindApplicationSpecificData) {
			switch {
			case ws.Authed == "":
				//return "restricted: é necessário autenticação para acessar eventos do tipo KindApplicationSpecificData"
				ws.ChanSender <- nostr.ClosedEnvelope{
					Reason:         "auth-required: é necessário autenticação para acessar eventos do tipo KindApplicationSpecificData",
					SubscriptionID: id,
				}
				return ""
			case !slices.Contains(filter.Authors, ws.Authed):
				//return "auth-required: usuário autenticado não tem autorização para acessar eventos do tipo KindApplicationSpecificData"
				ws.ChanSender <- nostr.ClosedEnvelope{
					Reason:         "auth-required: usuário autenticado não tem autorização para acessar eventos do tipo KindApplicationSpecificData",
					SubscriptionID: id,
				}
				return ""
			}
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
