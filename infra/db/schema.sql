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
