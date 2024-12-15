--- Functions
CREATE OR REPLACE FUNCTION tags_to_tagvalues(jsonb) RETURNS text[]
    AS 'SELECT array_agg(t->>1) FROM (SELECT jsonb_array_elements($1) AS t)s WHERE length(t->>0) = 1;'
    LANGUAGE SQL
    IMMUTABLE
    RETURNS NULL ON NULL INPUT;

--- Tables
CREATE TABLE IF NOT EXISTS event (
                                     id text NOT NULL,
                                     pubkey text NOT NULL,
                                     created_at integer NOT NULL,
                                     kind integer NOT NULL,
                                     tags jsonb NOT NULL,
                                     content text NOT NULL,
                                     sig text NOT NULL,

                                     tagvalues text[] GENERATED ALWAYS AS (tags_to_tagvalues(tags)) STORED
    );

--- Indexes
CREATE UNIQUE INDEX IF NOT EXISTS ididx ON event USING btree (id text_pattern_ops);
CREATE INDEX IF NOT EXISTS pubkeyprefix ON event USING btree (pubkey text_pattern_ops);
CREATE INDEX IF NOT EXISTS timeidx ON event (created_at DESC);
CREATE INDEX IF NOT EXISTS kindidx ON event (kind);
CREATE INDEX IF NOT EXISTS kindtimeidx ON event(kind,created_at DESC);
CREATE INDEX IF NOT EXISTS arbitrarytagvalues ON event USING gin (tagvalues);

-- Tabela para armazenar perfis
CREATE TABLE profiles (
                          id BIGSERIAL PRIMARY KEY,
                          public_key TEXT NOT NULL,
                          name TEXT NOT NULL,
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
CREATE INDEX idx_profiles_name ON profiles(name);
CREATE INDEX idx_profiles_nip05 ON profiles(nip05);
CREATE INDEX idx_profiles_display_name ON profiles(display_name);

-- Tabela para armazenar usuários banidos
CREATE TABLE banned_users (
                              id BIGSERIAL PRIMARY KEY,
                              user_id BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
                              reason TEXT NOT NULL,
                              related_ids VARCHAR(60)[] -- Array de strings com até 60 caracteres
);

-- Índices para a tabela banned_users
CREATE INDEX idx_banned_users_user_id ON banned_users(user_id);
CREATE INDEX idx_banned_users_id ON banned_users(id);
