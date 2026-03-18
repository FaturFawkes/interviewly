-- No-op rollback migration to preserve historical chain continuity.
DO $$
BEGIN
    NULL;
END $$;
