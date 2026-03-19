DROP INDEX IF EXISTS idx_app_progress_tracking_user_id_created_at;
DROP TABLE IF EXISTS app_progress_tracking;

DROP TABLE IF EXISTS app_coaching_memory;

DROP INDEX IF EXISTS idx_app_review_sessions_input_mode;
DROP INDEX IF EXISTS idx_app_review_sessions_status;
DROP INDEX IF EXISTS idx_app_review_sessions_user_id_created_at;
DROP TABLE IF EXISTS app_review_sessions;

ALTER TABLE app_usage_tracking
    DROP CONSTRAINT IF EXISTS app_usage_tracking_usage_type_check;

ALTER TABLE app_usage_tracking
    ADD CONSTRAINT app_usage_tracking_usage_type_check
    CHECK (usage_type IN ('voice_minutes', 'session_count', 'voice_addon'));
