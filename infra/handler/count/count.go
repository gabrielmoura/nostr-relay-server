package count

import (
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	policies "github.com/gabrielmoura/nostr-relay-server/internal/policies"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
	"time"
)

func DoCOUNT(ws *dto.WsServer, data dto.Data) string {
	var id string
	err := json.Unmarshal(data[1], &id)
	if err != nil {
		log.Logger.Warn("failed to decode COUNT id", zap.Error(err))
		return ""
	}
	if id == "" {
		return "COUNT has no <id>"
	}

	total := int64(0)
	filters := make(nostr.Filters, len(data)-2)

	for i, filterReq := range data[2:] {
		if err := json.Unmarshal(filterReq, &filters[i]); err != nil {
			return "failed to decode filter"
		}

	}

	normalized, reject, reason := policies.P.ValidateCount(ws.Ctx, ws, filters)
	if reject {
		return reason
	}

	for _, filter := range normalized {
		count, err := db.DbQueries.CountEvents(ws.Ctx, filter)
		if err != nil {
			log.Logger.Error("store: %v", zap.Error(err))
			continue
		}
		total += count
	}

	ws.ChanSender <- []any{"COUNT", id, map[string]int64{"count": total}}
	metrics.NostrRequestDuration.WithLabelValues("COUNT").Observe(time.Since(ws.StartTime).Seconds())

	return ""
}
