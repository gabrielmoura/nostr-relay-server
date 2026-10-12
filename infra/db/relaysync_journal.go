package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	relaysync "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
	"github.com/jackc/pgx/v5"
)

// RelaySyncJournalProvider streams the durable PostgreSQL journal without
// materializing retained changes in memory. It is safe to construct per
// relay/process and leaves retention to the projector worker.
type RelaySyncJournalProvider struct {
	queries *Queries
	epoch   uint64
}

func NewRelaySyncJournalProvider(queries *Queries, epoch uint64) (*RelaySyncJournalProvider, error) {
	if queries == nil || epoch == 0 {
		return nil, fmt.Errorf("relaysync: journal provider requires queries and a positive epoch")
	}
	return &RelaySyncJournalProvider{queries: queries, epoch: epoch}, nil
}

func (p *RelaySyncJournalProvider) OpenJournal(ctx context.Context, scope relaysync.Scope, after relaysync.Cursor) (relaysync.Journal, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if after.Epoch != 0 && after.Epoch != p.epoch {
		return nil, relaysync.ErrJournalEpochChanged
	}
	var minSequence, maxSequence uint64
	if err := p.queries.db.QueryRow(ctx, `
SELECT COALESCE(MIN(sequence), 0), COALESCE(MAX(sequence), 0)
FROM relay_sync_journal
WHERE epoch = $1 AND scope_name = $2 AND scope_version = $3 AND scope_hash = $4`,
		p.epoch, scope.Name, scope.Version, scope.Hash[:]).Scan(&minSequence, &maxSequence); err != nil {
		return nil, fmt.Errorf("relaysync: read journal bounds: %w", err)
	}
	if minSequence != 0 && after.Sequence+1 < minSequence {
		return nil, relaysync.ErrJournalGap
	}
	rows, err := p.queries.db.Query(ctx, `
SELECT sequence, operation, event_id, pubkey, created_at, kind, tags, content, sig
FROM relay_sync_journal
WHERE epoch = $1 AND scope_name = $2 AND scope_version = $3 AND scope_hash = $4 AND sequence > $5
ORDER BY sequence`, p.epoch, scope.Name, scope.Version, scope.Hash[:], after.Sequence)
	if err != nil {
		return nil, fmt.Errorf("relaysync: open journal: %w", err)
	}
	return &postgresJournal{rows: rows, epoch: p.epoch, scope: scope}, nil
}

type postgresJournal struct {
	rows   pgx.Rows
	epoch  uint64
	scope  relaysync.Scope
	closed bool
}

func (j *postgresJournal) Epoch() uint64 { return j.epoch }

func (j *postgresJournal) Next(ctx context.Context) (relaysync.JournalChange, error) {
	if err := ctx.Err(); err != nil {
		return relaysync.JournalChange{}, err
	}
	if j.closed || !j.rows.Next() {
		if err := j.rows.Err(); err != nil {
			return relaysync.JournalChange{}, err
		}
		return relaysync.JournalChange{}, io.EOF
	}
	var sequence uint64
	var operation, eventID string
	var pubkey, content, signature *string
	var createdAt, kind *int32
	var tags []byte
	if err := j.rows.Scan(&sequence, &operation, &eventID, &pubkey, &createdAt, &kind, &tags, &content, &signature); err != nil {
		return relaysync.JournalChange{}, err
	}
	id, err := relaysync.ParseEventID(eventID)
	if err != nil {
		return relaysync.JournalChange{}, err
	}
	change := relaysync.JournalChange{Cursor: relaysync.Cursor{Epoch: j.epoch, Sequence: sequence}, ID: id}
	if createdAt != nil {
		change.Metadata.CreatedAt = int64(*createdAt)
	}
	change.Metadata.ID = id
	switch operation {
	case "DELETE":
		change.Type = relaysync.ChangeDelete
	case "UPSERT":
		if pubkey == nil || content == nil || signature == nil || createdAt == nil || kind == nil {
			return relaysync.JournalChange{}, errors.New("relaysync: incomplete upsert journal row")
		}
		publicKey, err := relaysync.ParsePublicKey(*pubkey)
		if err != nil {
			return relaysync.JournalChange{}, err
		}
		sig, err := relaysync.ParseSignature(*signature)
		if err != nil {
			return relaysync.JournalChange{}, err
		}
		var rawTags [][]string
		if len(tags) > 0 && string(tags) != "null" {
			if err := json.Unmarshal(tags, &rawTags); err != nil {
				return relaysync.JournalChange{}, err
			}
		}
		tagsValue := make([]relaysync.Tag, len(rawTags))
		for i := range rawTags {
			tagsValue[i] = append(relaysync.Tag(nil), rawTags[i]...)
		}
		change.Type = relaysync.ChangeUpsert
		change.Event = &relaysync.Event{ID: id, PublicKey: publicKey, Signature: sig, CreatedAt: int64(*createdAt), Kind: uint32(*kind), Tags: tagsValue, Content: *content}
	default:
		return relaysync.JournalChange{}, fmt.Errorf("relaysync: unknown journal operation %q", operation)
	}
	return change, nil
}

func (j *postgresJournal) Close() error {
	if j.closed {
		return nil
	}
	j.closed = true
	j.rows.Close()
	return j.rows.Err()
}
