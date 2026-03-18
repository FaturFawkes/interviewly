-- No-op rollback migration to restore missing version 13 in migration history.
DO $$
BEGIN
    NULL;
END $$;
