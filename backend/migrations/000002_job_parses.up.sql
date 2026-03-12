CREATE TABLE IF NOT EXISTS app_job_parses (
    id UUID PRIMARY KEY,
    user_id TEXT NOT NULL,
    raw_description TEXT NOT NULL,
    skills TEXT[] NOT NULL DEFAULT '{}',
    keywords TEXT[] NOT NULL DEFAULT '{}',
    themes TEXT[] NOT NULL DEFAULT '{}',
    seniority VARCHAR(50) NOT NULL DEFAULT 'unspecified',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_app_job_parses_user_id ON app_job_parses(user_id);
CREATE INDEX IF NOT EXISTS idx_app_job_parses_created_at ON app_job_parses(created_at DESC);
