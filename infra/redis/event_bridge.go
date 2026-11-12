package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/events"
	"go.uber.org/zap"
)

const EventBridgeChannel = "nostr-relay:admin-events:v1"

type eventEnvelope struct {
	Origin  string          `json:"origin"`
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

// StartEventBridge mirrors the process-local event bus through Redis Pub/Sub
// when Redis is enabled. It is optional: without Redis, subscriptions still
// work for events produced by this process.
func StartEventBridge(ctx context.Context, bus *events.Bus) {
	if bus == nil {
		return
	}
	client := GetClient()
	if client == nil || !client.IsEnabled() {
		return
	}

	origin := bridgeOrigin()
	local := bus.Subscribe(ctx, 256)

	go func() {
		for event := range local {
			if event.External {
				continue
			}
			envelope := eventEnvelope{Origin: origin, Topic: event.Topic, Payload: event.Payload}
			payload, err := json.Marshal(envelope)
			if err != nil {
				continue
			}
			publishCtx, cancel := context.WithTimeout(ctx, time.Second)
			if err := client.Publish(publishCtx, EventBridgeChannel, payload); err != nil && ctx.Err() == nil {
				log.Logger.Debug("admin event bridge publish failed", zap.Error(err))
			}
			cancel()
		}
	}()

	sub := client.Subscribe(ctx, EventBridgeChannel)
	go func() {
		defer func() { _ = sub.Close() }()
		channel := sub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case message, ok := <-channel:
				if !ok {
					return
				}
				var envelope eventEnvelope
				if err := json.Unmarshal([]byte(message.Payload), &envelope); err != nil {
					continue
				}
				if envelope.Origin == origin || envelope.Topic == "" || len(envelope.Payload) == 0 {
					continue
				}
				bus.PublishExternal(envelope.Topic, envelope.Payload)
			}
		}
	}()
}

func bridgeOrigin() string {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		return "relay"
	}
	return hex.EncodeToString(buffer)
}
