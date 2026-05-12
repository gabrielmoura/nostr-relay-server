package http

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/gabrielmoura/nostr-relay-server/infra/db/helper"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

func WarmAdminSearchCache(ctx context.Context) error {
	if !cache.IsEnabled() {
		return nil
	}

	jobs := []adminWarmupJob{
		{Name: "search:first_page", Run: func() error { _, err := loadSearchEventsResponse(ctx, nostr.Filter{Limit: 50}, 0); return err }},
		{Name: "search:aggregates", Run: func() error { _, err := loadSearchEventsAggregatesResponse(ctx, nostr.Filter{}); return err }},
		{Name: "search:timeline:day", Run: func() error { _, err := loadSearchEventsTimelineResponse(ctx, nostr.Filter{}, "day"); return err }},
		{Name: "search:timeline:hour", Run: func() error { _, err := loadSearchEventsTimelineResponse(ctx, nostr.Filter{}, "hour"); return err }},
	}

	for _, job := range jobs {
		if err := job.Run(); err != nil {
			return fmt.Errorf("warm admin cache %s: %w", job.Name, err)
		}
	}

	return nil
}

func loadAdminCachedPayload[T any](cacheKey string, build func() (T, error)) (T, error) {
	var zero T

	if !cache.IsEnabled() {
		return build()
	}

	if raw, ok := cache.GetQueryResult(cacheKey); ok {
		cache.QueryCacheHit(cacheKey)
		var payload T
		if err := json.Unmarshal([]byte(raw), &payload); err == nil {
			return payload, nil
		}
	}

	cache.QueryCacheMiss(cacheKey)
	payload, err := build()
	if err != nil {
		return zero, err
	}

	if raw, err := json.Marshal(payload); err == nil {
		_ = cache.SetQueryResult(cacheKey, string(raw))
	}

	return payload, nil
}

func adminSearchPageCacheKey(filter nostr.Filter, offset int) string {
	filter = helper.NormalizeFilter(&config.Cfg.Relay, filter)
	return adminSearchCacheHash("page", struct {
		Filter nostr.Filter `json:"filter"`
		Offset int          `json:"offset"`
	}{Filter: filter, Offset: offset})
}

func adminSearchAggregatesCacheKey(filter nostr.Filter) string {
	filter = helper.NormalizeFilter(&config.Cfg.Relay, filter)
	filter.Limit = 0
	return adminSearchCacheHash("aggregates", filter)
}

func adminSearchTimelineCacheKey(filter nostr.Filter, bucket string) string {
	filter = helper.NormalizeFilter(&config.Cfg.Relay, filter)
	filter.Limit = 0
	return adminSearchCacheHash("timeline", struct {
		Filter nostr.Filter `json:"filter"`
		Bucket string       `json:"bucket"`
	}{Filter: filter, Bucket: normalizeTimelineBucket(bucket)})
}

func adminSearchCacheHash(namespace string, payload any) string {
	raw, err := json.Marshal(struct {
		Namespace string `json:"namespace"`
		Payload   any    `json:"payload"`
	}{Namespace: namespace, Payload: payload})
	if err != nil {
		return namespace
	}
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("admin:%s:%x", namespace, sum[:])
}

func normalizeTimelineBucket(bucket string) string {
	if strings.EqualFold(strings.TrimSpace(bucket), "day") {
		return "day"
	}
	return "hour"
}

func adminQueryWarmupTimeout() time.Duration {
	if config.Cfg.Redis.Cache.QueryTTL <= 0 {
		return 15 * time.Second
	}
	return 15 * time.Second
}

func logAdminWarmupFailure(err error) {
	if err == nil {
		return
	}
	log.Logger.Warn("admin search cache warmup failed", zap.Error(err))
}
