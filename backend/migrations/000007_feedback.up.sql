CREATE TABLE IF NOT EXISTS app_feedback (
    id UUID PRIMARY KEY,
    user_id TEXT NOT NULL,
    session_id UUID NOT NULL,
    question_id UUID NOT NULL,
    question_text TEXT NOT NULL,
    answer_text TEXT NOT NULL,
    score INTEGER NOT NULL,
    strengths TEXT[] NOT NULL DEFAULT '{}',
    weaknesses TEXT[] NOT NULL DEFAULT '{}',
    improvements TEXT[] NOT NULL DEFAULT '{}',
    star_feedback TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_app_feedback_user_id ON app_feedback(user_id);
CREATE INDEX IF NOT EXISTS idx_app_feedback_session_id ON app_feedback(session_id);
CREATE INDEX IF NOT EXISTS idx_app_feedback_created_at ON app_feedback(created_at DESC);
