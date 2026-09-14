package db

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	dbPool    *pgxpool.Pool
	dbQueries *Queries
	dbMu      sync.RWMutex
)

type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	SendBatch(context.Context, *pgx.Batch) pgx.BatchResults
}

func New(db DBTX) *Queries {
	return &Queries{
		db: db,
	}
}

func (q *Queries) StatPool() *pgxpool.Stat {
	return q.db.(*pgxpool.Pool).Stat()
}

type Queries struct {
	db DBTX
}

func (q *Queries) WithTx(tx pgx.Tx) *Queries {
	return &Queries{
		db: tx,
	}
}

func DBPool() *pgxpool.Pool {
	dbMu.RLock()
	defer dbMu.RUnlock()
	return dbPool
}

func DbQueries() *Queries {
	dbMu.RLock()
	defer dbMu.RUnlock()
	return dbQueries
}

func Init(ctx context.Context) error {
	dbMu.Lock()
	defer dbMu.Unlock()

	if dbPool != nil {
		return nil
	}

	connStr := config.Cfg.DB.PostgresURI
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return err
	}

	// Pool config
	if config.Cfg.DB.MaxConns > 0 {
		poolConfig.MaxConns = config.Cfg.DB.MaxConns
	}
	if config.Cfg.DB.MinConns > 0 {
		poolConfig.MinConns = config.Cfg.DB.MinConns
	}
	if config.Cfg.DB.MaxConnLifetimeMinutes > 0 {
		poolConfig.MaxConnLifetime = time.Duration(config.Cfg.DB.MaxConnLifetimeMinutes) * time.Minute
	}
	if config.Cfg.DB.MaxConnIdleMinutes > 0 {
		poolConfig.MaxConnIdleTime = time.Duration(config.Cfg.DB.MaxConnIdleMinutes) * time.Minute
	}

	dbPool, err = pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return err
	}

	if err := dbPool.Ping(ctx); err != nil {
		dbPool.Close()
		dbPool = nil
		return err
	}

	log.Logger.Info("database connected",
		zap.String("host", poolConfig.ConnConfig.Host),
		zap.Int("max", int(poolConfig.MaxConns)),
	)

	if err := CheckMigrationsCurrent(ctx, connStr); err != nil {
		dbPool.Close()
		dbPool = nil
		return err
	}

	q := New(dbPool)
	dbQueries = q
	log.Logger.Info("database initialized")
	return nil
}

func Close() {
	dbMu.Lock()
	defer dbMu.Unlock()

	if dbPool != nil {
		dbPool.Close()
		dbPool = nil
		dbQueries = nil
		log.Logger.Info("database closed")
	}
}

func HealthCheck(ctx context.Context) error {
	dbMu.RLock()
	defer dbMu.RUnlock()

	if dbPool == nil {
		return errors.New("pool not initialized")
	}
	return dbPool.Ping(ctx)
}
