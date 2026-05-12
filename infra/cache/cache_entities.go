package cache

import (
	"context"
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
