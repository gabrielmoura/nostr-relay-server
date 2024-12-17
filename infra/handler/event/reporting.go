package event

import (
	"context"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

// reportingEvent tratamento de eventos de report
func reportingEvent(ctx context.Context, event nostr.Event) {
	// TODO: Caso haja 3 reports para um mesmo evento, o evento deve ser deletado
	// TODO; Caso haja 5 reports para a mesma public key, a public key deve ser banida
	var (
		key         string
		reason      string
		denunciator string
	)
	for _, tag := range event.Tags {
		if tag[0] == "p" {
			key = tag[1]
			reason = tag[2]
		}
		if tag[0] == "e" {
			denunciator = tag[1]
		}
	}
	if key == "" || reason == "" || denunciator == "" {
		return
	}

	log.Logger.Info(
		"reporting event",
		zap.String("key", event.ID),
		zap.String("pubKey", key),
		zap.String("reason", reason),
		zap.String("denunciator", denunciator),
	)

	// Busca todos os eventos de report para a public key
	count, err := db.DbQueries.GetCountReportsKey(ctx, key)
	if err != nil {
		log.Logger.Error("failed to get count reports key", zap.Error(err))
		return
	}
	// atingindo a marca de 5 reports, a public key é banida
	if count+1 >= 5 {
		err := db.DbQueries.BanUserByPubKey(ctx, key, reason, []string{})
		if err != nil {
			log.Logger.Error("failed to ban user", zap.Error(err))
			return
		}
	}

}
