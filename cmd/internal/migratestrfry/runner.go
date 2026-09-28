package migratestrfry

import (
	"context"
	"errors"
	"fmt"

	"github.com/gabrielmoura/nostr-relay-server/config"
	storedb "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/nbd-wtf/go-nostr"
)

func Run(raw CLIOptions) (Result, error) {
	options, err := BuildOptions(raw)
	if err != nil {
		return Result{}, err
	}
	if options.Source == SourceLMDB {
		return Result{}, errors.New("direct LMDB reader is not available for this strfry database layout; use --source=export")
	}
	if err := config.LoadConfig(); err != nil {
		return Result{}, fmt.Errorf("load target conf.yaml: %w", err)
	}

	var persister batchPersister
	if !options.DryRun {
		log.Init()
		if err := storedb.Init(context.Background()); err != nil {
			return Result{}, fmt.Errorf("init target database: %w", err)
		}
		persister = databasePersister{store: storedb.DbQueries()}
	}

	processor, err := newProcessor(options, persister)
	if err != nil {
		return Result{}, err
	}
	runContext := context.Background()
	var source eventSource
	switch options.Source {
	case SourceExport:
		source = newExportSource(options)
	default:
		return Result{}, fmt.Errorf("unsupported strfry source %q", options.Source)
	}
	if err := source.Stream(runContext, func(lineNumber int, line []byte) error {
		return processor.AddLine(runContext, lineNumber, line)
	}); err != nil {
		return Result{}, err
	}

	return processor.Finish(runContext)
}

type eventSource interface {
	Stream(context.Context, func(int, []byte) error) error
}

type databasePersister struct {
	store *storedb.Queries
}

func (p databasePersister) Persist(ctx context.Context, events []*nostr.Event) (persistResult, error) {
	copyResult, copyErr := p.store.MigrateEventsViaCopy(ctx, events)
	if copyErr == nil {
		return persistResult{Persisted: copyResult.Inserted + copyResult.Duplicates}, nil
	}

	result, err := p.store.InsertEventBatch(ctx, events)
	if err != nil {
		return persistResult{}, fmt.Errorf("persist COPY batch: %w; persist fallback batch: %w", copyErr, err)
	}
	return persistResult{
		Persisted: result.Inserted + result.Duplicates,
		Rejected:  len(result.RejectedEventIDs),
	}, nil
}
