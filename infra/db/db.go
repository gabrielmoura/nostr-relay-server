package db

import (
	"context"
	_ "embed"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

//go:embed schema.sql
var schema string

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

func (q *Queries) Migrate(ctx context.Context) error {
	for _, statement := range splitSQLStatements(schema) {
		if _, err := q.db.Exec(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func splitSQLStatements(input string) []string {
	statements := make([]string, 0, 64)
	var builder strings.Builder
	inSingleQuote := false
	runes := []rune(input)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		builder.WriteRune(r)
		if r == '\'' {
			nextIsQuote := i+1 < len(runes) && runes[i+1] == '\''
			if nextIsQuote {
				builder.WriteRune(runes[i+1])
				i++
				continue
			}
			inSingleQuote = !inSingleQuote
			continue
		}
		if r == ';' && !inSingleQuote {
			statement := strings.TrimSpace(builder.String())
			if statement != ";" && statement != "" {
				statements = append(statements, statement)
			}
			builder.Reset()
		}
	}
	if tail := strings.TrimSpace(builder.String()); tail != "" {
		statements = append(statements, tail)
	}
	return statements
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

	// Ensure schema_version table
	_, _ = dbPool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_version (
			version VARCHAR(14) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			description TEXT NOT NULL
		)
	`)

	// Run schema migration
	q := New(dbPool)
	if err := q.Migrate(ctx); err != nil {
		log.Logger.Warn("schema migration", zap.Error(err))
	}

	// Record migration
	_, _ = dbPool.Exec(ctx,
		"INSERT INTO schema_version (version, description) VALUES ('001', 'Initial schema') ON CONFLICT (version) DO NOTHING",
	)

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
