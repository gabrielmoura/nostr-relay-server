package db

import (
	"context"
	"errors"
	"fmt"

	relaysync "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
	"github.com/jackc/pgx/v5"
)

// RelaySyncCheckpointStore persists monotonic cursors across process restarts.
type RelaySyncCheckpointStore struct{ queries *Queries }

func NewRelaySyncCheckpointStore(queries *Queries) (*RelaySyncCheckpointStore, error) {
	if queries == nil {
		return nil, fmt.Errorf("relaysync: checkpoint store requires queries")
	}
	return &RelaySyncCheckpointStore{queries: queries}, nil
}

func (s *RelaySyncCheckpointStore) LoadCheckpoint(ctx context.Context, key relaysync.CheckpointKey) (relaysync.Checkpoint, bool, error) {
	var epoch, sequence uint64
	var decision string
	err := s.queries.db.QueryRow(ctx, `SELECT journal_epoch, journal_sequence, decision_id FROM relay_sync_checkpoint WHERE peer_id=$1 AND scope_name=$2 AND scope_version=$3 AND scope_hash=$4`, key.PeerID, key.Scope.Name, key.Scope.Version, key.Scope.Hash[:]).Scan(&epoch, &sequence, &decision)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return relaysync.Checkpoint{}, false, nil
		}
		return relaysync.Checkpoint{}, false, err
	}
	return relaysync.Checkpoint{Key: key, Cursor: relaysync.Cursor{Epoch: epoch, Sequence: sequence}, DecisionID: decision}, true, nil
}

func (s *RelaySyncCheckpointStore) AdvanceCheckpoint(ctx context.Context, next relaysync.Checkpoint) error {
	result, err := s.queries.db.Exec(ctx, `
INSERT INTO relay_sync_checkpoint (peer_id,scope_name,scope_version,scope_hash,journal_epoch,journal_sequence,decision_id)
VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT (peer_id,scope_name,scope_version,scope_hash) DO UPDATE SET journal_epoch=EXCLUDED.journal_epoch,journal_sequence=EXCLUDED.journal_sequence,decision_id=EXCLUDED.decision_id,updated_at=clock_timestamp()
WHERE relay_sync_checkpoint.journal_epoch=EXCLUDED.journal_epoch AND relay_sync_checkpoint.journal_sequence <= EXCLUDED.journal_sequence`, next.Key.PeerID, next.Key.Scope.Name, next.Key.Scope.Version, next.Key.Scope.Hash[:], next.Cursor.Epoch, next.Cursor.Sequence, next.DecisionID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return relaysync.ErrConflict
	}
	return nil
}

func (s *RelaySyncCheckpointStore) ResetCheckpoint(ctx context.Context, next relaysync.Checkpoint) error {
	_, err := s.queries.db.Exec(ctx, `INSERT INTO relay_sync_checkpoint (peer_id,scope_name,scope_version,scope_hash,journal_epoch,journal_sequence,decision_id) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (peer_id,scope_name,scope_version,scope_hash) DO UPDATE SET journal_epoch=EXCLUDED.journal_epoch,journal_sequence=EXCLUDED.journal_sequence,decision_id=EXCLUDED.decision_id,updated_at=clock_timestamp()`, next.Key.PeerID, next.Key.Scope.Name, next.Key.Scope.Version, next.Key.Scope.Hash[:], next.Cursor.Epoch, next.Cursor.Sequence, next.DecisionID)
	return err
}
