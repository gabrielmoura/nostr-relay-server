package db

import (
	"context"
	"sync"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type recordingPreparer struct {
	mu       sync.Mutex
	prepared map[string]string
}

func (p *recordingPreparer) Prepare(_ context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.prepared[name] = sql
	return &pgconn.StatementDescription{Name: name, SQL: sql}, nil
}

func TestPrepareConn_PreparesStatementsIndependentlyForConcurrentConnections(t *testing.T) {
	const connections = 32

	previousLogger := log.Logger
	log.Logger = zap.NewNop()
	t.Cleanup(func() {
		log.Logger = previousLogger
	})

	preparers := make([]*recordingPreparer, connections)
	for i := range preparers {
		preparers[i] = &recordingPreparer{prepared: make(map[string]string)}
	}

	start := make(chan struct{})
	errCh := make(chan error, connections)
	var wg sync.WaitGroup
	for _, preparer := range preparers {
		wg.Go(func() {
			<-start
			errCh <- PrepareConn(context.Background(), preparer)
		})
	}

	close(start)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("PrepareConn() error = %v", err)
		}
	}

	for connection, preparer := range preparers {
		preparer.mu.Lock()
		prepared := preparer.prepared
		preparer.mu.Unlock()

		if len(prepared) != len(stmtSQLs) {
			t.Errorf("connection %d prepared %d statements, want %d", connection, len(prepared), len(stmtSQLs))
		}
		for name, sql := range stmtSQLs {
			if got := prepared[name]; got != sql {
				t.Errorf("connection %d statement %q = %q, want %q", connection, name, got, sql)
			}
		}
	}
}
