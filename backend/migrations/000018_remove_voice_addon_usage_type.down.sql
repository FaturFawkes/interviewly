ALTER TABLE app_usage_tracking
    DROP CONSTRAINT IF EXISTS app_usage_tracking_usage_type_check;

ALTER TABLE app_usage_tracking
    ADD CONSTRAINT app_usage_tracking_usage_type_check
    CHECK (usage_type IN ('voice_minutes', 'session_count', 'voice_addon'));
