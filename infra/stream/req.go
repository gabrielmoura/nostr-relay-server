package stream

import (
	"context"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/pkg/nostrpool"
	"sync"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

var relayInitOnce sync.Once

// ForwardRequest forwards requests to the relays and processes the events.
func ForwardRequest(ws *dto.WsServer, filter nostr.Filter, id *string) {
	if !config.Cfg.StreamDown.Enabled {
		return
	}

	//initializeRelays(ws)
	//allEvents := collectUniqueEvents(ws, filter)

	allEvents, _ := nostrpool.Subscribe(nostr.Filters{filter})
	log.Logger.Info("Eventos encaminhados", zap.Int("total_events", len(allEvents)))

	for ev := range allEvents {
		metrics.NostrRelayRequestForwardedTotal.Inc()
		ws.ChanSender <- nostr.EventEnvelope{
			Event:          *ev,
			SubscriptionID: id,
		}
	}
}

// initializeRelays initializes relay connections if not already done.
func initializeRelays(ws *dto.WsServer) {
	relayInitOnce.Do(func() {
		for _, relayURL := range config.Cfg.StreamUp.Relays {
			connectRelay(ws, relayURL)
		}
	})
}

// collectUniqueEvents fetches events from all relays, ensuring uniqueness.
func collectUniqueEvents(ws *dto.WsServer, filter nostr.Filter) []nostr.Event {
	eventSet := make(map[string]struct{})
	var allEvents []nostr.Event

	for _, relay := range ws.StreamPoll {
		for _, ev := range fetchEvents(relay, filter, ws.Ctx) {
			if _, exists := eventSet[ev.ID]; !exists {
				eventSet[ev.ID] = struct{}{}
				allEvents = append(allEvents, ev)
			}
		}
	}

	return allEvents
}

// connectRelay establishes a connection to the relay and adds it to the pool.
func connectRelay(ws *dto.WsServer, relayURL string) {
	ctx, cancel := context.WithTimeout(ws.Ctx, 3*time.Second)
	defer cancel()

	relay, err := nostr.RelayConnect(ctx, relayURL)
	if err != nil {
		log.Logger.Error("Falha ao conectar no relay", zap.String("relay", relayURL), zap.Error(err))
		return
	}

	ws.Mutex.Lock()
	defer ws.Mutex.Unlock()

	ws.StreamPoll = append(ws.StreamPoll, relay)
	log.Logger.Info("Conexão estabelecida com relay", zap.String("relay", relayURL))
}

// fetchEvents retrieves events from a relay based on the given filter.
func fetchEvents(relay *nostr.Relay, filter nostr.Filter, mainCtx context.Context) []nostr.Event {
	ctx, cancel := context.WithTimeout(mainCtx, 3*time.Second)
	defer cancel()

	sub, err := relay.Subscribe(ctx, nostr.Filters{filter})
	if err != nil {
		log.Logger.Error(
			"Falha ao criar inscrição no relay",
			zap.String("relay", relay.URL),
			zap.Error(err),
		)
		return nil
	}

	var events []nostr.Event
	for ev := range sub.Events {
		events = append(events, *ev)
	}

	log.Logger.Debug(
		"Eventos recebidos do relay",
		zap.String("relay", relay.URL),
		zap.Int("event_count", len(events)),
	)
	return events
}
