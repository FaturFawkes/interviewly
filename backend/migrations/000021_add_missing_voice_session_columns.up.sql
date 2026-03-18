ALTER TABLE app_practice_sessions
    ADD COLUMN IF NOT EXISTS interview_mode VARCHAR(10) NOT NULL DEFAULT 'text',
    ADD COLUMN IF NOT EXISTS target_role TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS target_company TEXT NOT NULL DEFAULT '';

ALTER TABLE app_practice_sessions
    DROP CONSTRAINT IF EXISTS app_practice_sessions_interview_mode_check;

ALTER TABLE app_practice_sessions
    ADD CONSTRAINT app_practice_sessions_interview_mode_check
    CHECK (interview_mode IN ('text', 'voice'));

CREATE INDEX IF NOT EXISTS idx_app_practice_sessions_mode
    ON app_practice_sessions(interview_mode);
