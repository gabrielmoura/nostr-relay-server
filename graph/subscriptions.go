package graph

import (
	"context"
	"sync"

	"github.com/gabrielmoura/nostr-relay-server/graph/model"
)

type SubscriptionsBroker struct {
	mu           sync.RWMutex
	planSubs     map[string]chan *model.AdminBlossomPlan
	objectSubs   map[string]chan *model.AdminBlossomObject
}

var (
	defaultBroker *SubscriptionsBroker
	brokerOnce    sync.Once
)

func GetSubscriptionsBroker() *SubscriptionsBroker {
	brokerOnce.Do(func() {
		defaultBroker = &SubscriptionsBroker{
			planSubs:   make(map[string]chan *model.AdminBlossomPlan),
			objectSubs: make(map[string]chan *model.AdminBlossomObject),
		}
	})
	return defaultBroker
}

func (b *SubscriptionsBroker) SubscribePlanChanges(ctx context.Context, id string) <-chan *model.AdminBlossomPlan {
	ch := make(chan *model.AdminBlossomPlan, 32)
	b.mu.Lock()
	b.planSubs[id] = ch
	b.mu.Unlock()

	go func() {
		<-ctx.Done()
		b.mu.Lock()
		delete(b.planSubs, id)
		close(ch)
		b.mu.Unlock()
	}()

	return ch
}

func (b *SubscriptionsBroker) SubscribeObjectChanges(ctx context.Context, id string) <-chan *model.AdminBlossomObject {
	ch := make(chan *model.AdminBlossomObject, 32)
	b.mu.Lock()
	b.objectSubs[id] = ch
	b.mu.Unlock()

	go func() {
		<-ctx.Done()
		b.mu.Lock()
		delete(b.objectSubs, id)
		close(ch)
		b.mu.Unlock()
	}()

	return ch
}

func (b *SubscriptionsBroker) PublishPlanChange(plan *model.AdminBlossomPlan) {
	if plan == nil {
		return
	}
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.planSubs {
		select {
		case ch <- plan:
		default:
			// Non-blocking drop to enforce backpressure limit
		}
	}
}

func (b *SubscriptionsBroker) PublishObjectChange(obj *model.AdminBlossomObject) {
	if obj == nil {
		return
	}
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.objectSubs {
		select {
		case ch <- obj:
		default:
			// Non-blocking drop to enforce backpressure limit
		}
	}
}
