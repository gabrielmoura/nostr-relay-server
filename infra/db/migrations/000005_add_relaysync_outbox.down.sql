DROP TRIGGER IF EXISTS relay_sync_event_outbox_trigger ON public.event;
DROP FUNCTION IF EXISTS public.relay_sync_enqueue_outbox();
DROP TABLE IF EXISTS public.relay_sync_journal;
DROP TABLE IF EXISTS public.relay_sync_outbox;
