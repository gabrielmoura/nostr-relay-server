package stream

import (
	"context"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
	"slices"
	"time"
)

// TODO: Implementar funções para encaminhar pedido e repostas para outros servidores definidos na configuração

func ForwardRequest(ws *dto.WsServer, filter nostr.Filter, id *string) {
	if config.Cfg.StreamDown.Enabled {
		events := make([]nostr.Event, 0)
		for _, relay := range config.Cfg.StreamUp.Relays {
			getEvents(relay, filter, events)
		}
		log.Logger.Info("Eventos encaminhados", zap.Int("total_events", len(events)))

		for _, ev := range events {
			//	ws.ChanSender <- nostr.EventEnvelope{
			//		Event:          ev,
			//		SubscriptionID: id,
			//	}
			//	// TODO: verificar se o evento já existe no banco de dados antes de enviar.
			//	//db.DbQueries.InsertEvent(ws.Ctx, &ev)
			listener.NotifyListeners(&ev)
		}
	}

}

func getEvents(relay string, filter nostr.Filter, events []nostr.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	rs, err := nostr.RelayConnect(ctx, relay)
	if err != nil {
		log.Logger.Error("failed to connect to relay", zap.Error(err))
	}
	defer func() {
		rs.Close()
		cancel()
	}()
	sub, _ := rs.Subscribe(ctx, nostr.Filters{
		filter,
	})

	for ev := range sub.Events {
		for _, e := range events {
			if e.ID != ev.ID {
				events = append(events, *ev)
			}
		}
	}
	return
}
func ForwardEvent(event nostr.Event) {

	if config.Cfg.StreamUp.Enabled {
		kindsAccepted := []int{nostr.KindTextNote, nostr.KindDeletion, nostr.KindReaction, nostr.KindProfileMetadata, nostr.KindRepost}
		if slices.Contains(kindsAccepted, event.Kind) {
			for _, relay := range config.Cfg.StreamUp.Relays {
				go publishEvent(relay, event)
			}
		}
	}
}

func publishEvent(relay string, event nostr.Event) {
	// TODO: Da forma que é feita é aberta uma nova conexão para cada evento, isso pode ser otimizado, abrindo uma conexão (deixar no WsServer) e mantendo ela aberta para enviar vários eventos.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	rs, err := nostr.RelayConnect(ctx, relay)
	if err != nil {
		log.Logger.Error("failed to connect to relay", zap.Error(err))
	}
	defer func() {
		rs.Close()
		cancel()
	}()

	if err := rs.Publish(ctx, event); err != nil {
		log.Logger.Error(
			"failed to Forward event",
			zap.String("rs", relay),
			zap.String("ID", event.ID),
			zap.Error(err),
		)
		return
	}
	log.Logger.Debug(
		"Event forwarded",
		zap.String("ID", event.ID),
		zap.String("rs", relay),
	)
}
