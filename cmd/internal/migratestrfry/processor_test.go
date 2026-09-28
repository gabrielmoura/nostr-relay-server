package migratestrfry

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/require"
)

func TestProcessExportDryRunValidatesWithoutPersisting(t *testing.T) {
	t.Parallel()

	valid := signedEventJSON(t)
	persister := &fakePersister{}
	result, err := processExport(
		context.Background(),
		strings.NewReader(valid+"\n{bad json}\n"),
		CLIOptions{BatchSize: 1, DryRun: true},
		persister,
	)

	require.NoError(t, err)
	require.Equal(t, Result{Read: 2, Valid: 1, Invalid: 1}, result)
	require.Empty(t, persister.batches)
}

func TestProcessExportFailsOnRowErrorsWhenRequested(t *testing.T) {
	t.Parallel()

	result, err := processExport(
		context.Background(),
		strings.NewReader("{}\n"),
		CLIOptions{BatchSize: 2, DryRun: true, FailOnError: true},
		nil,
	)

	require.ErrorContains(t, err, "1 row errors")
	require.Equal(t, 1, result.Invalid)
}

func TestProcessExportAbortsOnPersistenceError(t *testing.T) {
	t.Parallel()

	valid := signedEventJSON(t)
	persister := &fakePersister{err: errors.New("database unavailable")}
	result, err := processExport(
		context.Background(),
		strings.NewReader(valid+"\n"),
		CLIOptions{BatchSize: 1},
		persister,
	)

	require.ErrorContains(t, err, "persist export batch")
	require.Zero(t, result.Persisted)
	require.Len(t, persister.batches, 1)
}

func TestProcessExportCountsRejectedRowsUnlessFailOnError(t *testing.T) {
	t.Parallel()

	valid := signedEventJSON(t)
	persister := &fakePersister{result: persistResult{Persisted: 1, Rejected: 1}}
	result, err := processExport(
		context.Background(),
		strings.NewReader(valid+"\n"),
		CLIOptions{BatchSize: 1},
		persister,
	)

	require.NoError(t, err)
	require.Equal(t, Result{Read: 1, Valid: 1, Persisted: 1, Rejected: 1}, result)

	_, err = processExport(
		context.Background(),
		strings.NewReader(valid+"\n"),
		CLIOptions{BatchSize: 1, FailOnError: true},
		persister,
	)
	require.ErrorContains(t, err, "1 row errors")
}

type fakePersister struct {
	batches [][]*nostr.Event
	err     error
	result  persistResult
}

func (p *fakePersister) Persist(_ context.Context, events []*nostr.Event) (persistResult, error) {
	p.batches = append(p.batches, events)
	if p.result == (persistResult{}) {
		p.result.Persisted = len(events)
	}
	return p.result, p.err
}

func signedEventJSON(t *testing.T) string {
	t.Helper()
	privateKey := nostr.GeneratePrivateKey()
	event := nostr.Event{CreatedAt: nostr.Now(), Kind: 1, Content: "migration test", Tags: nostr.Tags{}}
	require.NoError(t, event.Sign(privateKey))
	data, err := json.Marshal(event)
	require.NoError(t, err)
	return string(data)
}
