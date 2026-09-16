-- Forward migration: persist relay-generated content spam flags for admin review.
CREATE TABLE content_spam_flags (
  event_id TEXT PRIMARY KEY REFERENCES event(id) ON DELETE CASCADE,
  flagged_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
