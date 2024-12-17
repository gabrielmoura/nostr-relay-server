package req

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/goccy/go-json"
	"github.com/nbd-wtf/go-nostr"
	"log/slog"
	"slices"
)

func DoREQ(ws *dto.WsServer, data dto.Data) string {
	var id string
	err := json.Unmarshal(data[1], &id)
	if err != nil {
		log.Logger.WarnContext(ws.Ctx, "failed to decode REQ id", slog.AnyValue(err))
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

		// prevent kind-4 events from being returned to unauthed users,
		//   only when authentication is a thing
		if slices.Contains(filter.Kinds, nostr.KindEncryptedDirectMessage) {
			senders := filter.Authors
			receivers, _ := filter.Tags["p"]
			switch {
			case ws.Authed == "":
				// not authenticated
				return "restricted: this relay does not serve kind-4 to unauthenticated users, does your client implement NIP-42?"
			case len(senders) == 1 && len(receivers) < 2 && (senders[0] == ws.Authed):
				// allowed filter:ws.Authed is sole sender (filter specifies one or all receivers)
			case len(receivers) == 1 && len(senders) < 2 && (receivers[0] == ws.Authed):
				// allowed filter: req.authed is sole receiver (filter specifies one or all senders)
			default:
				// restricted filter: do not return any events,
				//   even if other elements in filters array were not restricted).
				//   client should know better.
				return "restricted: authenticated user does not have authorization for requested filters."
			}
		}

		if slices.Contains(filter.Kinds, nostr.KindApplicationSpecificData) {
			switch {
			case ws.Authed == "":
				return "restricted: é necessário autenticação para acessar eventos do tipo KindApplicationSpecificData"
			case !slices.Contains(filter.Authors, ws.Authed):
				return "restricted: usuário autenticado não tem autorização para acessar eventos do tipo KindApplicationSpecificData"
			}
		}

		events, err := db.DbQueries.QueryEvents(ws.Ctx, filter)
		if err != nil {
			log.Logger.ErrorContext(ws.Ctx, "store", slog.AnyValue(err))
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
	}

	ws.ChanSender <- nostr.EOSEEnvelope(id)

	listener.SetListener(id, ws, filters)
	return ""
}
