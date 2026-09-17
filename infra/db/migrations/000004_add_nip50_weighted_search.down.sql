DROP INDEX IF EXISTS public.content_search_idx;

ALTER TABLE public.event
    DROP COLUMN IF EXISTS content_search;

ALTER TABLE public.event
    ADD COLUMN content_search TSVECTOR GENERATED ALWAYS AS (
        to_tsvector('simple'::regconfig, content)
    ) STORED;

CREATE INDEX content_search_idx
    ON public.event USING gin (content_search);

DROP FUNCTION IF EXISTS public.tags_to_descriptions(jsonb);

ANALYZE public.event;
