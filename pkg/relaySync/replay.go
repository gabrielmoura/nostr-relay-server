package relaysync

import (
	"context"
	"errors"
	"io"
)

type ReplayService struct {
	journal     JournalProvider
	applier     ChangeApplier
	checkpoints CheckpointStore
}

func NewReplayService(journal JournalProvider, applier ChangeApplier, checkpoints CheckpointStore) (*ReplayService, error) {
	if journal == nil || applier == nil || checkpoints == nil {
		return nil, ErrNotFound
	}
	return &ReplayService{journal: journal, applier: applier, checkpoints: checkpoints}, nil
}

func (r *ReplayService) Replay(ctx context.Context, key CheckpointKey, scope Scope, after Cursor, limit int) (Cursor, error) {
	stream, err := r.journal.OpenJournal(ctx, scope, after)
	if err != nil {
		return after, err
	}
	defer stream.Close()
	expected := after
	if expected.Epoch == 0 {
		expected.Epoch = stream.Epoch()
	}
	if limit <= 0 {
		limit = 1000
	}
	for count := 0; count < limit; count++ {
		change, err := stream.Next(ctx)
		if errors.Is(err, io.EOF) {
			return expected, nil
		}
		if err != nil {
			return expected, err
		}
		if change.Cursor.Epoch != expected.Epoch || change.Cursor.Sequence != expected.Sequence+1 {
			return expected, ErrJournalGap
		}
		result, err := r.applier.ApplyChange(ctx, change)
		if err != nil {
			return expected, err
		}
		if result.Status == WriteFailedTemporary {
			return expected, ErrJournalGap
		}
		expected = change.Cursor
		if err := r.checkpoints.AdvanceCheckpoint(ctx, Checkpoint{Key: key, Cursor: expected, DecisionID: result.DecisionID}); err != nil {
			return expected, err
		}
	}
	return expected, nil
}
