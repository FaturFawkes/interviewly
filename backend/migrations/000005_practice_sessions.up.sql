CREATE TABLE IF NOT EXISTS app_practice_sessions (
    id UUID PRIMARY KEY,
    user_id TEXT NOT NULL,
    resume_id UUID NOT NULL,
    job_parse_id UUID NOT NULL,
    question_ids UUID[] NOT NULL DEFAULT '{}',
    status VARCHAR(20) NOT NULL CHECK (status IN ('active', 'completed', 'abandoned')),
    score INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_app_practice_sessions_user_id ON app_practice_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_app_practice_sessions_status ON app_practice_sessions(status);
CREATE INDEX IF NOT EXISTS idx_app_practice_sessions_created_at ON app_practice_sessions(created_at DESC);
