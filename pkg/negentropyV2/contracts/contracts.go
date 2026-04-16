package contracts

import (
	"context"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/model"
)

type EventStore interface {
	QueryEventRefs(ctx context.Context, filter model.Filter) ([]model.EventRef, error)
}

type QueryCache interface {
	Get(key string) ([]model.EventRef, bool)
	Set(key string, refs []model.EventRef, ttl time.Duration)
	Delete(key string)
	PurgeExpired(now time.Time)
}
