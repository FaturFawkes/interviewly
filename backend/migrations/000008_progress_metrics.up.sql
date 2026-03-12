CREATE TABLE IF NOT EXISTS app_progress_metrics (
    user_id TEXT PRIMARY KEY,
    average_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    weak_areas TEXT[] NOT NULL DEFAULT '{}',
    sessions_completed INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_app_progress_metrics_updated_at ON app_progress_metrics(updated_at DESC);
