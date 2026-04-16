package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/model"
)

type fakeRedisKV struct {
	mu      sync.Mutex
	entries map[string]string
}

func newFakeRedisKV() *fakeRedisKV {
	return &fakeRedisKV{entries: map[string]string{}}
}

func (f *fakeRedisKV) Get(_ context.Context, key string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	v, ok := f.entries[key]
	if !ok {
		return "", context.Canceled
	}

	return v, nil
}

func (f *fakeRedisKV) Set(_ context.Context, key string, value any, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	bytes, _ := value.([]byte)
	f.entries[key] = string(bytes)

	return nil
}

func (f *fakeRedisKV) Del(_ context.Context, keys ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, key := range keys {
		delete(f.entries, key)
	}

	return nil
}

func TestRedisQueryCacheSetGetDelete(t *testing.T) {
	kv := newFakeRedisKV()
	cache := NewRedisQueryCache(kv, RedisOptions{Prefix: "test", Timeout: time.Second})

	refs := []model.EventRef{{CreatedAt: 1}}
	cache.Set("f:a", refs, time.Second)

	got, ok := cache.Get("f:a")
	if !ok {
		t.Fatalf("expected cache hit")
	}

	if len(got) != 1 || got[0].CreatedAt != 1 {
		t.Fatalf("unexpected refs: %+v", got)
	}

	cache.Delete("f:a")
	if _, ok := cache.Get("f:a"); ok {
		t.Fatalf("expected cache miss after delete")
	}
}
