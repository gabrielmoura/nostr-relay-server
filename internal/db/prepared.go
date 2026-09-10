package db

import (
	"context"
	"fmt"

	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type statementPreparer interface {
	Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error)
}

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

// PrepareConn prepares the statements needed by one physical PostgreSQL connection.
// Statement descriptions belong to that connection and must not be shared with the pool.
func PrepareConn(ctx context.Context, conn statementPreparer) error {
	return prepareAll(ctx, conn)
}

func prepareAll(ctx context.Context, conn statementPreparer) error {
	for name, sql := range stmtSQLs {
		_, err := conn.Prepare(ctx, name, sql)
		if err != nil {
			return fmt.Errorf("failed to prepare %s: %w", name, err)
		}
		log.Logger.Debug("prepared statement created", zap.String("name", name))
	}
	return nil
}
