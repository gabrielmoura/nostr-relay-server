package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	goredis "github.com/redis/go-redis/v9"
)

func SetQueryResult(filterHash string, val string) error {
	if !IsEnabled() {
		return nil
	}
	version, err := GetQueryVersion()
	if err != nil {
		return err
	}
	ctx, cancel := cacheContext()
	defer cancel()
	queryKey := queryCacheKey(version, filterHash)
	ttl := ttlOr(config.Cfg.Redis.Cache.QueryTTL, 30*time.Second)
	if err := redisClient.Set(ctx, queryKey, val, ttl); err != nil {
		return err
	}
	return setQueryMeta(ctx, version, filterHash, ttl, false)
}

func GetQueryResult(filterHash string) (string, bool) {
	if !IsEnabled() {
		return "", false
	}
	version, err := GetQueryVersion()
	if err != nil {
		return "", false
	}
	ctx, cancel := cacheContext()
	defer cancel()

	val, err := redisClient.Get(ctx, queryCacheKey(version, filterHash))
	if err != nil {
		_ = setQueryMeta(ctx, version, filterHash, queryMetaTTL(), false)
		return "", false
	}
	_ = setQueryMeta(ctx, version, filterHash, queryMetaTTL(), true)
	return val, true
}

func InvalidateQueryCache() error {
	if !IsEnabled() {
		return nil
	}
	ctx, cancel := cacheContext()
	defer cancel()
	_, err := redisClient.Incr(ctx, queryVersionKey)
	return err
}

func GetQueryVersion() (int64, error) {
	if !IsEnabled() {
		return 0, nil
	}
	ctx, cancel := cacheContext()
	defer cancel()

	val, err := redisClient.Get(ctx, queryVersionKey)
	if err == nil {
		return strconv.ParseInt(val, 10, 64)
	}
	if err != goredis.Nil {
		return 0, err
	}
	if err := redisClient.Set(ctx, queryVersionKey, "1", 0); err != nil {
		return 0, err
	}
	return 1, nil
}

func QueryCacheHit(filterHash string) {
	recordQueryMeta(filterHash, true)
}

func QueryCacheMiss(filterHash string) {
	recordQueryMeta(filterHash, false)
}

func recordQueryMeta(filterHash string, hit bool) {
	if !IsEnabled() {
		return
	}
	version, err := GetQueryVersion()
	if err != nil {
		return
	}
	ctx, cancel := cacheContext()
	defer cancel()
	_ = setQueryMeta(ctx, version, filterHash, queryMetaTTL(), hit)
}

func queryCacheKey(version int64, filterHash string) string {
	return fmt.Sprintf("query:v%d:%s", version, filterHash)
}

func queryMetaKey(version int64, filterHash string) string {
	return fmt.Sprintf("query:meta:v%d:%s", version, filterHash)
}

func queryMetaTTL() time.Duration {
	return ttlOr(config.Cfg.Redis.Cache.QueryMetaTTL, 30*time.Second)
}

func setQueryMeta(ctx context.Context, version int64, filterHash string, ttl time.Duration, hit bool) error {
	result := "miss"
	if hit {
		result = "hit"
	}
	if err := redisClient.HSet(ctx, queryMetaKey(version, filterHash), "last_result", result, "last_access_unix", strconv.FormatInt(time.Now().Unix(), 10), "filter_hash", filterHash); err != nil {
		return err
	}
	return redisClient.Expire(ctx, queryMetaKey(version, filterHash), ttl)
}

func MarshalQueryPayload(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
