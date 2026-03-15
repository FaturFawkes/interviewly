ALTER TABLE app_practice_sessions
    ADD COLUMN IF NOT EXISTS interview_language VARCHAR(5) NOT NULL DEFAULT 'en';

ALTER TABLE app_practice_sessions
    DROP CONSTRAINT IF EXISTS app_practice_sessions_interview_language_check;

ALTER TABLE app_practice_sessions
    ADD CONSTRAINT app_practice_sessions_interview_language_check
    CHECK (interview_language IN ('en', 'id'));

CREATE INDEX IF NOT EXISTS idx_app_practice_sessions_language ON app_practice_sessions(interview_language);
