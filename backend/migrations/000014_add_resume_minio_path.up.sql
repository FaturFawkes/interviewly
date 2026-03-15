ALTER TABLE app_resumes
    ADD COLUMN IF NOT EXISTS minio_path TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_app_resumes_minio_path ON app_resumes(minio_path);
