CREATE OR REPLACE FUNCTION public.tags_to_descriptions(input jsonb)
    RETURNS text
    LANGUAGE sql
    IMMUTABLE
    STRICT
AS $$
SELECT COALESCE(string_agg(tag ->> 1, ' '), '')
FROM jsonb_array_elements(
    CASE
        WHEN jsonb_typeof(input) = 'array' THEN input
        ELSE '[]'::jsonb
    END
) AS tag
WHERE jsonb_typeof(tag) = 'array'
  AND lower(tag ->> 0) = 'description'
  AND COALESCE(tag ->> 1, '') <> '';
$$;

DROP INDEX IF EXISTS public.content_search_idx;

ALTER TABLE public.event
    DROP COLUMN IF EXISTS content_search;

ALTER TABLE public.event
    ADD COLUMN content_search TSVECTOR GENERATED ALWAYS AS (
        setweight(to_tsvector('simple'::regconfig, content), 'A') ||
        setweight(to_tsvector('simple'::regconfig, public.tags_to_descriptions(tags)), 'B')
    ) STORED;

CREATE INDEX content_search_idx
    ON public.event USING gin (content_search);

ANALYZE public.event;
