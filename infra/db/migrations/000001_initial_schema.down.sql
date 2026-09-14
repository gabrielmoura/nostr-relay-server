DROP TABLE IF EXISTS public.nip86_relay_metadata;
DROP TABLE IF EXISTS public.nip86_blocked_ips;
DROP TABLE IF EXISTS public.nip86_banned_events;
DROP TABLE IF EXISTS public.nip86_allowed_pubkeys;

DROP TABLE IF EXISTS public.nip29_group_hierarchy;
DROP TABLE IF EXISTS public.nip29_group_pins;
DROP TABLE IF EXISTS public.nip29_group_invites;
DROP TABLE IF EXISTS public.nip29_group_bans;
DROP TABLE IF EXISTS public.nip29_group_members;
DROP TABLE IF EXISTS public.nip29_group_roles;
DROP TABLE IF EXISTS public.nip29_groups;
DROP TABLE IF EXISTS public.nip29_roles;

DROP TABLE IF EXISTS public.blossom_audit_log;
DROP TABLE IF EXISTS public.blossom_review_reports;
DROP TABLE IF EXISTS public.blossom_plan_assignments;
DROP TABLE IF EXISTS public.blossom_plans;
DROP TABLE IF EXISTS public.blossom_server_policy;
DROP TABLE IF EXISTS public.blossom_pubkey_quotas;
DROP TABLE IF EXISTS public.blossom_objects_admin;
DROP TABLE IF EXISTS public.objects;

DROP TABLE IF EXISTS public.banned_users;
DROP TABLE IF EXISTS public.nip05_identities;
DROP TABLE IF EXISTS public.nip05;

DROP TRIGGER IF EXISTS trg_event_expiration ON public.event;
DROP TABLE IF EXISTS public.event;
DROP TABLE IF EXISTS public.profiles;

DROP FUNCTION IF EXISTS public.nostr_expiration_submission();
DROP FUNCTION IF EXISTS public.tags_to_tagvalues(jsonb);
