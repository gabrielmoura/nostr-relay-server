package event

import (
	"context"
	"sync"

	"github.com/gabrielmoura/nostr-relay-server/internal/cordn"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/nbd-wtf/go-nostr"
)

var cordnCoordinator struct {
	sync.RWMutex
	value *cordn.NostrCoordinator
}

// SetCordnCoordinator installs the optional Cordn transport during server
// startup. A nil coordinator disables handling and leaves the existing relay
// event path untouched.
func SetCordnCoordinator(value *cordn.NostrCoordinator) {
	cordnCoordinator.Lock()
	defer cordnCoordinator.Unlock()
	cordnCoordinator.value = value
}

func handleCordnEvent(ws *dto.WsServer, evt *nostr.Event) bool {
	cordnCoordinator.RLock()
	coordinator := cordnCoordinator.value
	cordnCoordinator.RUnlock()
	if coordinator == nil {
		return false
	}
	return coordinator.Handle(ws.Ctx, evt, func(response nostr.Event) {
		select {
		case ws.ChanSender <- nostr.EventEnvelope{Event: response}:
		case <-requestContext(ws.Ctx).Done():
		}
	})
}

func requestContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
