package db

import (
	"context"
	_ "embed"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schema string

type DBTX interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
	SendBatch(context.Context, *pgx.Batch) pgx.BatchResults
}

func New(db DBTX) *Queries {
	return &Queries{
		db: db,
	}
}

func (q *Queries) Migrate(ctx context.Context) error {
	_, err := q.db.Exec(ctx, schema)
	return err
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
