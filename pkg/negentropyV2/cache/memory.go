package cache

import (
	"sync"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/model"
)

type entry struct {
	refs      []model.EventRef
	expiresAt time.Time
}

type MemoryQueryCache struct {
	mu      sync.RWMutex
	entries map[string]entry
}

func NewMemoryQueryCache() *MemoryQueryCache {
	return &MemoryQueryCache{
		entries: make(map[string]entry),
	}
}

func (c *MemoryQueryCache) Get(key string) ([]model.EventRef, bool) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		return nil, false
	}

	out := make([]model.EventRef, len(e.refs))
	copy(out, e.refs)

	return out, true
}

func (c *MemoryQueryCache) Set(key string, refs []model.EventRef, ttl time.Duration) {
	expiresAt := time.Time{}
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	cp := make([]model.EventRef, len(refs))
	copy(cp, refs)

	c.mu.Lock()
	c.entries[key] = entry{refs: cp, expiresAt: expiresAt}
	c.mu.Unlock()
}

func (c *MemoryQueryCache) Delete(key string) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

func (c *MemoryQueryCache) PurgeExpired(now time.Time) {
	c.mu.Lock()
	for key, item := range c.entries {
		if item.expiresAt.IsZero() {
			continue
		}

		if now.After(item.expiresAt) {
			delete(c.entries, key)
		}
	}
	c.mu.Unlock()
}
