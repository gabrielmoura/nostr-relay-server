package redis

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Client struct {
	rdb     *goredis.Client
	cfg     *config.RedisConfig
	enabled bool
	mu      sync.RWMutex
}

var (
	client     *Client
	clientOnce sync.Once
)

func New(cfg *config.RedisConfig) *Client {
	if !cfg.Enabled {
		return &Client{enabled: false}
	}

	addr := cfg.Addr
	if addr == "" {
		addr = "127.0.0.1:6379"
	}

	rdb := goredis.NewClient(&goredis.Options{
		Addr:         addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		PoolTimeout:  time.Second,
		MaxRetries:   0,
	})

	return &Client{
		rdb:     rdb,
		cfg:     cfg,
		enabled: true,
	}
}

func Init(cfg *config.RedisConfig) error {
	clientOnce.Do(func() {
		client = New(cfg)
		if !client.IsEnabled() {
			log.Logger.Info("Redis is disabled, using fallback")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel()

		if err := client.Ping(ctx); err != nil {
			var dnsErr *net.DNSError
			if errors.As(err, &dnsErr) {
				log.Logger.Info("Redis DNS lookup failed, using in-memory fallback", zap.String("addr", client.cfg.Addr), zap.Error(err))
			} else {
				log.Logger.Info("Redis connection failed, using in-memory fallback", zap.String("addr", client.cfg.Addr), zap.Error(err))
			}
			_ = client.Close()
			client.enabled = false
			return
		}

		log.Logger.Info("Redis connected successfully", zap.String("addr", client.cfg.Addr))
	})
	return nil
}

func GetClient() *Client {
	return client
}

func (c *Client) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Client) Close() error {
	if c.rdb != nil {
		return c.rdb.Close()
	}
	return nil
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if !c.IsEnabled() {
		return "", goredis.Nil
	}
	return c.rdb.Get(ctx, key).Result()
}

func (c *Client) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if !c.IsEnabled() {
		return nil
	}
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

func (c *Client) SetNX(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	if !c.IsEnabled() {
		return false, nil
	}
	return c.rdb.SetNX(ctx, key, value, ttl).Result()
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	if !c.IsEnabled() || len(keys) == 0 {
		return nil
	}
	return c.rdb.Del(ctx, keys...).Err()
}

func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	if !c.IsEnabled() || len(keys) == 0 {
		return 0, nil
	}
	return c.rdb.Exists(ctx, keys...).Result()
}

func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	if !c.IsEnabled() {
		return "", goredis.Nil
	}
	return c.rdb.HGet(ctx, key, field).Result()
}

func (c *Client) HSet(ctx context.Context, key string, values ...any) error {
	if !c.IsEnabled() {
		return nil
	}
	return c.rdb.HSet(ctx, key, values...).Err()
}

func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	if !c.IsEnabled() {
		return nil, goredis.Nil
	}
	return c.rdb.HGetAll(ctx, key).Result()
}

func (c *Client) HDel(ctx context.Context, key string, fields ...string) error {
	if !c.IsEnabled() {
		return nil
	}
	return c.rdb.HDel(ctx, key, fields...).Err()
}

func (c *Client) HLen(ctx context.Context, key string) (int64, error) {
	if !c.IsEnabled() {
		return 0, nil
	}
	return c.rdb.HLen(ctx, key).Result()
}

func (c *Client) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if !c.IsEnabled() {
		return nil
	}
	return c.rdb.Expire(ctx, key, ttl).Err()
}

func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	if !c.IsEnabled() {
		return 0, goredis.Nil
	}
	return c.rdb.TTL(ctx, key).Result()
}

func (c *Client) Publish(ctx context.Context, channel string, message any) error {
	if !c.IsEnabled() {
		return nil
	}
	return c.rdb.Publish(ctx, channel, message).Err()
}

func (c *Client) Subscribe(ctx context.Context, channels ...string) *goredis.PubSub {
	if !c.IsEnabled() {
		return nil
	}
	return c.rdb.Subscribe(ctx, channels...)
}

func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	if !c.IsEnabled() {
		return 0, nil
	}
	return c.rdb.Incr(ctx, key).Result()
}

func (c *Client) Decr(ctx context.Context, key string) (int64, error) {
	if !c.IsEnabled() {
		return 0, nil
	}
	return c.rdb.Decr(ctx, key).Result()
}

func (c *Client) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	if !c.IsEnabled() {
		return 0, nil
	}
	return c.rdb.IncrBy(ctx, key, value).Result()
}

func (c *Client) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	if !c.IsEnabled() {
		return nil, 0, nil
	}
	return c.rdb.Scan(ctx, cursor, match, count).Result()
}

func (c *Client) Raw() *goredis.Client {
	return c.rdb
}
