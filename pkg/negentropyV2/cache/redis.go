package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/model"
)

type redisKV interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

type RedisOptions struct {
	Prefix    string
	Timeout   time.Duration
	KeyPrefix string
}

type RedisQueryCache struct {
	client    redisKV
	prefix    string
	timeout   time.Duration
	keyPrefix string
}

func NewRedisQueryCache(client redisKV, opts RedisOptions) *RedisQueryCache {
	prefix := opts.Prefix
	if prefix == "" {
		prefix = "negentropy:v2"
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 150 * time.Millisecond
	}

	keyPrefix := opts.KeyPrefix
	if keyPrefix == "" {
		keyPrefix = "query"
	}

	return &RedisQueryCache{
		client:    client,
		prefix:    prefix,
		timeout:   timeout,
		keyPrefix: keyPrefix,
	}
}

func (c *RedisQueryCache) Get(key string) ([]model.EventRef, bool) {
	if c == nil || c.client == nil || key == "" {
		return nil, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	raw, err := c.client.Get(ctx, c.cacheKey(key))
	if err != nil || raw == "" {
		return nil, false
	}

	refs := make([]model.EventRef, 0)
	if err := json.Unmarshal([]byte(raw), &refs); err != nil {
		return nil, false
	}

	out := make([]model.EventRef, len(refs))
	copy(out, refs)

	return out, true
}

func (c *RedisQueryCache) Set(key string, refs []model.EventRef, ttl time.Duration) {
	if c == nil || c.client == nil || key == "" || ttl <= 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	payload, err := json.Marshal(refs)
	if err != nil {
		return
	}

	_ = c.client.Set(ctx, c.cacheKey(key), payload, ttl)
}

func (c *RedisQueryCache) Delete(key string) {
	if c == nil || c.client == nil || key == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	_ = c.client.Del(ctx, c.cacheKey(key))
}

func (c *RedisQueryCache) PurgeExpired(_ time.Time) {}

func (c *RedisQueryCache) cacheKey(key string) string {
	return c.prefix + ":" + c.keyPrefix + ":" + key
}
