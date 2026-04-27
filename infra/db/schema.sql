--- Functions
CREATE
    OR REPLACE FUNCTION tags_to_tagvalues ( JSONB ) RETURNS TEXT [] AS 'SELECT array_agg(t->>1) FROM (SELECT jsonb_array_elements($1) AS t)s WHERE length(t->>0) = 1;' LANGUAGE SQL IMMUTABLE RETURNS NULL ON NULL INPUT;
--- Tables
CREATE TABLE
    IF
    NOT EXISTS event (
                         ID TEXT NOT NULL,
                         pubkey TEXT NOT NULL,
                         created_at INTEGER NOT NULL,
                         kind INTEGER NOT NULL,
                         tags JSONB NOT NULL,
                         CONTENT TEXT NOT NULL,
                         sig TEXT NOT NULL,
                         tagvalues TEXT [] GENERATED ALWAYS AS ( tags_to_tagvalues ( tags ) ) STORED,
                         content_search TSVECTOR GENERATED ALWAYS AS ( to_tsvector( 'portuguese', CONTENT ) ) STORED,
                         deleted_by varchar(64) DEFAULT NULL
);
--- Indexes
CREATE UNIQUE INDEX
    IF
    NOT EXISTS ididx ON event USING btree ( ID text_pattern_ops );
CREATE INDEX
    IF
    NOT EXISTS pubkeyprefix ON event USING btree ( pubkey text_pattern_ops );
CREATE INDEX
    IF
    NOT EXISTS timeidx ON event ( created_at DESC );
CREATE INDEX
    IF
    NOT EXISTS kindidx ON event ( kind );
CREATE INDEX
    IF
    NOT EXISTS kindtimeidx ON event ( kind, created_at DESC );
CREATE INDEX
    IF
    NOT EXISTS arbitrarytagvalues ON event USING gin ( tagvalues );
CREATE INDEX
    IF
    NOT EXISTS content_search_idx ON event USING gin ( content_search );
CREATE INDEX IF NOT EXISTS idx_event_created_at_id ON event (created_at, id);
-- Tabela para armazenar perfis
CREATE TABLE profiles (
                          ID BIGSERIAL PRIMARY KEY,
                          public_key VARCHAR(64) NOT NULL UNIQUE CHECK ( length(public_key) = 64 ),
                          NAME TEXT NOT NULL,
                          about TEXT,
                          picture TEXT,
                          bot BOOLEAN DEFAULT FALSE,
                          banner TEXT,
                          website TEXT,
                          display_name TEXT,
                          lud16 TEXT,
                          pronouns TEXT,
                          nip05 TEXT,
                          enable_store_files BOOLEAN default false,
                           enable_nip05 BOOLEAN DEFAULT FALSE

);

