package conf

import (
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/stretchr/testify/require"
)

func TestValidateCronSchedulesDailyStatsRequiresRedis(t *testing.T) {
	cfg := &config.Config{
		Cron: config.CronConfig{
			DailyStats: config.CronDailyStatsConfig{
				Enabled:        true,
				Schedule:       "0 10 0 * * *",
				RedisNamespace: "daily_stats",
				TTLDays:        400,
				LockTTLSeconds: 1860,
			},
		},
	}

	err := validateCronSchedules(cfg)

	require.EqualError(t, err, "cron.daily_stats is enabled but redis.enabled is false")
}

func TestValidateCronSchedulesDailyStatsAcceptsValidConfig(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{Enabled: true},
		Cron: config.CronConfig{
			DailyStats: config.CronDailyStatsConfig{
				Enabled:        true,
				Schedule:       "0 10 0 * * *",
				RedisNamespace: "daily_stats",
				TTLDays:        400,
				LockTTLSeconds: 1860,
			},
		},
	}

	require.NoError(t, validateCronSchedules(cfg))
}
