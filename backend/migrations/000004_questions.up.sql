CREATE TABLE IF NOT EXISTS app_questions (
    id UUID PRIMARY KEY,
    user_id TEXT NOT NULL,
    resume_id UUID NOT NULL,
    job_parse_id UUID NOT NULL,
    question_type VARCHAR(50) NOT NULL,
    question_text TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_app_questions_user_id ON app_questions(user_id);
CREATE INDEX IF NOT EXISTS idx_app_questions_resume_id ON app_questions(resume_id);
CREATE INDEX IF NOT EXISTS idx_app_questions_job_parse_id ON app_questions(job_parse_id);
