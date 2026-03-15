DROP INDEX IF EXISTS idx_app_practice_sessions_language;

ALTER TABLE app_practice_sessions
    DROP CONSTRAINT IF EXISTS app_practice_sessions_interview_language_check;

ALTER TABLE app_practice_sessions
    DROP COLUMN IF EXISTS interview_language;
