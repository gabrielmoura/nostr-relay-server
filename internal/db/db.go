package db

import (
	"context"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
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
		// ...
		return nil
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
