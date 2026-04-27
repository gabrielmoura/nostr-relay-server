package db

import "context"

const upsertRelayMetadata = `
INSERT INTO nip86_relay_metadata (relay_url, name, description, updated_by, updated_at)
VALUES ($1::text, $2::text, $3::text, $4::varchar, NOW())
ON CONFLICT (relay_url) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    updated_by = EXCLUDED.updated_by,
    updated_at = NOW();
`

const getRelayMetadata = `
SELECT relay_url, COALESCE(name, ''), COALESCE(description, ''), updated_by, updated_at
FROM nip86_relay_metadata
WHERE relay_url = $1::text
LIMIT 1
`

func (q *Queries) UpsertRelayMetadata(ctx context.Context, relayURL, name, description, updatedBy string) error {
	_, err := q.db.Exec(ctx, upsertRelayMetadata, relayURL, name, description, updatedBy)
	return err
}

func (q *Queries) GetRelayMetadata(ctx context.Context, relayURL string) (NIP86RelayMetadataRecord, bool, error) {
	var item NIP86RelayMetadataRecord
	err := q.db.QueryRow(ctx, getRelayMetadata, relayURL).Scan(
		&item.RelayURL,
		&item.Name,
		&item.Description,
		&item.UpdatedBy,
		&item.UpdatedAt,
	)
	if isNotFound(err) {
		return NIP86RelayMetadataRecord{}, false, nil
	}
	if err != nil {
		return NIP86RelayMetadataRecord{}, false, err
	}
	return item, true, nil
}
