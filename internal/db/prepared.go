package db

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type PreparedStmtManager struct {
	pool  *pgxpool.Pool
	mu    sync.RWMutex
	cache map[string]*pgconn.StatementDescription
}

var (
	psm *PreparedStmtManager
)

const (
	StmtEventByID          = "ps_event_by_id"
	StmtEventsByPubkey     = "ps_events_by_pubkey"
	StmtEventsByKind       = "ps_events_by_kind"
	StmtEventsByPubkeyKind = "ps_events_by_pubkey_kind"
	StmtCountByFilter      = "ps_count_by_filter"
	StmtEventsByTag        = "ps_events_by_tag"
	StmtEventsRecent       = "ps_events_recent"
)

var stmtSQLs = map[string]string{
	StmtEventByID: `
		SELECT id, pubkey, created_at, kind, tags, content, sig 
		FROM event 
		WHERE id = $1
		LIMIT 1
	`,
	StmtEventsByPubkey: `
		SELECT id, pubkey, created_at, kind, tags, content, sig 
		FROM event 
		WHERE pubkey = $1 
		ORDER BY created_at DESC 
		LIMIT $2
	`,
	StmtEventsByKind: `
		SELECT id, pubkey, created_at, kind, tags, content, sig 
		FROM event 
		WHERE kind = $1 AND created_at > $2 
		ORDER BY created_at DESC 
		LIMIT $3
	`,
	StmtEventsByPubkeyKind: `
		SELECT id, pubkey, created_at, kind, tags, content, sig 
		FROM event 
		WHERE pubkey = $1 AND kind = $2 
		ORDER BY created_at DESC 
		LIMIT $3
	`,
	StmtCountByFilter: `
		SELECT COUNT(*) 
		FROM event 
		WHERE pubkey = $1
	`,
	StmtEventsByTag: `
		SELECT id, pubkey, created_at, kind, tags, content, sig 
		FROM event 
		WHERE $1 = ANY(tagvalues) 
		ORDER BY created_at DESC 
		LIMIT $2
	`,
	StmtEventsRecent: `
		SELECT id, pubkey, created_at, kind, tags, content, sig 
		FROM event 
		WHERE created_at > $1 
		ORDER BY created_at DESC 
		LIMIT $2
	`,
}

func InitPreparedStatements(ctx context.Context, pool *pgxpool.Pool) error {
	psm = &PreparedStmtManager{
		pool:  pool,
		cache: make(map[string]*pgconn.StatementDescription),
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	if err := psm.prepareAll(ctx, conn.Conn()); err != nil {
		return fmt.Errorf("failed to prepare statements: %w", err)
	}

	log.Logger.Info("prepared statements initialized",
		zap.Int("count", len(psm.cache)),
	)
	return nil
}

func PrepareConn(ctx context.Context, conn interface {
	Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error)
}) error {
	if psm == nil {
		psm = &PreparedStmtManager{cache: make(map[string]*pgconn.StatementDescription)}
	}
	return psm.prepareAll(ctx, conn)
}

func (p *PreparedStmtManager) prepareAll(ctx context.Context, conn interface {
	Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error)
}) error {
	for name, sql := range stmtSQLs {
		desc, err := conn.Prepare(ctx, name, sql)
		if err != nil {
			return fmt.Errorf("failed to prepare %s: %w", name, err)
		}
		p.cache[name] = desc
		log.Logger.Debug("prepared statement created", zap.String("name", name))
	}
	return nil
}

func GetPreparedStatement(name string) *pgconn.StatementDescription {
	if psm == nil {
		return nil
	}
	psm.mu.RLock()
	defer psm.mu.RUnlock()
	return psm.cache[name]
}

func PreparedStatementExists(name string) bool {
	if psm == nil {
		return false
	}
	psm.mu.RLock()
	defer psm.mu.RUnlock()
	_, ok := psm.cache[name]
	return ok
}
