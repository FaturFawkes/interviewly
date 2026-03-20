ALTER TABLE app_subscriptions
    ADD COLUMN IF NOT EXISTS total_text_requests INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS used_text_requests INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_jd_limit INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS used_jd_parses INTEGER NOT NULL DEFAULT 0;

ALTER TABLE app_subscriptions
    DROP CONSTRAINT IF EXISTS app_subscriptions_text_jd_non_negative_check;

ALTER TABLE app_subscriptions
    ADD CONSTRAINT app_subscriptions_text_jd_non_negative_check
    CHECK (
        total_text_requests >= 0
        AND used_text_requests >= 0
        AND total_jd_limit >= 0
        AND used_jd_parses >= 0
    );

ALTER TABLE app_usage_tracking
    DROP CONSTRAINT IF EXISTS app_usage_tracking_usage_type_check;

ALTER TABLE app_usage_tracking
    ADD CONSTRAINT app_usage_tracking_usage_type_check
    CHECK (usage_type IN ('voice_minutes', 'session_count', 'text_request', 'jd_parse', 'voice_topup'));

CREATE TABLE IF NOT EXISTS app_voice_wallet_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    source TEXT NOT NULL,
    purchased_minutes INTEGER NOT NULL,
    consumed_minutes INTEGER NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_app_voice_wallet_entries_minutes_non_negative CHECK (purchased_minutes >= 0 AND consumed_minutes >= 0),
    CONSTRAINT chk_app_voice_wallet_entries_consumed_lte_purchased CHECK (consumed_minutes <= purchased_minutes)
);

CREATE INDEX IF NOT EXISTS idx_app_voice_wallet_entries_user_created_at
    ON app_voice_wallet_entries(user_id, created_at ASC);

CREATE TABLE IF NOT EXISTS app_billing_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider TEXT NOT NULL,
    event_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    processing_status TEXT NOT NULL DEFAULT 'processed' CHECK (processing_status IN ('processed', 'ignored', 'failed')),
    error_message TEXT,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_app_billing_events_provider_event
    ON app_billing_events(provider, event_id);

CREATE TABLE IF NOT EXISTS app_voice_topup_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    package_code TEXT NOT NULL,
    purchased_minutes INTEGER NOT NULL,
    amount_idr INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'failed', 'canceled')),
    provider_checkout_session_id TEXT,
    provider_payment_intent_id TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_app_voice_topup_orders_numeric_non_negative CHECK (purchased_minutes >= 0 AND amount_idr >= 0),
    CONSTRAINT chk_app_voice_topup_orders_package_code CHECK (package_code IN ('voice_topup_10', 'voice_topup_30'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_app_voice_topup_orders_checkout_session
    ON app_voice_topup_orders(provider, provider_checkout_session_id)
    WHERE provider_checkout_session_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_app_voice_topup_orders_payment_intent
    ON app_voice_topup_orders(provider, provider_payment_intent_id)
    WHERE provider_payment_intent_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_app_voice_topup_orders_user_created_at
    ON app_voice_topup_orders(user_id, created_at DESC);

ALTER TABLE app_practice_sessions
    ADD COLUMN IF NOT EXISTS last_activity_at TIMESTAMPTZ;

UPDATE app_practice_sessions
SET last_activity_at = COALESCE(last_activity_at, created_at)
WHERE last_activity_at IS NULL;

ALTER TABLE app_practice_sessions
    ALTER COLUMN last_activity_at SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_app_practice_sessions_status_last_activity
    ON app_practice_sessions(status, last_activity_at ASC);
