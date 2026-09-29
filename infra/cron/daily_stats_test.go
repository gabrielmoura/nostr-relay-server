package cron

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	infraDb "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRunDailyStatsPublishesThreeClosedUTCDays(t *testing.T) {
	log.Logger = zap.NewNop()
	collector := &fakeDailyStatsCollector{}
	client := &fakeDailyStatsRedis{values: map[string]string{}}
	now := time.Date(2026, time.September, 27, 0, 10, 0, 0, time.UTC)
	closedToday := now.Truncate(24 * time.Hour)

	err := runDailyStats(context.Background(), collector, client, "stats", 400*24*time.Hour, 31*time.Minute, now)

	require.NoError(t, err)
	require.Equal(t, 31*time.Minute, client.lockTTL)
	require.Equal(t, [][2]int64{
		{closedToday.AddDate(0, 0, -3).Unix(), closedToday.AddDate(0, 0, -2).Unix()},
		{closedToday.AddDate(0, 0, -2).Unix(), closedToday.AddDate(0, 0, -1).Unix()},
		{closedToday.AddDate(0, 0, -1).Unix(), closedToday.Unix()},
	}, collector.windows)
	keys := newDailyStatsKeys("stats")
	require.Equal(t, "2026-09-26", client.values[keys.latest])

	var snapshot dailyStatsSnapshot
	require.NoError(t, json.Unmarshal([]byte(client.values[keys.daily("2026-09-26")]), &snapshot))
	require.Equal(t, dailyStatsSchemaVersion, snapshot.SchemaVersion)
	require.Equal(t, "2026-09-26", snapshot.Day)
	require.Equal(t, now, snapshot.GeneratedAt)
	require.Equal(t, []infraDb.DailyStatsKind{}, snapshot.TopKinds)
	require.Equal(t, []infraDb.DailyStatsAuthor{}, snapshot.TopAuthors)
	require.Equal(t, []infraDb.DailyStatsTag{}, snapshot.TopTags)
}

func TestDailyStatsLatestNeverMovesBackwards(t *testing.T) {
	client := &fakeDailyStatsRedis{values: map[string]string{}}
	collector := &fakeDailyStatsCollector{}
	keys := newDailyStatsKeys("stats")
	generatedAt := time.Date(2026, time.September, 27, 0, 10, 0, 0, time.UTC)

	require.NoError(t, collectAndPublishDailyStats(context.Background(), collector, client, keys, time.Hour, generatedAt.AddDate(0, 0, -1), generatedAt, generatedAt))
	require.NoError(t, collectAndPublishDailyStats(context.Background(), collector, client, keys, time.Hour, generatedAt.AddDate(0, 0, -2), generatedAt.AddDate(0, 0, -1), generatedAt))

	require.Equal(t, "2026-09-26", client.values[keys.latest])
}

func TestDailyStatsLockReleaseRequiresMatchingToken(t *testing.T) {
	client := &fakeDailyStatsRedis{values: map[string]string{"lock": "new-owner"}}

	_, err := client.Eval(context.Background(), dailyStatsUnlockLua, []string{"lock"}, "old-owner")

	require.NoError(t, err)
	require.Equal(t, "new-owner", client.values["lock"])
	require.Contains(t, dailyStatsUnlockLua, "redis.call('GET', KEYS[1]) == ARGV[1]")
	require.Contains(t, dailyStatsUnlockLua, "redis.call('DEL', KEYS[1])")
}

func TestDailyStatsKeysUseOneRedisHashSlot(t *testing.T) {
	keys := newDailyStatsKeys("stats")

	require.Equal(t, "daily_stats:{stats}:lock", keys.lock)
	require.Equal(t, "daily_stats:{stats}:latest", keys.latest)
	require.Equal(t, "daily_stats:{stats}:day:2026-09-26", keys.daily("2026-09-26"))
}

func TestValidateDailyStatsOptions(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		ttl       time.Duration
		lockTTL   time.Duration
	}{
		{name: "valid", namespace: "stats", ttl: time.Hour, lockTTL: time.Minute},
		{name: "empty namespace", namespace: " ", ttl: time.Hour, lockTTL: time.Minute},
		{name: "invalid hash tag", namespace: "{stats}", ttl: time.Hour, lockTTL: time.Minute},
		{name: "zero ttl", namespace: "stats", lockTTL: time.Minute},
		{name: "zero lock ttl", namespace: "stats", ttl: time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDailyStatsOptions(tt.namespace, tt.ttl, tt.lockTTL)
			if tt.name == "valid" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
		})
	}
}

func TestValidateDailyStatsLockTTLCoversJobDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	require.Error(t, validateDailyStatsLockTTL(ctx, 30*time.Minute))
	require.NoError(t, validateDailyStatsLockTTL(ctx, 31*time.Minute))
}

type fakeDailyStatsCollector struct {
	windows [][2]int64
}

func (c *fakeDailyStatsCollector) CollectDailyStats(_ context.Context, start, end int64) (infraDb.DailyStatsAggregate, error) {
	c.windows = append(c.windows, [2]int64{start, end})
	return infraDb.DailyStatsAggregate{}, nil
}

type fakeDailyStatsRedis struct {
	values  map[string]string
	lockTTL time.Duration
}

func (c *fakeDailyStatsRedis) SetNX(_ context.Context, key string, value any, ttl time.Duration) (bool, error) {
	if _, exists := c.values[key]; exists {
		return false, nil
	}
	c.values[key] = value.(string)
	c.lockTTL = ttl
	return true, nil
}

func (c *fakeDailyStatsRedis) Eval(_ context.Context, script string, keys []string, args ...any) (any, error) {
	switch script {
	case dailyStatsUnlockLua:
		if c.values[keys[0]] == args[0].(string) {
			delete(c.values, keys[0])
			return int64(1), nil
		}
		return int64(0), nil
	case dailyStatsPublishLua:
		c.values[keys[0]] = args[0].(string)
		day := args[2].(string)
		if current := c.values[keys[1]]; current == "" || current < day {
			c.values[keys[1]] = day
		}
		return int64(1), nil
	default:
		return nil, nil
	}
}
