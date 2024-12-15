package req

import (
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
	json.Unmarshal(data[1], &id)
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

	if !ws.AcceptReq(filters) {
		//ws.Conn.WriteJSON(nostr.ClosedEnvelope{
		//	Reason:         "auth-required: REQ filters are not accepted",
		//	SubscriptionID: id,
		//})
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
		if ws.Authed != "" {
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
			log.Logger.ErrorContext(ws.Ctx, "store", slog.String("error", err.Error()))
			continue
		}

		// ensures the client won't be bombarded with events in case Storage doesn't do limits right
		if filter.Limit == 0 {
			filter.Limit = 9999999999
		}
		i := 0
		if events != nil {
			for event := range events {
				// regra para filtrar eventos que não devem ser enviados
				if ws.SkipEventFunc(event) {
					continue
				}

				//ws.Conn.WriteJSON(nostr.EventEnvelope{SubscriptionID: &id, Event: *event})
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

	//ws.Conn.WriteJSON(nostr.EOSEEnvelope(id))
	ws.ChanSender <- nostr.EOSEEnvelope(id)

	listener.SetListener(id, ws, filters)
	return ""
}
