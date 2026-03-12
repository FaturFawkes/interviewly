CREATE TABLE IF NOT EXISTS app_resumes (
    id UUID PRIMARY KEY,
    user_id TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_app_resumes_user_id ON app_resumes(user_id);
CREATE INDEX IF NOT EXISTS idx_app_resumes_created_at ON app_resumes(created_at DESC);
