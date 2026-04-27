package nip86

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
)

func cacheTTL() time.Duration {
	if config.Cfg == nil || config.Cfg.NIP86.CacheTTLSeconds <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(config.Cfg.NIP86.CacheTTLSeconds) * time.Second
}

func getCachedEntry(key string) (cacheEntry, bool) {
	raw, err := cache.Get(key)
	if err != nil {
		return cacheEntry{}, false
	}
	var entry cacheEntry
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		return cacheEntry{}, false
	}
	return entry, true
}

func setCachedEntry(key string, entry cacheEntry) {
	payload, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_ = cache.SetWithTTL(key, string(payload), cacheTTL())
}

func invalidateKeys(keys ...string) {
	for _, key := range keys {
		_ = cache.Delete(key)
	}
}

func (s *Service) IsIPBlocked(ctx context.Context, ip string) (string, bool, error) {
	if s == nil || !s.enabled {
		return "", false, nil
	}
	if entry, ok := getCachedEntry(blockIPCacheKey(ip)); ok {
		return entry.Reason, entry.State == cacheStatePresent, nil
	}
	record, exists, err := s.repo.GetBlockedIP(ctx, ip)
	if err != nil {
		return "", false, err
	}
	entry := cacheEntry{State: cacheStateMissing}
	if exists {
		entry.State = cacheStatePresent
		entry.Reason = record.Reason
	}
	setCachedEntry(blockIPCacheKey(ip), entry)
	return record.Reason, exists, nil
}

func (s *Service) IsEventBanned(ctx context.Context, eventID string) (string, bool, error) {
	if s == nil || !s.enabled {
		return "", false, nil
	}
	if entry, ok := getCachedEntry(bannedEventCacheKey(eventID)); ok {
		return entry.Reason, entry.State == cacheStatePresent, nil
	}
	record, exists, err := s.repo.GetBannedEvent(ctx, eventID)
	if err != nil {
		return "", false, err
	}
	entry := cacheEntry{State: cacheStateMissing}
	if exists {
		entry.State = cacheStatePresent
		entry.Reason = record.Reason
	}
	setCachedEntry(bannedEventCacheKey(eventID), entry)
	return record.Reason, exists, nil
}

func blockIPCacheKey(ip string) string {
	return "blockip:" + ip
}

func bannedEventCacheKey(eventID string) string {
	return "banevent:" + eventID
}

func allowedPubKeyCacheKey(pubkey string) string {
	return "allow:" + pubkey
}
