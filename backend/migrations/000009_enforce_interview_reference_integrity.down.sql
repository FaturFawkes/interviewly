ALTER TABLE IF EXISTS app_feedback
    DROP CONSTRAINT IF EXISTS fk_app_feedback_question;

ALTER TABLE IF EXISTS app_feedback
    DROP CONSTRAINT IF EXISTS fk_app_feedback_session;

ALTER TABLE IF EXISTS app_session_answers
    DROP CONSTRAINT IF EXISTS fk_app_session_answers_question;

ALTER TABLE IF EXISTS app_session_answers
    DROP CONSTRAINT IF EXISTS fk_app_session_answers_session;

DROP INDEX IF EXISTS idx_app_feedback_question_id;
