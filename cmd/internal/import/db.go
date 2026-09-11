package _import

import (
	"context"
	"errors"
	"fmt"
	dbx "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nbd-wtf/go-nostr"
)

func saveToDatabase(ctx context.Context, store *dbx.Queries, event *nostr.Event) error {
	nCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := store.InsertEvent(nCtx, event); err != nil {
		return fmt.Errorf("failed to save event to event store: %w", err)
	}
	return nil
}

func saveBatchToDatabase(ctx context.Context, store *dbx.Queries, batch Batch) []ErrorInfo {
	nCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	if _, err := store.MigrateEventsViaCopy(nCtx, batch.Items); err == nil {
		return nil
	} else if !isRowPersistenceError(err) {
		return allBatchLineErrors(batch, fmt.Errorf("persistência COPY: %w", err))
	}

	result, err := store.InsertEventBatch(nCtx, batch.Items)
	if err != nil {
		return allBatchLineErrors(batch, fmt.Errorf("persistência fallback: %w", err))
	}

	return rejectedEventLineErrors(batch, result.RejectedEventIDs)
}

func isRowPersistenceError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr)
}

func allBatchLineErrors(batch Batch, err error) []ErrorInfo {
	errorsByLine := make([]ErrorInfo, 0, len(batch.LineNumbers))
	for _, lineNumber := range batch.LineNumbers {
		errorsByLine = append(errorsByLine, ErrorInfo{LineNumber: lineNumber, Err: err})
	}
	return errorsByLine
}

func rejectedEventLineErrors(batch Batch, rejectedIDs []string) []ErrorInfo {
	if len(rejectedIDs) == 0 {
		return nil
	}

	rejected := make(map[string]struct{}, len(rejectedIDs))
	for _, id := range rejectedIDs {
		rejected[id] = struct{}{}
	}

	errorsByLine := make([]ErrorInfo, 0, len(rejectedIDs))
	for index, event := range batch.Items {
		if event == nil || index >= len(batch.LineNumbers) {
			continue
		}
		if _, ok := rejected[event.ID]; !ok {
			continue
		}

		errorsByLine = append(errorsByLine, ErrorInfo{
			LineNumber: batch.LineNumbers[index],
			Err:        fmt.Errorf("persistência: evento %s rejeitado; veja o log do relay para a constraint", event.ID),
		})
	}

	return errorsByLine
}
