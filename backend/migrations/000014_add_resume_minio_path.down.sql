DROP INDEX IF EXISTS idx_app_resumes_minio_path;

ALTER TABLE app_resumes
    DROP COLUMN IF EXISTS minio_path;
