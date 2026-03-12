CREATE TABLE IF NOT EXISTS app_session_answers (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL,
    question_id UUID NOT NULL,
    user_id TEXT NOT NULL,
    answer_text TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_app_session_answers_session_id ON app_session_answers(session_id);
CREATE INDEX IF NOT EXISTS idx_app_session_answers_question_id ON app_session_answers(question_id);
CREATE INDEX IF NOT EXISTS idx_app_session_answers_user_id ON app_session_answers(user_id);
