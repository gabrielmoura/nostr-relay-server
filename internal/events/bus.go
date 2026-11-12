// Package events provides a small, bounded in-process event bus used by the
// admin GraphQL subscriptions and by durable job/state transitions.
//
// The bus is intentionally ephemeral. It never stores events: a subscriber
// that falls behind is dropped instead of blocking producers or growing
// without bound. Durable state is always recovered through normal queries.
package events

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
)

const (
	TopicPresence        = "admin.presence"
	TopicJob             = "job.changed"
	TopicBlossomPlan     = "blossom.plan.changed"
	TopicBlossomObject   = "blossom.object.changed"
	defaultSubscriberBuf = 64
)

// Event is one published topic payload. Payload is always JSON so local and
// Redis-bridged events share the same decoding path.
type Event struct {
	Topic    string
	Payload  json.RawMessage
	External bool
}

type subscriber struct {
	topics map[string]struct{}
	ch     chan Event
}

// Bus fans events out to subscribers. Each subscriber owns a bounded channel;
// publishing is non-blocking and drops the event for slow consumers.
type Bus struct {
	mu      sync.RWMutex
	nextID  uint64
	subs    map[uint64]*subscriber
	dropped atomic.Uint64
}

func NewBus() *Bus {
	return &Bus{subs: make(map[uint64]*subscriber)}
}

var defaultBus = NewBus()

// Default returns the process-wide bus.
func Default() *Bus { return defaultBus }

// Dropped reports how many events were discarded because a subscriber was
// slower than the publisher.
func (b *Bus) Dropped() uint64 { return b.dropped.Load() }

// Publish marshals payload as JSON and delivers it to matching subscribers.
func (b *Bus) Publish(topic string, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	b.PublishRaw(topic, raw)
}

// PublishRaw delivers an already-encoded JSON payload.
func (b *Bus) PublishRaw(topic string, payload json.RawMessage) {
	b.deliver(topic, payload, false)
}

// PublishExternal delivers an event that arrived from another relay instance,
// for example through Redis Pub/Sub. Bridges use External to avoid rebroadcast
// loops.
func (b *Bus) PublishExternal(topic string, payload json.RawMessage) {
	b.deliver(topic, payload, true)
}

func (b *Bus) deliver(topic string, payload json.RawMessage, external bool) {
	if b == nil || topic == "" || len(payload) == 0 {
		return
	}
	event := Event{Topic: topic, Payload: payload, External: external}

	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, sub := range b.subs {
		if !sub.matches(topic) {
			continue
		}
		select {
		case sub.ch <- event:
		default:
			b.dropped.Add(1)
		}
	}
}

// Subscribe returns a channel that receives events for the given topics. With
// no topics every event is delivered. The channel is closed when ctx is done.
func (b *Bus) Subscribe(ctx context.Context, buffer int, topics ...string) <-chan Event {
	if buffer <= 0 {
		buffer = defaultSubscriberBuf
	}
	topicsSet := make(map[string]struct{}, len(topics))
	for _, topic := range topics {
		if topic != "" {
			topicsSet[topic] = struct{}{}
		}
	}
	sub := &subscriber{topics: topicsSet, ch: make(chan Event, buffer)}

	b.mu.Lock()
	b.nextID++
	id := b.nextID
	b.subs[id] = sub
	b.mu.Unlock()

	go func() {
		<-ctx.Done()
		b.mu.Lock()
		if current, ok := b.subs[id]; ok && current == sub {
			delete(b.subs, id)
			close(sub.ch)
		}
		b.mu.Unlock()
	}()

	return sub.ch
}

func (s *subscriber) matches(topic string) bool {
	if len(s.topics) == 0 {
		return true
	}
	_, ok := s.topics[topic]
	return ok
}

// Decode unmarshals an event payload into T.
func Decode[T any](event Event) (T, error) {
	var value T
	err := json.Unmarshal(event.Payload, &value)
	return value, err
}
