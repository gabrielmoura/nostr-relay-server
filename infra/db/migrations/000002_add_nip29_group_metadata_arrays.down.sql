ALTER TABLE public.nip29_groups
    DROP COLUMN IF EXISTS geohashes,
    DROP COLUMN IF EXISTS topics;
