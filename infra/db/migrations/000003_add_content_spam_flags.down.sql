-- Reverse migration: remove relay-generated content spam flags.
DROP TABLE IF EXISTS content_spam_flags;
