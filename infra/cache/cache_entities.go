package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"go.uber.org/zap"
)

type UserBanned struct {
	Reason string `json:"r"`
	Banned bool   `json:"b"`
}

func FindAndAddContentSimhash(fingerprint uint64, ttl time.Duration) ([]uint64, error) {
	if !IsEnabled() {
		return []uint64{}, nil
	}
	ctx, cancel := cacheContext()
	defer cancel()
	raw := redisClient.Raw()
	values := map[uint64]struct{}{}
	for band := range 4 {
		key := fmt.Sprintf("simhash:%d:%04x", band, uint16(fingerprint>>(band*16)))
		members, err := raw.SMembers(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		for _, member := range members {
			if value, err := strconv.ParseUint(member, 16, 64); err == nil {
				values[value] = struct{}{}
			}
		}
		if err := raw.SAdd(ctx, key, fmt.Sprintf("%016x", fingerprint)).Err(); err != nil {
			return nil, err
		}
		if err := raw.Expire(ctx, key, ttl).Err(); err != nil {
			return nil, err
		}
	}
	result := make([]uint64, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	return result, nil
}

type GetUserBannedByKey func(ctx context.Context, key string) (reason string, exists bool, err error)

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

func GetBanned(pubKey string) (reason string, banned bool, found bool) {
	if !IsEnabled() {
		return "", false, false
	}
	rawJSON, err := Get("ban:" + pubKey)
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
	rawJSON, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return SetWithTTL("ban:"+pubKey, string(rawJSON), ttlOr(config.Cfg.Redis.Cache.BanTTL, time.Hour))
}

func SetProfile(pubKey string, val *ProfileCache) error {
	rawJSON, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return SetWithTTL("profile:"+pubKey, string(rawJSON), ttlOr(config.Cfg.Redis.Cache.ProfileTTL, 5*time.Minute))
}

func GetProfile(pubKey string) (*ProfileCache, bool) {
	rawJSON, err := Get("profile:" + pubKey)
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
	return SetWithTTL("event:"+eventID, val, ttlOr(config.Cfg.Redis.Cache.EventTTL, 10*time.Minute))
}

func GetEvent(eventID string) (string, bool) {
	val, err := Get("event:" + eventID)
	return val, err == nil
}

func SetDedup(eventID string) (bool, error) {
	set, err := SetNX("dedup:"+eventID, "1", ttlOr(config.Cfg.Redis.Cache.DedupTTL, time.Hour))
	return !set, err
}

func AddContentPubkey(hash string, pubkey string, ttl time.Duration) (int64, error) {
	if !IsEnabled() {
		return 0, nil
	}

	ctx, cancel := cacheContext()
	defer cancel()

	key := contentPubkeyKey(hash)
	pipeline := redisClient.Raw().TxPipeline()
	pipeline.SAdd(ctx, key, pubkey)
	pipeline.Expire(ctx, key, ttl)
	count := pipeline.SCard(ctx, key)
	if _, err := pipeline.Exec(ctx); err != nil {
		return 0, err
	}
	return count.Val(), nil
}

func ContentPubkeyCount(hash string) (int64, error) {
	if !IsEnabled() {
		return 0, nil
	}

	ctx, cancel := cacheContext()
	defer cancel()
	return redisClient.Raw().SCard(ctx, contentPubkeyKey(hash)).Result()
}

func GetContentAction(hash string) (string, bool) {
	value, err := Get(contentActionKey(hash))
	return value, err == nil
}

func SetContentAction(hash string, action string, ttl time.Duration) error {
	return SetWithTTL(contentActionKey(hash), action, ttl)
}

func contentPubkeyKey(hash string) string {
	return "content:" + hash + ":pubkeys"
}

func contentActionKey(hash string) string {
	return "content:" + hash + ":action"
}

func WrapGetBanned(internalLookup GetUserBannedByKey) GetUserBannedByKey {
	return func(ctx context.Context, key string) (reason string, exists bool, err error) {
		if !IsEnabled() {
			return internalLookup(ctx, key)
		}
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
			log.Logger.Debug("failed to cache ban status", zap.String("key", "ban:"+key), zap.Error(err))
		}
		return reason, exists, nil
	}
}
