package count

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/goccy/go-json"
	"github.com/nbd-wtf/go-nostr"
	"log/slog"
	"slices"
)

func DoCOUNT(ws *dto.WsServer, data dto.Data) string {
	var id string
	json.Unmarshal(data[1], &id)
	if id == "" {
		return "COUNT has no <id>"
	}

	total := int64(0)
	filters := make(nostr.Filters, len(data)-2)

	for i, filterReq := range data[2:] {
		if err := json.Unmarshal(filterReq, &filters[i]); err != nil {
			return "failed to decode filter"
		}

		filter := filters[i]

		// prevent kind-4 events from being returned to unauthed users,
		//   only when authentication is a thing
		if config.Cfg.Ws.Auth {
			if slices.Contains(filter.Kinds, 4) {
				senders := filter.Authors
				receivers, _ := filter.Tags["p"]
				switch {
				case ws.Authed == "":
					// not authenticated
					return "restricted: this relay does not serve kind-4 to unauthenticated users, does your client implement NIP-42?"
				case len(senders) == 1 && len(receivers) < 2 && (senders[0] == ws.Authed):
					// allowed filter: ws.authed is sole sender (filter specifies one or all receivers)
				case len(receivers) == 1 && len(senders) < 2 && (receivers[0] == ws.Authed):
					// allowed filter: ws.authed is sole receiver (filter specifies one or all senders)
				default:
					// restricted filter: do not return any events,
					//   even if other elements in filters array were not restricted).
					//   client should know better.
					return "restricted: authenticated user does not have authorization for requested filters."
				}
			}
		}

		count, err := db.DbQueries.CountEvents(ws.Ctx, filter)
		if err != nil {
			log.Logger.Error("store: %v", slog.AnyValue(err))
			continue
		}
		total += count
	}

	//ws.WriteJSON([]interface{}{"COUNT", id, map[string]int64{"count": total}})
	ws.ChanSender <- []interface{}{"COUNT", id, map[string]int64{"count": total}}

	return ""
}
