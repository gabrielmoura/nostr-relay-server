CREATE TABLE IF NOT EXISTS public.relay_sync_checkpoint (
    peer_id TEXT NOT NULL,
    scope_name TEXT NOT NULL,
    scope_version INTEGER NOT NULL,
    scope_hash BYTEA NOT NULL,
    journal_epoch BIGINT NOT NULL,
    journal_sequence BIGINT NOT NULL,
    decision_id TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (peer_id, scope_name, scope_version, scope_hash),
    CONSTRAINT relay_sync_checkpoint_scope_hash_length CHECK (octet_length(scope_hash) = 32)
);
