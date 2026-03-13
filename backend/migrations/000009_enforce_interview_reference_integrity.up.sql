DELETE FROM app_session_answers sa
WHERE NOT EXISTS (
    SELECT 1
    FROM app_practice_sessions ps
    WHERE ps.id = sa.session_id
)
OR NOT EXISTS (
    SELECT 1
    FROM app_questions q
    WHERE q.id = sa.question_id
);

DELETE FROM app_feedback fb
WHERE NOT EXISTS (
    SELECT 1
    FROM app_practice_sessions ps
    WHERE ps.id = fb.session_id
)
OR NOT EXISTS (
    SELECT 1
    FROM app_questions q
    WHERE q.id = fb.question_id
);

CREATE INDEX IF NOT EXISTS idx_app_feedback_question_id ON app_feedback(question_id);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_app_session_answers_session'
    ) THEN
        ALTER TABLE app_session_answers
            ADD CONSTRAINT fk_app_session_answers_session
            FOREIGN KEY (session_id)
            REFERENCES app_practice_sessions(id)
            ON DELETE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_app_session_answers_question'
    ) THEN
        ALTER TABLE app_session_answers
            ADD CONSTRAINT fk_app_session_answers_question
            FOREIGN KEY (question_id)
            REFERENCES app_questions(id)
            ON DELETE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_app_feedback_session'
    ) THEN
        ALTER TABLE app_feedback
            ADD CONSTRAINT fk_app_feedback_session
            FOREIGN KEY (session_id)
            REFERENCES app_practice_sessions(id)
            ON DELETE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_app_feedback_question'
    ) THEN
        ALTER TABLE app_feedback
            ADD CONSTRAINT fk_app_feedback_question
            FOREIGN KEY (question_id)
            REFERENCES app_questions(id)
            ON DELETE CASCADE;
    END IF;
END $$;
