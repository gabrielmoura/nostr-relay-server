package cron

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	infraDb "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	redisinfra "github.com/gabrielmoura/nostr-relay-server/infra/redis"
	"go.uber.org/zap"
)

const (
	dailyStatsSchemaVersion = 1
	dailyStatsDaysToRefresh = 3
	dailyStatsUnlockTimeout = 2 * time.Second
)

const dailyStatsUnlockLua = `
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('DEL', KEYS[1])
end
return 0`

const dailyStatsPublishLua = `
redis.call('SET', KEYS[1], ARGV[1], 'EX', ARGV[2])
local current = redis.call('GET', KEYS[2])
if not current or current < ARGV[3] then
  redis.call('SET', KEYS[2], ARGV[3], 'EX', ARGV[2])
end
return 1`

type dailyStatsSnapshot struct {
	SchemaVersion      int                        `json:"schema_version"`
	Day                string                     `json:"day"`
	WindowStart        time.Time                  `json:"window_start"`
	WindowEnd          time.Time                  `json:"window_end"`
	GeneratedAt        time.Time                  `json:"generated_at"`
	TotalEvents        int64                      `json:"total_events"`
	UniqueAuthors      int64                      `json:"unique_authors"`
	UniqueKinds        int64                      `json:"unique_kinds"`
	EventsWithHashtags int64                      `json:"events_with_hashtags"`
	TopKinds           []infraDb.DailyStatsKind   `json:"top_kinds"`
	TopAuthors         []infraDb.DailyStatsAuthor `json:"top_authors"`
	TopTags            []infraDb.DailyStatsTag    `json:"top_tags"`
}

type dailyStatsCollector interface {
	CollectDailyStats(context.Context, int64, int64) (infraDb.DailyStatsAggregate, error)
}

type dailyStatsRedisClient interface {
	SetNX(context.Context, string, any, time.Duration) (bool, error)
	Eval(context.Context, string, []string, ...any) (any, error)
}

type dailyStatsRedisAdapter struct {
	client *redisinfra.Client
}

func (a dailyStatsRedisAdapter) SetNX(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	return a.client.SetNX(ctx, key, value, ttl)
}

func (a dailyStatsRedisAdapter) Eval(ctx context.Context, script string, keys []string, args ...any) (any, error) {
	return a.client.Raw().Eval(ctx, script, keys, args...).Result()
}

func RunDailyStats(ctx context.Context) error {
	cfg := config.Cfg.Cron.DailyStats
	if !cfg.Enabled {
		return nil
	}

	redisClient := redisinfra.GetClient()
	if redisClient == nil || !redisClient.IsEnabled() {
		return fmt.Errorf("daily stats requires an available Redis client")
	}

	return runDailyStats(
		ctx,
		infraDb.DbQueries(),
		dailyStatsRedisAdapter{client: redisClient},
		cfg.RedisNamespace,
		time.Duration(cfg.TTLDays)*24*time.Hour,
		time.Duration(cfg.LockTTLSeconds)*time.Second,
		time.Now().UTC(),
	)
}

func runDailyStats(
	ctx context.Context,
	collector dailyStatsCollector,
	client dailyStatsRedisClient,
	namespace string,
	ttl time.Duration,
	lockTTL time.Duration,
	now time.Time,
) error {
	if collector == nil {
		return fmt.Errorf("daily stats collector cannot be nil")
	}
	if client == nil {
		return fmt.Errorf("daily stats Redis client cannot be nil")
	}
	if err := validateDailyStatsOptions(namespace, ttl, lockTTL); err != nil {
		return err
	}

	start := time.Now()
	token, err := newDailyStatsLockToken()
	if err != nil {
		return fmt.Errorf("generate daily stats lock token: %w", err)
	}
	keys := newDailyStatsKeys(namespace)
	locked, err := client.SetNX(ctx, keys.lock, token, lockTTL)
	if err != nil {
		metrics.NostrCronDailyStatsRunsTotal.WithLabelValues("error").Inc()
		return fmt.Errorf("acquire daily stats lock: %w", err)
	}
	if !locked {
		metrics.NostrCronDailyStatsRunsTotal.WithLabelValues("skipped").Inc()
		log.Logger.Info("cron daily stats skipped: lock is held")
		return nil
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), dailyStatsUnlockTimeout)
		defer cancel()
		if _, unlockErr := client.Eval(unlockCtx, dailyStatsUnlockLua, []string{keys.lock}, token); unlockErr != nil {
			log.Logger.Warn("cron daily stats lock release failed", zap.Error(unlockErr))
		}
	}()

	closedToday := now.UTC().Truncate(24 * time.Hour)
	for offset := dailyStatsDaysToRefresh; offset >= 1; offset-- {
		windowStart := closedToday.AddDate(0, 0, -offset)
		windowEnd := windowStart.AddDate(0, 0, 1)
		if err := collectAndPublishDailyStats(ctx, collector, client, keys, ttl, windowStart, windowEnd, now.UTC()); err != nil {
			metrics.NostrCronDailyStatsRunsTotal.WithLabelValues("error").Inc()
			return err
		}
	}

	metrics.NostrCronDailyStatsRunsTotal.WithLabelValues("success").Inc()
	metrics.NostrCronDailyStatsDurationSeconds.Observe(time.Since(start).Seconds())
	log.Logger.Info("cron daily stats completed", zap.Int("days", dailyStatsDaysToRefresh), zap.Duration("duration", time.Since(start)))
	return nil
}

