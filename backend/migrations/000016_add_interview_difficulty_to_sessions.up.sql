ALTER TABLE app_practice_sessions
    ADD COLUMN IF NOT EXISTS interview_difficulty VARCHAR(10) NOT NULL DEFAULT 'medium';

ALTER TABLE app_practice_sessions
    DROP CONSTRAINT IF EXISTS app_practice_sessions_interview_difficulty_check;

ALTER TABLE app_practice_sessions
    ADD CONSTRAINT app_practice_sessions_interview_difficulty_check
    CHECK (interview_difficulty IN ('easy', 'medium', 'hard'));

CREATE INDEX IF NOT EXISTS idx_app_practice_sessions_difficulty ON app_practice_sessions(interview_difficulty);
