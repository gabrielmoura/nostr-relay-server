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