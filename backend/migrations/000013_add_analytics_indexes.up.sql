CREATE INDEX IF NOT EXISTS idx_app_practice_sessions_user_created_at
    ON app_practice_sessions(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_app_practice_sessions_user_status_created_at
    ON app_practice_sessions(user_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_app_feedback_user_created_at
    ON app_feedback(user_id, created_at DESC);
