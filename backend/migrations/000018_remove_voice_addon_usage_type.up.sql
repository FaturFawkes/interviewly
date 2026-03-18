ALTER TABLE app_usage_tracking
    DROP CONSTRAINT IF EXISTS app_usage_tracking_usage_type_check;

DELETE FROM app_usage_tracking
WHERE usage_type = 'voice_addon';

ALTER TABLE app_usage_tracking
    ADD CONSTRAINT app_usage_tracking_usage_type_check
    CHECK (usage_type IN ('voice_minutes', 'session_count'));
