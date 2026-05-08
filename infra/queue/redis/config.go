package redisqueue

import (
	"fmt"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/internal/jobs"
)

type RuntimeConfig struct {
	ConsumerGroup              string
	WorkerCount                int
	BlockTimeout               time.Duration
	BatchSize                  int64
	MaxLenApprox               int64
	BodyTTL                    time.Duration
	ResultTTL                  time.Duration
	MetricsTTL                 time.Duration
	ReclaimIdle                time.Duration
	ReclaimBatchSize           int64
	PromoteBatchSize           int64
	PromoteInterval            time.Duration
	RecentJobsLimit            int64
	DefaultQueue               string
	DefaultTimeout             time.Duration
	DefaultMaxAttempts         uint8
	SyncMaxConcurrentPerRemote int
	RetryPolicy                jobs.RetryPolicy
	WorkerWindow               time.Duration
}

func NewRuntimeConfig(redisCfg config.RedisQueueConfig, jobsCfg config.JobsConfig) (RuntimeConfig, error) {
	if redisCfg.ConsumerGroup == "" {
		return RuntimeConfig{}, fmt.Errorf("redis.queue.consumer_group is required")
	}
	if jobsCfg.DefaultQueue == "" {
		return RuntimeConfig{}, fmt.Errorf("jobs.default_queue is required")
	}
	if jobsCfg.DefaultMaxAttempts <= 0 || jobsCfg.DefaultMaxAttempts > 255 {
		return RuntimeConfig{}, fmt.Errorf("jobs.default_max_attempts must be between 1 and 255")
	}

	return RuntimeConfig{
		ConsumerGroup:              redisCfg.ConsumerGroup,
		WorkerCount:                redisCfg.WorkerCount,
		BlockTimeout:               time.Duration(redisCfg.BlockMS) * time.Millisecond,
		BatchSize:                  maxInt64(redisCfg.BatchSize, 1),
		MaxLenApprox:               maxInt64(redisCfg.MaxLenApprox, 1),
		BodyTTL:                    time.Duration(maxInt(redisCfg.BodyTTLSeconds, 1)) * time.Second,
		ResultTTL:                  time.Duration(maxInt(redisCfg.ResultTTLSeconds, 1)) * time.Second,
		MetricsTTL:                 time.Duration(maxInt(redisCfg.MetricsTTLSeconds, 1)) * time.Second,
		ReclaimIdle:                time.Duration(maxInt(redisCfg.ReclaimIdleSeconds, 1)) * time.Second,
		ReclaimBatchSize:           maxInt64(redisCfg.ReclaimBatchSize, 1),
		PromoteBatchSize:           maxInt64(redisCfg.PromoteBatchSize, 1),
		PromoteInterval:            time.Duration(maxInt(redisCfg.PromoteIntervalMS, 1)) * time.Millisecond,
		RecentJobsLimit:            maxInt64(redisCfg.RecentJobsListLimit, 1),
		DefaultQueue:               jobsCfg.DefaultQueue,
		DefaultTimeout:             time.Duration(maxInt(jobsCfg.DefaultTimeoutSeconds, 1)) * time.Second,
		DefaultMaxAttempts:         uint8(jobsCfg.DefaultMaxAttempts),
		SyncMaxConcurrentPerRemote: maxInt(jobsCfg.Sync.MaxConcurrentPerRemote, 1),
		RetryPolicy: jobs.RetryPolicy{
			BaseDelay: time.Duration(maxInt(jobsCfg.RetryBaseDelayMS, 1)) * time.Millisecond,
			MaxDelay:  time.Duration(maxInt(jobsCfg.RetryMaxDelayMS, 1)) * time.Millisecond,
			Jitter:    time.Duration(maxInt(jobsCfg.RetryJitterMS, 0)) * time.Millisecond,
		},
		WorkerWindow: time.Duration(maxInt(jobsCfg.ActiveWorkerWindowSeconds, 1)) * time.Second,
	}, nil
}

func maxInt(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func maxInt64(value int64, fallback int64) int64 {
	if value > 0 {
		return value
	}
	return fallback
}
