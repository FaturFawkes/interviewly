DROP INDEX IF EXISTS uq_app_usage_tracking_session_type_period;
DROP INDEX IF EXISTS idx_app_usage_tracking_usage_type;
DROP INDEX IF EXISTS idx_app_usage_tracking_user_period;
DROP TABLE IF EXISTS app_usage_tracking;

DROP INDEX IF EXISTS idx_app_subscriptions_period_end;
DROP INDEX IF EXISTS idx_app_subscriptions_status;
DROP INDEX IF EXISTS idx_app_subscriptions_user_id;
DROP INDEX IF EXISTS uq_app_subscriptions_user_period;
DROP TABLE IF EXISTS app_subscriptions;
