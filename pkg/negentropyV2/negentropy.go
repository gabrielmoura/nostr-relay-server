package negentropy

import (
	"context"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/cache"
	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/contracts"
	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/engine"
	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/model"
	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/service"
)

type (
	Manager = service.Manager
	Options = service.Options

	EventStore = contracts.EventStore
	QueryCache = contracts.QueryCache

	OpenRequest    = model.OpenRequest
	MessageRequest = model.MessageRequest
	Response       = model.Response
	ResponseType   = model.ResponseType
	Filter         = model.Filter
	EventRef       = model.EventRef
	EventID        = model.EventID

	EngineOptions = engine.Options
	Reconciler    = engine.Reconciler
	Diff          = engine.Diff
)

const (
	ResponseTypeMessage = model.ResponseTypeMessage
	ResponseTypeError   = model.ResponseTypeError
	ResponseTypeClosed  = model.ResponseTypeClosed
)

var ParseEventIDHex = model.ParseEventIDHex

func NewManager(store EventStore, queryCache QueryCache, opts Options) *Manager {
	return service.NewManager(store, queryCache, opts)
}

func NewMemoryQueryCache() *cache.MemoryQueryCache {
	return cache.NewMemoryQueryCache()
}

func NewRedisQueryCache(client interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}, opts cache.RedisOptions) *cache.RedisQueryCache {
	return cache.NewRedisQueryCache(client, opts)
}

func NewReconciler(refs []EventRef, opts EngineOptions) (*Reconciler, error) {
	return engine.NewReconciler(refs, opts)
}
