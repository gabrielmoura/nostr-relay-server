package _import

import (
	"context"
	"errors"
	dbx "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/nbd-wtf/go-nostr"
)

var ErrDupEvent = errors.New("duplicate: event already exists")

type Job struct {
	Line       string
	LineNumber int
}

type Batch struct {
	Items       []*nostr.Event
	LineNumbers []int
}

type ErrorInfo struct {
	Err        error
	LineNumber int
}
type ConfImport struct {
	ctx        context.Context
	dbc        *dbx.Queries
	filename   string
	batchSize  int
	numWorkers int
	timeout    int
}
