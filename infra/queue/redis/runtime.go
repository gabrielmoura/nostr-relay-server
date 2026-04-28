package redisqueue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	iredis "github.com/gabrielmoura/nostr-relay-server/infra/redis"
	"github.com/gabrielmoura/nostr-relay-server/internal/jobs"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Runtime struct {
	client     *iredis.Client
	registry   *jobs.MemoryRegistry
	config     RuntimeConfig
	scripts    *Scripts
	dispatcher *Dispatcher
	tracker    *Tracker
	workers    []*Worker
	once       sync.Once
}

func NewRuntime(client *iredis.Client, redisCfg config.RedisQueueConfig, jobsCfg config.JobsConfig) (*Runtime, error) {
	if client == nil || !client.IsEnabled() {
		return nil, errQueueDisabled
	}

	runtimeConfig, err := NewRuntimeConfig(redisCfg, jobsCfg)
	if err != nil {
		return nil, err
	}
	scripts := NewScripts()
	if err := scripts.Load(context.Background(), client.Raw()); err != nil {
		return nil, err
	}

	registry := jobs.NewRegistry()
	runtime := &Runtime{
		client:     client,
		registry:   registry,
		config:     runtimeConfig,
		scripts:    scripts,
		dispatcher: NewDispatcher(client, runtimeConfig, scripts),
		tracker:    NewTracker(client, runtimeConfig, scripts),
	}

	return runtime, nil
}

func (r *Runtime) Service() *jobs.Service {
	return &jobs.Service{
		Dispatcher: r.dispatcher,
		Monitor:    r.tracker,
		Registry:   r.registry,
	}
}

func (r *Runtime) Registry() *jobs.MemoryRegistry {
	return r.registry
}

func (r *Runtime) Start(ctx context.Context) error {
	var startErr error
	r.once.Do(func() {
		workerCount := r.config.WorkerCount
		if workerCount <= 0 {
			return
		}

		for i := 0; i < workerCount; i++ {
			workerName := fmt.Sprintf("%s-%d", r.config.ConsumerGroup, i+1)
			worker := NewWorker(workerName, r.client, r.registry, r.dispatcher, r.tracker, r.config, r.scripts)
			r.workers = append(r.workers, worker)
			if err := worker.Prepare(ctx); err != nil {
				startErr = err
				return
			}
			go worker.Run(ctx)
		}

		go r.runPromoter(ctx)
		go r.runDepthSampler(ctx)
	})

	return startErr
}

func (r *Runtime) runPromoter(ctx context.Context) {
	ticker := time.NewTicker(r.config.PromoteInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			queues, _, err := r.client.Raw().Scan(ctx, 0, "rq:*:delayed", 100).Result()
			if err != nil && !errors.Is(err, goredis.Nil) {
				continue
			}
			for _, delayedKey := range queues {
				queueName := extractQueueName(delayedKey)
				if queueName == "" {
					continue
				}
				r.promoteQueue(ctx, queueName)
			}
		}
	}
}

func (r *Runtime) promoteQueue(ctx context.Context, queue string) {
	keys := NewKeys(queue)
	now := time.Now().UTC()
	_, err := r.scripts.promoteDelayed.Run(ctx, r.client.Raw(), []string{
		keys.Delayed(),
		keys.Stream(jobs.PriorityHigh),
		keys.Stream(jobs.PriorityNormal),
		keys.Stream(jobs.PriorityLow),
		keys.State(),
		keys.MetricsBucket(now),
		keys.MetaPrefix(),
	}, now.UnixMilli(), r.config.PromoteBatchSize, r.config.MaxLenApprox, int(r.config.MetricsTTL.Seconds())).Result()
	if err != nil {
		metrics.NostrQueueLuaErrorsTotal.WithLabelValues(queue, "promote_delayed").Inc()
		log.Logger.Warn("queue delayed promotion failed", zap.String("queue", queue), zap.Error(err))
	}
}

func (r *Runtime) runDepthSampler(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, queueName := range []string{config.Cfg.Jobs.Download.Queue, config.Cfg.Jobs.Sync.Queue, config.Cfg.Jobs.Cron.Queue, r.config.DefaultQueue} {
				if queueName == "" {
					continue
				}
				keys := NewKeys(queueName)
				for _, priority := range []jobs.Priority{jobs.PriorityHigh, jobs.PriorityNormal, jobs.PriorityLow} {
					length, err := r.client.Raw().XLen(ctx, keys.Stream(priority)).Result()
					if err == nil {
						metrics.NostrQueueDepth.WithLabelValues(queueName, string(priority)).Set(float64(length))
					}
				}
				delayed, err := r.client.Raw().ZCard(ctx, keys.Delayed()).Result()
				if err == nil {
					metrics.NostrQueueDelayedJobs.WithLabelValues(queueName).Set(float64(delayed))
				}
				dead, err := r.client.Raw().ZCard(ctx, keys.Dead()).Result()
				if err == nil {
					metrics.NostrQueueDeadJobs.WithLabelValues(queueName).Set(float64(dead))
				}
			}
		}
	}
}

func extractQueueName(key string) string {
	const prefix = "rq:{"
	const suffix = "}:delayed"
	if len(key) <= len(prefix)+len(suffix) {
		return ""
	}
	if key[:len(prefix)] != prefix || key[len(key)-len(suffix):] != suffix {
		return ""
	}
	return key[len(prefix) : len(key)-len(suffix)]
}
