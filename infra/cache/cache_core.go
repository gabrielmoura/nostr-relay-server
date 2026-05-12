package cache

import (
	"context"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/redis"
	goredis "github.com/redis/go-redis/v9"
)

var (
	initialized     bool
	redisClient     *redis.Client
	checkSpamScript *goredis.Script
)

const queryVersionKey = "query:version"

const checkSpamScriptSrc = `
local count = redis.call('INCR', KEYS[1])
if count == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return count
`

func Init() error {
	redisClient = redis.GetClient()
	if redisClient != nil && redisClient.IsEnabled() {
		initialized = true
		checkSpamScript = goredis.NewScript(checkSpamScriptSrc)
		log.Logger.Info("Redis cache initialized with Lua scripts")
	} else {
		log.Logger.Info("Redis cache disabled, using no-cache mode")
	}
	return nil
}

func GetRedis() *redis.Client {
	return redisClient
}

func IsEnabled() bool {
	return initialized && redisClient != nil && redisClient.IsEnabled()
}

func Set(key string, value string) error {
	return SetWithTTL(key, value, 0)
}

func SetWithTTL(key string, value string, ttl time.Duration) error {
	if !IsEnabled() {
		return nil
	}
	ctx, cancel := cacheContext()
	defer cancel()
	return redisClient.Set(ctx, key, value, ttl)
}

func Get(key string) (string, error) {
	if !IsEnabled() {
		return "", goredis.Nil
	}
	ctx, cancel := cacheContext()
	defer cancel()
	return redisClient.Get(ctx, key)
}

func Delete(key string) error {
	if !IsEnabled() {
		return nil
	}
	ctx, cancel := cacheContext()
	defer cancel()
	return redisClient.Del(ctx, key)
}

func SetNX(key string, value string, ttl time.Duration) (bool, error) {
	if !IsEnabled() {
		return false, nil
	}
	ctx, cancel := cacheContext()
	defer cancel()
	return redisClient.SetNX(ctx, key, value, ttl)
}

func cacheContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 100*time.Millisecond)
}

func ttlOr(seconds int, fallback time.Duration) time.Duration {
	ttl := time.Duration(seconds) * time.Second
	if ttl <= 0 {
		return fallback
	}
	return ttl
}
