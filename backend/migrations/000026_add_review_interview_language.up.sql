ALTER TABLE app_review_sessions
    ADD COLUMN IF NOT EXISTS interview_language TEXT NOT NULL DEFAULT 'en';

ALTER TABLE app_review_sessions
    DROP CONSTRAINT IF EXISTS app_review_sessions_interview_language_check;

ALTER TABLE app_review_sessions
    ADD CONSTRAINT app_review_sessions_interview_language_check
    CHECK (interview_language IN ('en', 'id'));
