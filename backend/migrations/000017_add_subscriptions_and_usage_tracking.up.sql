CREATE TABLE IF NOT EXISTS app_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    plan_id TEXT NOT NULL CHECK (plan_id IN ('starter', 'pro', 'elite')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'past_due', 'canceled')),
    total_voice_minutes INTEGER NOT NULL,
    used_voice_minutes INTEGER NOT NULL DEFAULT 0,
    total_sessions_limit INTEGER NOT NULL,
    used_sessions INTEGER NOT NULL DEFAULT 0,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_period_valid CHECK (period_end > period_start),
    CONSTRAINT chk_voice_minutes_non_negative CHECK (total_voice_minutes >= 0 AND used_voice_minutes >= 0),
    CONSTRAINT chk_sessions_non_negative CHECK (used_sessions >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_app_subscriptions_user_period
    ON app_subscriptions(user_id, period_start, period_end);

CREATE INDEX IF NOT EXISTS idx_app_subscriptions_user_id
    ON app_subscriptions(user_id);

CREATE INDEX IF NOT EXISTS idx_app_subscriptions_status
    ON app_subscriptions(status);

CREATE INDEX IF NOT EXISTS idx_app_subscriptions_period_end
    ON app_subscriptions(period_end);

CREATE TABLE IF NOT EXISTS app_usage_tracking (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    session_id UUID,
    usage_type TEXT NOT NULL CHECK (usage_type IN ('voice_minutes', 'session_count', 'voice_addon')),
    consumed_minutes INTEGER NOT NULL DEFAULT 0,
    consumed_sessions INTEGER NOT NULL DEFAULT 0,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_usage_minutes_non_negative CHECK (consumed_minutes >= 0),
    CONSTRAINT chk_usage_sessions_non_negative CHECK (consumed_sessions >= 0),
    CONSTRAINT chk_usage_period_valid CHECK (period_end > period_start)
);

CREATE INDEX IF NOT EXISTS idx_app_usage_tracking_user_period
    ON app_usage_tracking(user_id, period_start, period_end);

CREATE INDEX IF NOT EXISTS idx_app_usage_tracking_usage_type
    ON app_usage_tracking(usage_type);

CREATE UNIQUE INDEX IF NOT EXISTS uq_app_usage_tracking_session_type_period
    ON app_usage_tracking(user_id, session_id, usage_type, period_start)
    WHERE session_id IS NOT NULL;
