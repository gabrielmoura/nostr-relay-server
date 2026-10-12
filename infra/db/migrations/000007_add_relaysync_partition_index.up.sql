-- Accelerate bounded RelaySync manifest pages. The predicate mirrors the
-- snapshot adapter and deliberately excludes malformed legacy IDs without
-- changing those rows.
CREATE INDEX IF NOT EXISTS event_relaysync_partition_idx
    ON public.event (
        ((created_at::bigint / 2592000)::bigint),
        (lower(left(id, 2))),
        created_at,
        id
    )
    WHERE deleted_by IS NULL
      AND length(id) = 64
      AND id ~ '^[0-9A-Fa-f]{64}$';