func collectAndPublishDailyStats(
	ctx context.Context,
	collector dailyStatsCollector,
	client dailyStatsRedisClient,
	keys dailyStatsKeys,
	ttl time.Duration,
	windowStart time.Time,
	windowEnd time.Time,
	generatedAt time.Time,
) error {
	aggregate, err := collector.CollectDailyStats(ctx, windowStart.Unix(), windowEnd.Unix())
	if err != nil {
		return fmt.Errorf("collect daily stats for %s: %w", windowStart.Format(time.DateOnly), err)
	}

	snapshot := dailyStatsSnapshot{
		SchemaVersion:      dailyStatsSchemaVersion,
		Day:                windowStart.Format(time.DateOnly),
		WindowStart:        windowStart.UTC(),
		WindowEnd:          windowEnd.UTC(),
		GeneratedAt:        generatedAt.UTC(),
		TotalEvents:        aggregate.TotalEvents,
		UniqueAuthors:      aggregate.UniqueAuthors,
		UniqueKinds:        aggregate.UniqueKinds,
		EventsWithHashtags: aggregate.EventsWithHashtags,
		TopKinds:           nonNilDailyStatsKinds(aggregate.TopKinds),
		TopAuthors:         nonNilDailyStatsAuthors(aggregate.TopAuthors),
		TopTags:            nonNilDailyStatsTags(aggregate.TopTags),
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("marshal daily stats for %s: %w", snapshot.Day, err)
	}
	if _, err := client.Eval(ctx, dailyStatsPublishLua, []string{keys.daily(snapshot.Day), keys.latest}, string(payload), int64(ttl.Seconds()), snapshot.Day); err != nil {
		return fmt.Errorf("publish daily stats for %s: %w", snapshot.Day, err)
	}
	return nil
}

func nonNilDailyStatsKinds(items []infraDb.DailyStatsKind) []infraDb.DailyStatsKind {
	if items == nil {
		return []infraDb.DailyStatsKind{}
	}
	return items
}

func nonNilDailyStatsAuthors(items []infraDb.DailyStatsAuthor) []infraDb.DailyStatsAuthor {
	if items == nil {
		return []infraDb.DailyStatsAuthor{}
	}
	return items
}

func nonNilDailyStatsTags(items []infraDb.DailyStatsTag) []infraDb.DailyStatsTag {
	if items == nil {
		return []infraDb.DailyStatsTag{}
	}
	return items
}

type dailyStatsKeys struct {
	prefix string
	lock   string
	latest string
}

func newDailyStatsKeys(namespace string) dailyStatsKeys {
	prefix := "daily_stats:{" + namespace + "}"
	return dailyStatsKeys{
		prefix: prefix,
		lock:   prefix + ":lock",
		latest: prefix + ":latest",
	}
}

func (k dailyStatsKeys) daily(day string) string {
	return k.prefix + ":day:" + day
}

func validateDailyStatsOptions(namespace string, ttl, lockTTL time.Duration) error {
	if strings.TrimSpace(namespace) == "" {
		return fmt.Errorf("daily stats Redis namespace cannot be empty")
	}
	if strings.ContainsAny(namespace, "{}") {
		return fmt.Errorf("daily stats Redis namespace cannot contain Redis hash tag delimiters")
	}
	if ttl <= 0 {
		return fmt.Errorf("daily stats TTL must be positive")
	}
	if lockTTL <= 0 {
		return fmt.Errorf("daily stats lock TTL must be positive")
	}
	return nil
}

func newDailyStatsLockToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
