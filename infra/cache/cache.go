package cache

import (
	"context"
	"strconv"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/redis"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	initialized bool
	redisClient *redis.Client
)

type UserBanned struct {
	Reason string `json:"r"`
	Banned bool   `json:"b"`
}

type GetUserBannedByKey func(ctx context.Context, key string) (reason string, exists bool, err error)

func Init() error {
	redisClient = redis.GetClient()
	if redisClient != nil && redisClient.IsEnabled() {
		initialized = true
		log.Logger.Info("Redis cache initialized")
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
	if !IsEnabled() {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	return redisClient.Set(ctx, key, value, 0)
}

func SetWithTTL(key string, value string, ttl time.Duration) error {
	if !IsEnabled() {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	return redisClient.Set(ctx, key, value, ttl)
}

func Get(key string) (string, error) {
	if !IsEnabled() {
		return "", goredis.Nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	return redisClient.Get(ctx, key)
}

func Delete(key string) error {
	if !IsEnabled() {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	return redisClient.Del(ctx, key)
}

func SetNX(key string, value string, ttl time.Duration) (bool, error) {
	if !IsEnabled() {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	return redisClient.SetNX(ctx, key, value, ttl)
}

func CheckSpam(key string, threshold int) (bool, error) {
	spamKey := key + "_spam"

	if !IsEnabled() {
		return false, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	count, err := redisClient.Incr(ctx, spamKey)
	if err != nil {
		return false, err
	}

	if count == 1 {
		redisClient.Expire(ctx, spamKey, 5*time.Minute)
	}

	return count >= int64(threshold), nil
}

func GetBanned(pubKey string) (reason string, banned bool, found bool) {
	if !IsEnabled() {
		return "", false, false
	}

	bannedKey := "ban:" + pubKey
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	rawJSON, err := redisClient.Get(ctx, bannedKey)
	if err != nil {
		return "", false, false
	}

	var userStatus UserBanned
	if err := json.Unmarshal([]byte(rawJSON), &userStatus); err != nil {
		return "", false, false
	}

	return userStatus.Reason, userStatus.Banned, true
}

func SetBanned(pubKey string, val *UserBanned) error {
	if !IsEnabled() {
		return nil
	}

	rawJSON, err := json.Marshal(val)
	if err != nil {
		return err
	}

	bannedKey := "ban:" + pubKey
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ttl := time.Duration(config.Cfg.Redis.Cache.BanTTL) * time.Second
	if ttl == 0 {
		ttl = time.Hour
	}

	return redisClient.Set(ctx, bannedKey, string(rawJSON), ttl)
}

func SetProfile(pubKey string, val *ProfileCache) error {
	if !IsEnabled() {
		return nil
	}

	rawJSON, err := json.Marshal(val)
	if err != nil {
		return err
	}

	profileKey := "profile:" + pubKey
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ttl := time.Duration(config.Cfg.Redis.Cache.ProfileTTL) * time.Second
	if ttl == 0 {
		ttl = 5 * time.Minute
	}

	return redisClient.Set(ctx, profileKey, string(rawJSON), ttl)
}

func GetProfile(pubKey string) (*ProfileCache, bool) {
	if !IsEnabled() {
		return nil, false
	}

	profileKey := "profile:" + pubKey
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	rawJSON, err := redisClient.Get(ctx, profileKey)
	if err != nil {
		return nil, false
	}

	var profile ProfileCache
	if err := json.Unmarshal([]byte(rawJSON), &profile); err != nil {
		return nil, false
	}

	return &profile, true
}

func SetEvent(eventID string, val string) error {
	if !IsEnabled() {
		return nil
	}

	eventKey := "event:" + eventID
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ttl := time.Duration(config.Cfg.Redis.Cache.EventTTL) * time.Second
	if ttl == 0 {
		ttl = 10 * time.Minute
	}

	return redisClient.Set(ctx, eventKey, val, ttl)
}

func GetEvent(eventID string) (string, bool) {
	if !IsEnabled() {
		return "", false
	}

	eventKey := "event:" + eventID
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	val, err := redisClient.Get(ctx, eventKey)
	if err != nil {
		return "", false
	}

	return val, true
}

func SetDedup(eventID string) (bool, error) {
	if !IsEnabled() {
		return false, nil
	}

	dedupKey := "dedup:" + eventID
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ttl := time.Duration(config.Cfg.Redis.Cache.DedupTTL) * time.Second
	if ttl == 0 {
		ttl = time.Hour
	}

	set, err := redisClient.SetNX(ctx, dedupKey, "1", ttl)
	return !set, err
}

func SetQueryResult(filterHash string, val string) error {
	if !IsEnabled() {
		return nil
	}

	queryKey := "query:" + filterHash
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ttl := time.Duration(config.Cfg.Redis.Cache.QueryTTL) * time.Second
	if ttl == 0 {
		ttl = 30 * time.Second
	}

	return redisClient.Set(ctx, queryKey, val, ttl)
}

func GetQueryResult(filterHash string) (string, bool) {
	if !IsEnabled() {
		return "", false
	}

	queryKey := "query:" + filterHash
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	val, err := redisClient.Get(ctx, queryKey)
	if err != nil {
		return "", false
	}

	return val, true
}

func InvalidateQueryCache() error {
	if !IsEnabled() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cursor uint64
	for {
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, "query:*", 100)
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			if err := redisClient.Del(ctx, keys...); err != nil {
				log.Logger.Warn("failed to delete query cache keys", zap.Error(err))
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

type ProfileCache struct {
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	About       string `json:"about,omitempty"`
	Picture     string `json:"picture,omitempty"`
	Website     string `json:"website,omitempty"`
	NIP05       string `json:"nip05,omitempty"`
	LUD16       string `json:"lud16,omitempty"`
	Bot         bool   `json:"bot,omitempty"`
}

func WrapGetBanned(internalLookup GetUserBannedByKey) GetUserBannedByKey {
	return func(ctx context.Context, key string) (reason string, exists bool, err error) {
		if !IsEnabled() {
			return internalLookup(ctx, key)
		}

		bannedKey := "ban:" + key

		cachedReason, isBanned, foundInCache := GetBanned(key)
		if foundInCache {
			if isBanned {
				return cachedReason, true, nil
			}
			return "", false, nil
		}

		reason, exists, err = internalLookup(ctx, key)
		if err != nil {
			return "", false, err
		}

		if err := SetBanned(key, &UserBanned{Reason: reason, Banned: exists}); err != nil {
			log.Logger.Debug("failed to cache ban status", zap.String("key", bannedKey), zap.Error(err))
		}

		return reason, exists, nil
	}
}

func IncrCounter(key string) (int64, error) {
	if !IsEnabled() {
		return 0, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	return redisClient.Incr(ctx, key)
}

func IncrCounterWithExpiry(key string, expiry time.Duration) (int64, error) {
	if !IsEnabled() {
		return 0, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	count, err := redisClient.Incr(ctx, key)
	if err != nil {
		return count, err
	}

	if count == 1 {
		redisClient.Expire(ctx, key, expiry)
	}

	return count, nil
}

func GetCounter(key string) (int64, error) {
	if !IsEnabled() {
		return 0, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	val, err := redisClient.Get(ctx, key)
	if err != nil {
		return 0, err
	}

	return strconv.ParseInt(val, 10, 64)
}