CREATE TABLE IF NOT EXISTS nip05_identities (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    public_key VARCHAR(64) NOT NULL UNIQUE REFERENCES profiles(public_key) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_nip05_identities_public_key ON nip05_identities (public_key);
CREATE INDEX IF NOT EXISTS idx_nip05_identities_name ON nip05_identities (name);

-- Índices para a tabela profiles
CREATE INDEX idx_profiles_name ON profiles ( NAME );
CREATE INDEX idx_profiles_nip05 ON profiles ( nip05 );
CREATE INDEX idx_profiles_display_name ON profiles ( display_name );
-- Tabela para armazenar usuários banidos
CREATE TABLE banned_users (
                               ID BIGSERIAL PRIMARY KEY,
                               user_id BIGINT NOT NULL REFERENCES profiles ( ID ) ON DELETE CASCADE,
                               reason TEXT NOT NULL,
                               related_ids VARCHAR ( 60 ) [], -- Array de strings com até 60 caracteres
                               created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()

);
ALTER TABLE banned_users ADD COLUMN IF NOT EXISTS created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW();
-- Índices para a tabela banned_users
CREATE INDEX idx_banned_users_user_id ON banned_users ( user_id );
CREATE INDEX idx_banned_users_id ON banned_users ( ID );
CREATE INDEX IF NOT EXISTS idx_banned_users_created_at ON banned_users ( created_at DESC );

-- Table para armazenar metadados de arquivos
CREATE TABLE objects (
                         hash VARCHAR ( 64 ) NOT NULL PRIMARY KEY,
                         created_at TIMESTAMP WITH TIME ZONE NOT NULL,
                         mime_type VARCHAR ( 255 ),
                         SIZE BIGINT,
                         blocked BOOLEAN,
                         expires_at TIMESTAMP WITH TIME ZONE,
                         blocked_by_reason TEXT,
                         public_key VARCHAR ( 64 ) NOT NULL,
                         tags JSONB
);
-- Índices para a tabela objects
CREATE INDEX idx_objects_mime_type ON objects ( mime_type );
CREATE INDEX idx_objects_blocked ON objects ( blocked );



-- Para queries de deletion events
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_deletions
    ON event (created_at DESC, id)
    WHERE deleted_by IS NOT NULL;

-- Covering index para author queries (evita table access)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_covering_author
    ON event (pubkey, created_at DESC)
    INCLUDE (kind, content, tags, sig);

-- Partial para eventos recentes (sem função - usa constante estática)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_recent
    ON event (created_at DESC, id)
    WHERE created_at > 1735689600;  -- Unix timestamp de 24h atrás

-- Queries por author + kind (muito comum)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_author_kind
    ON event (pubkey, kind, created_at DESC);

-- Covering index para evitar table access
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_covering
    ON event (pubkey, created_at DESC)
    INCLUDE (kind, content, tags, sig);

CREATE TABLE IF NOT EXISTS nip29_roles (
    role_id SERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    description TEXT
);

CREATE INDEX IF NOT EXISTS idx_roles_name ON nip29_roles (name);

CREATE TABLE IF NOT EXISTS nip29_groups (
    relay TEXT NOT NULL,
    group_id TEXT NOT NULL,
    name VARCHAR(255) NOT NULL,
    picture TEXT,
    about TEXT,
    private BOOLEAN NOT NULL DEFAULT FALSE,
    closed BOOLEAN NOT NULL DEFAULT FALSE,
    last_metadata_update TIMESTAMPTZ NOT NULL,
    last_admins_update TIMESTAMPTZ NOT NULL,
    last_members_update TIMESTAMPTZ NOT NULL,
    last_roles_update TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (relay, group_id)
);

ALTER TABLE nip29_groups ADD COLUMN IF NOT EXISTS restricted BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE nip29_groups ADD COLUMN IF NOT EXISTS hidden BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE nip29_groups ADD COLUMN IF NOT EXISTS created_by VARCHAR(64);
ALTER TABLE nip29_groups ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE nip29_groups ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE nip29_groups ADD COLUMN IF NOT EXISTS min_pow INTEGER NOT NULL DEFAULT 0;
ALTER TABLE nip29_groups ADD COLUMN IF NOT EXISTS require_moderation_timeline_ref BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE nip29_groups ADD COLUMN IF NOT EXISTS min_timeline_references INTEGER NOT NULL DEFAULT 0;
ALTER TABLE nip29_groups ADD COLUMN IF NOT EXISTS timeline_recent_window INTEGER NOT NULL DEFAULT 50;
ALTER TABLE nip29_groups ADD COLUMN IF NOT EXISTS allow_late_publication BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_groups_name ON nip29_groups (name);
CREATE INDEX IF NOT EXISTS idx_groups_last_metadata_update ON nip29_groups (last_metadata_update);
CREATE INDEX IF NOT EXISTS idx_groups_last_members_update ON nip29_groups (last_members_update);

CREATE TABLE IF NOT EXISTS nip29_group_roles (
    relay TEXT NOT NULL,
    group_id TEXT NOT NULL,
    role_id INTEGER NOT NULL REFERENCES nip29_roles(role_id) ON DELETE CASCADE,
    PRIMARY KEY (relay, group_id, role_id),
    FOREIGN KEY (relay, group_id) REFERENCES nip29_groups(relay, group_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS nip29_group_members (
    relay TEXT NOT NULL,
    group_id TEXT NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    role_id INTEGER NOT NULL REFERENCES nip29_roles(role_id) ON DELETE CASCADE,
    banned BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (relay, group_id, user_id, role_id, banned),
    FOREIGN KEY (relay, group_id) REFERENCES nip29_groups(relay, group_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_group_members_user_id ON nip29_group_members (user_id);
CREATE INDEX IF NOT EXISTS idx_group_members_lookup ON nip29_group_members (relay, group_id, user_id) WHERE banned = FALSE;

CREATE TABLE IF NOT EXISTS nip29_group_bans (
    relay TEXT NOT NULL,
    group_id TEXT NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    reason TEXT,
    created_by VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (relay, group_id, user_id),
    FOREIGN KEY (relay, group_id) REFERENCES nip29_groups(relay, group_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_group_bans_user_lookup ON nip29_group_bans (relay, group_id, user_id);

CREATE TABLE IF NOT EXISTS nip29_group_invites (
    relay TEXT NOT NULL,
    group_id TEXT NOT NULL,
    code TEXT NOT NULL,
    created_by VARCHAR(64) NOT NULL,
    max_uses INTEGER NOT NULL DEFAULT 1,
    uses INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    PRIMARY KEY (relay, group_id, code),
    FOREIGN KEY (relay, group_id) REFERENCES nip29_groups(relay, group_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_group_invites_expires_at ON nip29_group_invites (expires_at);

CREATE TABLE IF NOT EXISTS nip86_allowed_pubkeys (
    pubkey VARCHAR(64) PRIMARY KEY,
    reason TEXT,
    created_by VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_nip86_allowed_pubkeys_updated_at ON nip86_allowed_pubkeys (updated_at DESC);

CREATE TABLE IF NOT EXISTS nip86_banned_events (
    event_id VARCHAR(64) PRIMARY KEY,
    reason TEXT,
    created_by VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_nip86_banned_events_updated_at ON nip86_banned_events (updated_at DESC);

CREATE TABLE IF NOT EXISTS nip86_blocked_ips (
    ip INET PRIMARY KEY,
    reason TEXT,
    created_by VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_nip86_blocked_ips_updated_at ON nip86_blocked_ips (updated_at DESC);

CREATE TABLE IF NOT EXISTS nip86_relay_metadata (
    relay_url TEXT PRIMARY KEY,
    name TEXT,
    description TEXT,
    updated_by VARCHAR(64) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
