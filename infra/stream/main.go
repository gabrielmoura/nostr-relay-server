package stream

import (
	"context"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	nostr_custom "github.com/gabrielmoura/nostr-relay-server/infra/nostr-custom"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
	"slices"
	"time"
)

// TODO: Implementar funções para encaminhar pedido e repostas para outros servidores definidos na configuração

func ForwardRequest(ws *dto.WsServer, filter nostr.Filter, id *string) {
	if !config.Cfg.StreamDown.Enabled {
		return
	}

	var allEvents []nostr.Event
	eventSet := make(map[string]struct{}) // Para evitar duplicados.

	for _, relay := range config.Cfg.StreamUp.Relays {
		events := getEvents(ws.Ctx, relay, filter)
		for _, ev := range events {
			if _, exists := eventSet[ev.ID]; !exists {
				eventSet[ev.ID] = struct{}{}
				allEvents = append(allEvents, ev)
			}
		}
	}

	log.Logger.Debug("Eventos encaminhados", zap.Int("total_events", len(allEvents)))

	for _, ev := range allEvents {
		ws.ChanSender <- nostr.EventEnvelope{
			Event:          ev,
			SubscriptionID: id,
		}
	}

}

func getEvents(wsCtx context.Context, relay string, filter nostr.Filter) []nostr.Event {
	ctx, cancel := context.WithTimeout(wsCtx, 3*time.Second)
	defer cancel()

	rs, err := nostr.RelayConnect(ctx, relay)
	if err != nil {
		log.Logger.Error("Falha ao conectar no relay", zap.String("relay", relay), zap.Error(err))
		return nil
	}
	defer rs.Close()

	sub, err := rs.Subscribe(ctx, nostr.Filters{filter})
	if err != nil {
		log.Logger.Error("Falha ao criar inscrição no relay", zap.String("relay", relay), zap.Error(err))
		return nil
	}

	var events []nostr.Event
	for ev := range sub.Events {
		events = append(events, *ev)
	}

	log.Logger.Debug("Eventos recebidos do relay", zap.String("relay", relay), zap.Int("event_count", len(events)))
	return events
}

func ForwardEvent(event nostr.Event) {

	if config.Cfg.StreamUp.Enabled {
		kindsAccepted := []int{nostr.KindTextNote, nostr.KindDeletion, nostr.KindReaction, nostr.KindProfileMetadata, nostr.KindRepost, nostr_custom.KindEditContent}
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
