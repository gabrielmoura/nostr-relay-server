package policies

import (
	"context"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/goccy/go-json"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

func (p Policies) AcceptEvent(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
	reason, exists, err := db.DbQueries.GetUserBannedByKey(ctx, event.PubKey)
	if err != nil {
		log.Logger.Error("Erro ao verificar se o usuário está banido", zap.Error(err))
		return true, "error: " + err.Error()
	}
	if exists {
		log.Logger.Info("Usuário banido", zap.String("reason", reason))
		return true, "banned user: " + reason
	}

	jsonb, _ := json.Marshal(event)
	if len(jsonb) > config.Cfg.Relay.MaxEventSize {
		log.Logger.Debug(
			"very big event",
			zap.Int("size", len(jsonb)),
			zap.Int("max", config.Cfg.Relay.MaxEventSize),
		)
		return true, "very big event"
	}

	return false, "event accepted"
}
