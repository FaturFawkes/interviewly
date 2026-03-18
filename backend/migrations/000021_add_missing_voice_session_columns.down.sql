DROP INDEX IF EXISTS idx_app_practice_sessions_mode;

ALTER TABLE app_practice_sessions
    DROP CONSTRAINT IF EXISTS app_practice_sessions_interview_mode_check;

ALTER TABLE app_practice_sessions
    DROP COLUMN IF EXISTS target_company,
    DROP COLUMN IF EXISTS target_role,
    DROP COLUMN IF EXISTS interview_mode;
