ALTER TABLE app_subscriptions
    DROP CONSTRAINT IF EXISTS app_subscriptions_plan_id_check;

ALTER TABLE app_subscriptions
    ADD CONSTRAINT app_subscriptions_plan_id_check
    CHECK (plan_id IN ('free', 'starter', 'pro', 'elite'));

ALTER TABLE app_subscriptions
    ADD COLUMN IF NOT EXISTS trial_started_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS trial_ends_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS trial_used_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS trial_plan_id TEXT,
    ADD COLUMN IF NOT EXISTS trial_voice_bonus_minutes INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS trial_consumed_voice_minutes INTEGER NOT NULL DEFAULT 0;

ALTER TABLE app_subscriptions
    DROP CONSTRAINT IF EXISTS app_subscriptions_trial_plan_id_check;

ALTER TABLE app_subscriptions
    ADD CONSTRAINT app_subscriptions_trial_plan_id_check
    CHECK (trial_plan_id IS NULL OR trial_plan_id IN ('pro', 'elite'));

ALTER TABLE app_subscriptions
    DROP CONSTRAINT IF EXISTS app_subscriptions_trial_voice_non_negative_check;

ALTER TABLE app_subscriptions
    ADD CONSTRAINT app_subscriptions_trial_voice_non_negative_check
    CHECK (trial_voice_bonus_minutes >= 0 AND trial_consumed_voice_minutes >= 0);

ALTER TABLE app_subscriptions
    DROP CONSTRAINT IF EXISTS app_subscriptions_trial_window_check;

ALTER TABLE app_subscriptions
    ADD CONSTRAINT app_subscriptions_trial_window_check
    CHECK (
        (trial_started_at IS NULL AND trial_ends_at IS NULL)
        OR (trial_started_at IS NOT NULL AND trial_ends_at IS NOT NULL AND trial_ends_at > trial_started_at)
    );
