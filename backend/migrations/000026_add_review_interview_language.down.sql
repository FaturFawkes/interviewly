ALTER TABLE app_review_sessions
    DROP CONSTRAINT IF EXISTS app_review_sessions_interview_language_check;

ALTER TABLE app_review_sessions
    DROP COLUMN IF EXISTS interview_language;
