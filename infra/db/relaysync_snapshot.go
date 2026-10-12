package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	relaysync "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
)

const relaySyncPartitionSeconds int64 = 30 * 24 * 60 * 60

// RelaySyncSnapshotProvider provides a bounded-lifetime PostgreSQL snapshot
// view. Queries are paginated and event payloads are only materialized for a
// requested batch. The host must keep the provider behind its scope policy.
type RelaySyncSnapshotProvider struct {
	queries *Queries
	epoch   uint64
	ttl     time.Duration
}

func NewRelaySyncSnapshotProvider(queries *Queries, epoch uint64, ttl time.Duration) (*RelaySyncSnapshotProvider, error) {
	if queries == nil || epoch == 0 || ttl <= 0 {
		return nil, fmt.Errorf("relaysync: snapshot provider requires queries, epoch and positive TTL")
	}
	return &RelaySyncSnapshotProvider{queries: queries, epoch: epoch, ttl: ttl}, nil
}

func (p *RelaySyncSnapshotProvider) OpenSnapshot(ctx context.Context, scope relaysync.Scope) (relaysync.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var watermark uint64
	if err := p.queries.db.QueryRow(ctx, `SELECT COALESCE(MAX(sequence), 0) FROM relay_sync_journal WHERE epoch=$1 AND scope_name=$2 AND scope_version=$3 AND scope_hash=$4`, p.epoch, scope.Name, scope.Version, scope.Hash[:]).Scan(&watermark); err != nil {
		return nil, fmt.Errorf("relaysync: snapshot watermark: %w", err)
	}
	rows, err := p.queries.db.Query(ctx, `
SELECT floor(created_at / $1)::bigint AS bucket, lower(left(id, 2)) AS prefix
FROM event
WHERE deleted_by IS NULL
GROUP BY bucket, prefix
ORDER BY bucket, prefix`, relaySyncPartitionSeconds)
	if err != nil {
		return nil, fmt.Errorf("relaysync: snapshot partitions: %w", err)
	}
	defer rows.Close()
	partitions := make([]relaysync.PartitionKey, 0)
	for rows.Next() {
		var bucket int64
		var prefixHex string
		if err := rows.Scan(&bucket, &prefixHex); err != nil {
			return nil, err
		}
		prefix, err := hex.DecodeString(prefixHex)
		if err != nil || len(prefix) != 1 {
			return nil, fmt.Errorf("relaysync: invalid partition prefix %q", prefixHex)
		}
		key := relaysync.PartitionKey{AlgorithmVersion: 1, Bucket: bucket, PrefixBits: 8}
		key.Prefix[0] = prefix[0]
		partitions = append(partitions, key)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	versionInput := fmt.Sprintf("%s\x00%d\x00%d", scope.Name, scope.Version, watermark)
	version := sha256.Sum256([]byte(versionInput))
	return &postgresSnapshot{queries: p.queries, descriptor: relaysync.SnapshotDescriptor{Version: version[:], Scope: scope, Watermark: relaysync.Cursor{Epoch: p.epoch, Sequence: watermark}, ExpiresAt: time.Now().Add(p.ttl), Partitions: partitions}}, nil
}

type postgresSnapshot struct {
	queries    *Queries
	descriptor relaysync.SnapshotDescriptor
	mu         sync.Mutex
	closed     bool
}

func (s *postgresSnapshot) Descriptor() relaysync.SnapshotDescriptor { return s.descriptor }
func (s *postgresSnapshot) Close() error                             { s.mu.Lock(); s.closed = true; s.mu.Unlock(); return nil }
func (s *postgresSnapshot) valid() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || !s.descriptor.ExpiresAt.After(time.Now()) {
		return relaysync.ErrSnapshotExpired
	}
	return nil
}

func (s *postgresSnapshot) OpenMetadata(ctx context.Context, partition relaysync.PartitionKey, after *relaysync.MetadataCursor) (relaysync.MetadataIterator, error) {
	if err := s.valid(); err != nil {
		return nil, err
	}
	if partition.PrefixBits != 8 {
		return nil, relaysync.ErrInvalidIdentifier
	}
	prefix := hex.EncodeToString(partition.Prefix[:1])
	query := `SELECT created_at, id FROM event WHERE deleted_by IS NULL AND floor(created_at / $1)::bigint=$2 AND lower(left(id,2))=$3`
	args := []any{relaySyncPartitionSeconds, partition.Bucket, prefix}
	if after != nil {
		query += ` AND (created_at, id) > ($4, $5)`
		args = append(args, after.CreatedAt, after.ID.String())
	}
	query += ` ORDER BY created_at, id`
	rows, err := s.queries.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &postgresMetadataIterator{rows: rows}, nil
}

