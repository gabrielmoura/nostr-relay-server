package listener

import (
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/nbd-wtf/go-nostr"
	"sync"
	"sync/atomic"
)

type Listener struct {
	filters nostr.Filters
}

var (
	listeners      = make(map[*dto.WsServer]map[string]*Listener)
	listenersMutex sync.RWMutex // Using a RWMutex for read-heavy operations
	listenerCount  atomic.Int32 // Use atomic for thread-safe counter
)

// GetListeningFilters retrieves all distinct filters currently active.
func GetListeningFilters() nostr.Filters {
	listenersMutex.RLock()
	defer listenersMutex.RUnlock()

	respfilters := nostr.Filters{}
	uniqueFilters := make(map[string]struct{})

	for _, connListeners := range listeners {
		for _, listener := range connListeners {
			for _, listenerFilter := range listener.filters {
				filterKey := listenerFilter.String()
				if _, exists := uniqueFilters[filterKey]; !exists {
					uniqueFilters[filterKey] = struct{}{}
					respfilters = append(respfilters, listenerFilter)
				}
			}
		}
	}
	return respfilters
}

// SetListener adds a listener with the given ID and filters to a WebSocket server.
func SetListener(id string, ws *dto.WsServer, filters nostr.Filters) {
	listenersMutex.Lock()
	defer listenersMutex.Unlock()

	if listeners[ws] == nil {
		listeners[ws] = make(map[string]*Listener)
	}

	if _, exists := listeners[ws][id]; !exists {
		//atomic.AddInt32(&listenerCount, 1) // Increment count on new listener
		listenerCount.Add(1)
	}

	listeners[ws][id] = &Listener{filters: filters}
}

// RemoveListenerId removes a specific listener ID from a WebSocket server.
func RemoveListenerId(ws *dto.WsServer, id string) {
	listenersMutex.Lock()
	defer listenersMutex.Unlock()

	if subs, ok := listeners[ws]; ok {
		if _, exists := subs[id]; exists {
			delete(subs, id)
			listenerCount.Add(-1)
		}
		if len(subs) == 0 {
			delete(listeners, ws)
		}
	}
}

// RemoveListener removes all listeners associated with a WebSocket server.
func RemoveListener(ws *dto.WsServer) {
	listenersMutex.Lock()
	defer listenersMutex.Unlock()

	if subs, ok := listeners[ws]; ok {
		removedCount := len(subs)
		delete(listeners, ws)
		listenerCount.Add(int32(-removedCount))
	}
}

// NotifyListeners sends an event to all matching listeners.
func NotifyListeners(event *nostr.Event) {
	listenersMutex.RLock()
	defer listenersMutex.RUnlock()

	for ws, subs := range listeners {
		for id, listener := range subs {
			if listener.filters.Match(event) {
				ws.ChanSender <- nostr.EventEnvelope{
					SubscriptionID: &id,
					Event:          *event,
				}
			}
		}
	}
}

// GetCount returns the current count of active listeners.
func GetCount() int {
	return int(listenerCount.Load())
}
