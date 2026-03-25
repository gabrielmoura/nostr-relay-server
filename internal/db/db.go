package db

import (
	"context"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"go.uber.org/zap"
	"sync"
)

var (
	DbQueries *db.Queries
	Pool      *pgxpool.Pool

	addQueue = sync.Mutex{}
)

func Init(ctx context.Context) error {

	poolConfig, err := pgxpool.ParseConfig(config.Cfg.DB.PostgresURI)
	if err != nil {
		log.Logger.Error("Error parsing Postgres URI", zap.Error(err))
		return err
	}
	poolConfig.MaxConns = config.Cfg.DB.MaxConns
	poolConfig.MinConns = config.Cfg.DB.MinConns
	poolConfig.MaxConnLifetime = time.Duration(config.Cfg.DB.MaxConnLifetimeMinutes) * time.Minute
	poolConfig.MaxConnIdleTime = time.Duration(config.Cfg.DB.MaxConnIdleMinutes) * time.Minute
	poolConfig.HealthCheckPeriod = time.Duration(config.Cfg.DB.HealthCheckPeriodSeconds) * time.Second

	var logDbLevel tracelog.LogLevel
	if config.Cfg.AppEnv != "production" {
		logDbLevel = tracelog.LogLevelWarn
	} else {
		logDbLevel = tracelog.LogLevelError
	}

	poolConfig.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger: tracelog.LoggerFunc(func(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
			if level == tracelog.LogLevelError {
				log.Logger.Error(msg, zap.Any("data", data))
			} else if level == tracelog.LogLevelWarn {
				log.Logger.Warn(msg, zap.Any("data", data))
			} else {
				log.Logger.Info(msg, zap.Any("data", data))
			}
		}),
		LogLevel: logDbLevel,
	}

	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return PrepareConn(ctx, conn)
	}
	poolConfig.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		return conn.Ping(ctx) == nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return err
	}

	if err := checkConnection(ctx, pool); err != nil {
		log.Logger.Error("Error connecting to Postgres", zap.Error(err))
		pool.Close()
		return err
	}

	DbQueries = db.New(pool)
	Pool = pool
	cache.Init()
	go watchPoolStats(ctx, pool)
	return nil
}

func Close() {
}

// checkConnection checks if the connection to the database is working
func checkConnection(ctx context.Context, pool *pgxpool.Pool) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	return conn.Ping(ctx)
}

func watchPoolStats(ctx context.Context, pool *pgxpool.Pool) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	var lastAcquireCount int64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := pool.Stat()
			metrics.NostrDBPoolAcquired.Set(float64(stats.AcquiredConns()))
			metrics.NostrDBPoolIdle.Set(float64(stats.IdleConns()))
			metrics.NostrDBPoolTotal.Set(float64(stats.TotalConns()))
			acquireDelta := stats.AcquireCount() - lastAcquireCount
			if acquireDelta > 0 {
				metrics.NostrDBPoolAcquireCount.Add(float64(acquireDelta))
				lastAcquireCount = stats.AcquireCount()
			}
		}
	}
}
