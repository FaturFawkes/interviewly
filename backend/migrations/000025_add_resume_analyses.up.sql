CREATE TABLE IF NOT EXISTS app_resume_analyses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    resume_id UUID REFERENCES app_resumes(id) ON DELETE SET NULL,
    content_hash TEXT NOT NULL,
    model TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    response TEXT NOT NULL DEFAULT '',
    highlights TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    recommendations TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    raw_response JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_app_resume_analyses_user_hash_model
    ON app_resume_analyses(user_id, content_hash, model);

CREATE INDEX IF NOT EXISTS idx_app_resume_analyses_user_created_at
    ON app_resume_analyses(user_id, created_at DESC);
