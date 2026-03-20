DROP INDEX IF EXISTS idx_app_practice_sessions_status_last_activity;

ALTER TABLE app_practice_sessions
    DROP COLUMN IF EXISTS last_activity_at;

DROP INDEX IF EXISTS idx_app_voice_topup_orders_user_created_at;
DROP INDEX IF EXISTS uq_app_voice_topup_orders_payment_intent;
DROP INDEX IF EXISTS uq_app_voice_topup_orders_checkout_session;
DROP TABLE IF EXISTS app_voice_topup_orders;

DROP INDEX IF EXISTS uq_app_billing_events_provider_event;
DROP TABLE IF EXISTS app_billing_events;

DROP INDEX IF EXISTS idx_app_voice_wallet_entries_user_created_at;
DROP TABLE IF EXISTS app_voice_wallet_entries;

ALTER TABLE app_usage_tracking
    DROP CONSTRAINT IF EXISTS app_usage_tracking_usage_type_check;

ALTER TABLE app_usage_tracking
    ADD CONSTRAINT app_usage_tracking_usage_type_check
    CHECK (usage_type IN ('voice_minutes', 'session_count'));

ALTER TABLE app_subscriptions
    DROP CONSTRAINT IF EXISTS app_subscriptions_text_jd_non_negative_check;

ALTER TABLE app_subscriptions
    DROP COLUMN IF EXISTS used_jd_parses,
    DROP COLUMN IF EXISTS total_jd_limit,
    DROP COLUMN IF EXISTS used_text_requests,
    DROP COLUMN IF EXISTS total_text_requests;
