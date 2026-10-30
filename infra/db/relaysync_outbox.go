package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrRelaySyncCommitTimestampsDisabled = errors.New("relaysync: PostgreSQL track_commit_timestamp must be enabled")

// RelaySyncOutboxEntry is the durable change captured by the event trigger.
// The projector converts it to the RelaySync journal representation only after
// PostgreSQL exposes a commit timestamp for its transaction.
type RelaySyncOutboxEntry struct {
	ID           int64
	TxID         int64
	CommittedAt  time.Time
	Operation    string
	EventID      string
	Pubkey       *string
	CreatedAt    *int32
	Kind         *int32
	Tags         []byte
	Content      *string
	Signature    *string
	ScopeName    string
	ScopeVersion int32
	ScopeHash    []byte
}

const relaySyncOutboxLockKey int64 = 0x52534f5554424f58 // "RSOUTBOX"

// ProjectRelaySyncOutbox publishes a bounded batch in commit order. It is
// intentionally not called by normal event ingestion yet; the host must wire
// it to a supervised worker after validating PostgreSQL retention settings.
func (q *Queries) ProjectRelaySyncOutbox(ctx context.Context, epoch uint64, limit int) (int, error) {
	if limit <= 0 {
		return 0, fmt.Errorf("relaysync: invalid projector limit %d", limit)
	}
	starter, ok := q.db.(transactionStarter)
	if !ok {
		return 0, errors.New("relaysync: database does not support transactions")
	}
	tx, err := starter.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("relaysync: begin projector transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", relaySyncOutboxLockKey); err != nil {
		return 0, fmt.Errorf("relaysync: lock projector: %w", err)
	}
	if err := ensureRelaySyncCommitTimestamps(ctx, tx); err != nil {
		return 0, err
	}
	entries, err := readRelaySyncOutbox(ctx, tx, limit)
	if err != nil {
		return 0, err
	}
	var sequence uint64
	if err := tx.QueryRow(ctx, "SELECT COALESCE(MAX(sequence), 0) FROM relay_sync_journal WHERE epoch = $1", epoch).Scan(&sequence); err != nil {
		return 0, fmt.Errorf("relaysync: read journal cursor: %w", err)
	}
	for _, entry := range entries {
		sequence++
		if _, err := tx.Exec(ctx, `
INSERT INTO relay_sync_journal
    (epoch, sequence, outbox_id, txid, committed_at, operation, event_id,
     pubkey, created_at, kind, tags, content, sig, scope_name, scope_version, scope_hash)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
ON CONFLICT (outbox_id) DO UPDATE SET outbox_id = EXCLUDED.outbox_id`,
			epoch, sequence, entry.ID, entry.TxID, entry.CommittedAt, entry.Operation,
			entry.EventID, entry.Pubkey, entry.CreatedAt, entry.Kind, entry.Tags,
			entry.Content, entry.Signature, entry.ScopeName, entry.ScopeVersion, entry.ScopeHash); err != nil {
			return 0, fmt.Errorf("relaysync: publish outbox %d: %w", entry.ID, err)
		}
		if _, err := tx.Exec(ctx, `UPDATE relay_sync_outbox SET published_at = $2, journal_epoch = $3, journal_sequence = $4 WHERE id = $1`, entry.ID, entry.CommittedAt, epoch, sequence); err != nil {
			return 0, fmt.Errorf("relaysync: mark outbox %d published: %w", entry.ID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("relaysync: commit projector transaction: %w", err)
	}
	return len(entries), nil
}

func ensureRelaySyncCommitTimestamps(ctx context.Context, tx pgx.Tx) error {
	var enabled string
	if err := tx.QueryRow(ctx, "SHOW track_commit_timestamp").Scan(&enabled); err != nil {
		return fmt.Errorf("relaysync: check track_commit_timestamp: %w", err)
	}
	if enabled != "on" {
		return ErrRelaySyncCommitTimestampsDisabled
	}
	return nil
}

func readRelaySyncOutbox(ctx context.Context, tx pgx.Tx, limit int) ([]RelaySyncOutboxEntry, error) {
	rows, err := tx.Query(ctx, `
SELECT id, txid, pg_xact_commit_timestamp(txid::xid), operation, event_id,
       pubkey, created_at, kind, tags, content, sig, scope_name, scope_version, scope_hash
FROM relay_sync_outbox
WHERE published_at IS NULL
ORDER BY pg_xact_commit_timestamp(txid::xid), txid, id
LIMIT $1
FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return nil, fmt.Errorf("relaysync: read outbox: %w", err)
	}
	defer rows.Close()
	entries := make([]RelaySyncOutboxEntry, 0, limit)
	for rows.Next() {
		var entry RelaySyncOutboxEntry
		var committedAt *time.Time
		if err := rows.Scan(&entry.ID, &entry.TxID, &committedAt, &entry.Operation, &entry.EventID,
			&entry.Pubkey, &entry.CreatedAt, &entry.Kind, &entry.Tags, &entry.Content,
			&entry.Signature, &entry.ScopeName, &entry.ScopeVersion, &entry.ScopeHash); err != nil {
			return nil, fmt.Errorf("relaysync: scan outbox: %w", err)
		}
		if committedAt == nil {
			return nil, ErrRelaySyncCommitTimestampsDisabled
		}
		entry.CommittedAt = committedAt.UTC()
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("relaysync: iterate outbox: %w", err)
	}
	return entries, nil
}

// PurgeRelaySyncOutbox removes only entries already published to the journal.
func (q *Queries) PurgeRelaySyncOutbox(ctx context.Context, before time.Time, limit int) (int64, error) {
	if limit <= 0 {
		return 0, fmt.Errorf("relaysync: invalid outbox retention limit %d", limit)
	}
	result, err := q.db.Exec(ctx, `
WITH doomed AS (
    SELECT id FROM relay_sync_outbox
    WHERE published_at IS NOT NULL AND published_at < $1
    ORDER BY published_at, id LIMIT $2
)
DELETE FROM relay_sync_outbox o USING doomed d WHERE o.id = d.id`, before, limit)
	if err != nil {
		return 0, fmt.Errorf("relaysync: purge outbox: %w", err)
	}
	return result.RowsAffected(), nil
}

// PurgeRelaySyncJournal applies journal retention independently of peers.
func (q *Queries) PurgeRelaySyncJournal(ctx context.Context, before time.Time, limit int) (int64, error) {
	if limit <= 0 {
		return 0, fmt.Errorf("relaysync: invalid journal retention limit %d", limit)
	}
	result, err := q.db.Exec(ctx, `
WITH doomed AS (
    SELECT epoch, sequence FROM relay_sync_journal
    WHERE committed_at < $1
    ORDER BY committed_at, epoch, sequence LIMIT $2
)
DELETE FROM relay_sync_journal j USING doomed d
WHERE j.epoch = d.epoch AND j.sequence = d.sequence`, before, limit)
	if err != nil {
		return 0, fmt.Errorf("relaysync: purge journal: %w", err)
	}
	return result.RowsAffected(), nil
}
