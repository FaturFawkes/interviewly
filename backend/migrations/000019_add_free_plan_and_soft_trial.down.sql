ALTER TABLE app_subscriptions
    DROP CONSTRAINT IF EXISTS app_subscriptions_trial_window_check;

ALTER TABLE app_subscriptions
    DROP CONSTRAINT IF EXISTS app_subscriptions_trial_voice_non_negative_check;

ALTER TABLE app_subscriptions
    DROP CONSTRAINT IF EXISTS app_subscriptions_trial_plan_id_check;

ALTER TABLE app_subscriptions
    DROP COLUMN IF EXISTS trial_consumed_voice_minutes,
    DROP COLUMN IF EXISTS trial_voice_bonus_minutes,
    DROP COLUMN IF EXISTS trial_plan_id,
    DROP COLUMN IF EXISTS trial_used_at,
    DROP COLUMN IF EXISTS trial_ends_at,
    DROP COLUMN IF EXISTS trial_started_at;

ALTER TABLE app_subscriptions
    DROP CONSTRAINT IF EXISTS app_subscriptions_plan_id_check;

ALTER TABLE app_subscriptions
    ADD CONSTRAINT app_subscriptions_plan_id_check
    CHECK (plan_id IN ('starter', 'pro', 'elite'));
