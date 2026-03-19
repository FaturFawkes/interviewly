ALTER TABLE app_usage_tracking
    DROP CONSTRAINT IF EXISTS app_usage_tracking_usage_type_check;

ALTER TABLE app_usage_tracking
    ADD CONSTRAINT app_usage_tracking_usage_type_check
    CHECK (usage_type IN ('voice_minutes', 'session_count', 'voice_addon', 'review_count', 'review_voice_minutes'));

CREATE TABLE IF NOT EXISTS app_review_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    session_type TEXT NOT NULL CHECK (session_type IN ('review', 'recovery')),
    input_mode TEXT NOT NULL CHECK (input_mode IN ('text', 'voice')),
    input_text TEXT NOT NULL DEFAULT '',
    voice_url TEXT NOT NULL DEFAULT '',
    transcript_text TEXT NOT NULL DEFAULT '',
    role_target TEXT NOT NULL DEFAULT '',
    company_target TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed', 'abandoned')),
    ai_feedback JSONB NOT NULL DEFAULT '{}'::jsonb,
    score INTEGER NOT NULL DEFAULT 0 CHECK (score >= 0 AND score <= 100),
    communication_score INTEGER NOT NULL DEFAULT 0 CHECK (communication_score >= 0 AND communication_score <= 100),
    structure_score INTEGER NOT NULL DEFAULT 0 CHECK (structure_score >= 0 AND structure_score <= 100),
    confidence_score INTEGER NOT NULL DEFAULT 0 CHECK (confidence_score >= 0 AND confidence_score <= 100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_app_review_sessions_user_id_created_at
    ON app_review_sessions(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_app_review_sessions_status
    ON app_review_sessions(status);

CREATE INDEX IF NOT EXISTS idx_app_review_sessions_input_mode
    ON app_review_sessions(input_mode);

CREATE TABLE IF NOT EXISTS app_coaching_memory (
    user_id TEXT PRIMARY KEY,
    target_role TEXT NOT NULL DEFAULT '',
    strengths TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    weaknesses TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    preferred_language TEXT NOT NULL DEFAULT 'en',
    last_summary TEXT NOT NULL DEFAULT '',
    focus_areas TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    next_actions TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS app_progress_tracking (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    review_session_id UUID REFERENCES app_review_sessions(id) ON DELETE CASCADE,
    communication_score INTEGER NOT NULL CHECK (communication_score >= 0 AND communication_score <= 100),
    structure_score INTEGER NOT NULL CHECK (structure_score >= 0 AND structure_score <= 100),
    confidence_score INTEGER NOT NULL CHECK (confidence_score >= 0 AND confidence_score <= 100),
    overall_score INTEGER NOT NULL CHECK (overall_score >= 0 AND overall_score <= 100),
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_app_progress_tracking_user_id_created_at
    ON app_progress_tracking(user_id, created_at DESC);
