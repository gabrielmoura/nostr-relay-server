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
                         content_search TSVECTOR GENERATED ALWAYS AS ( to_tsvector( 'portuguese', CONTENT ) ) STORED
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
                          nip05 TEXT
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
                              related_ids VARCHAR ( 60 ) [] -- Array de strings com até 60 caracteres

);
-- Índices para a tabela banned_users
CREATE INDEX idx_banned_users_user_id ON banned_users ( user_id );
CREATE INDEX idx_banned_users_id ON banned_users ( ID );

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
