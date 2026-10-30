CREATE TABLE IF NOT EXISTS public.cordn_key_package (
    key_package_ref TEXT PRIMARY KEY,
    stable_pubkey TEXT NOT NULL,
    key_package BYTEA NOT NULL,
    publication_event JSONB NOT NULL,
    is_last_resort BOOLEAN NOT NULL DEFAULT FALSE,
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS cordn_key_package_identity_idx
    ON public.cordn_key_package (stable_pubkey, published_at, key_package_ref);

CREATE TABLE IF NOT EXISTS public.cordn_pending_welcome (
    target_pubkey TEXT NOT NULL,
    key_package_ref TEXT NOT NULL,
    welcome BYTEA NOT NULL,
    join_after_cursor BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (target_pubkey, key_package_ref, created_at)
);

CREATE INDEX IF NOT EXISTS cordn_pending_welcome_target_idx
    ON public.cordn_pending_welcome (target_pubkey, created_at);

CREATE TABLE IF NOT EXISTS public.cordn_pending_join_request (
    gid TEXT NOT NULL,
    requester_pubkey TEXT NOT NULL,
    key_package_ref TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (gid, requester_pubkey, key_package_ref, created_at)
);

CREATE INDEX IF NOT EXISTS cordn_pending_join_request_gid_idx
    ON public.cordn_pending_join_request (gid, created_at);

CREATE TABLE IF NOT EXISTS public.cordn_group_cursor (
    gid TEXT PRIMARY KEY,
    next_cursor BIGINT NOT NULL CHECK (next_cursor > 0)
);

CREATE TABLE IF NOT EXISTS public.cordn_group_message (
    gid TEXT NOT NULL,
    cursor BIGINT NOT NULL CHECK (cursor > 0),
    payload BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (gid, cursor)
);

CREATE INDEX IF NOT EXISTS cordn_group_message_gid_cursor_idx
    ON public.cordn_group_message (gid, cursor);