func (s *postgresSnapshot) GetEvents(ctx context.Context, ids []relaysync.EventID) (relaysync.EventIterator, error) {
	if err := s.valid(); err != nil {
		return nil, err
	}
	values := make([]string, len(ids))
	for i := range ids {
		values[i] = ids[i].String()
	}
	rows, err := s.queries.db.Query(ctx, `SELECT id, pubkey, created_at, kind, tags, content, sig FROM event WHERE deleted_by IS NULL AND id = ANY($1::text[])`, values)
	if err != nil {
		return nil, err
	}
	items := make(map[string]relaysync.Event, len(ids))
	for rows.Next() {
		event, err := scanRelaySyncEvent(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		items[event.ID.String()] = event
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	result := make([]relaysync.EventItem, len(ids))
	for i, id := range ids {
		result[i] = relaysync.EventItem{ID: id}
		if event, ok := items[id.String()]; ok {
			result[i].Event = &event
		}
	}
	return &postgresEventIterator{items: result}, nil
}

type postgresMetadataIterator struct {
	rows interface {
		Next() bool
		Scan(...any) error
		Err() error
		Close()
	}
}

func (i *postgresMetadataIterator) Next(ctx context.Context) (relaysync.EventMetadata, bool, error) {
	if err := ctx.Err(); err != nil {
		return relaysync.EventMetadata{}, false, err
	}
	if !i.rows.Next() {
		return relaysync.EventMetadata{}, false, i.rows.Err()
	}
	var created int64
	var id string
	if err := i.rows.Scan(&created, &id); err != nil {
		return relaysync.EventMetadata{}, false, err
	}
	eventID, err := relaysync.ParseEventID(id)
	if err != nil {
		return relaysync.EventMetadata{}, false, err
	}
	return relaysync.EventMetadata{ID: eventID, CreatedAt: created}, true, nil
}
func (i *postgresMetadataIterator) Close() error { i.rows.Close(); return i.rows.Err() }

type postgresEventIterator struct {
	items  []relaysync.EventItem
	index  int
	closed bool
}

func (i *postgresEventIterator) Next(ctx context.Context) (relaysync.EventItem, bool, error) {
	if err := ctx.Err(); err != nil {
		return relaysync.EventItem{}, false, err
	}
	if i.closed || i.index >= len(i.items) {
		return relaysync.EventItem{}, false, nil
	}
	item := i.items[i.index]
	i.index++
	return item, true, nil
}
func (i *postgresEventIterator) Close() error { i.closed = true; return nil }

func scanRelaySyncEvent(row interface{ Scan(...any) error }) (relaysync.Event, error) {
	var id, pubkey, content, signature string
	var created int64
	var kind int32
	var tagsJSON []byte
	if err := row.Scan(&id, &pubkey, &created, &kind, &tagsJSON, &content, &signature); err != nil {
		return relaysync.Event{}, err
	}
	eventID, err := relaysync.ParseEventID(id)
	if err != nil {
		return relaysync.Event{}, err
	}
	publicKey, err := relaysync.ParsePublicKey(pubkey)
	if err != nil {
		return relaysync.Event{}, err
	}
	sig, err := relaysync.ParseSignature(signature)
	if err != nil {
		return relaysync.Event{}, err
	}
	var rawTags [][]string
	if len(tagsJSON) > 0 && string(tagsJSON) != "null" {
		if err := json.Unmarshal(tagsJSON, &rawTags); err != nil {
			return relaysync.Event{}, err
		}
	}
	tags := make([]relaysync.Tag, len(rawTags))
	for i := range rawTags {
		tags[i] = append(relaysync.Tag(nil), rawTags[i]...)
	}
	return relaysync.Event{ID: eventID, PublicKey: publicKey, Signature: sig, CreatedAt: created, Kind: uint32(kind), Tags: tags, Content: content}, nil
}
