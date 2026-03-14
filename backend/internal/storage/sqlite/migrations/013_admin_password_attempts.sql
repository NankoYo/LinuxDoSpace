-- 013_admin_password_attempts.sql persists the admin second-factor limiter so
-- failed-password lockouts survive process restarts and can be shared by
-- multiple backend replicas.

CREATE TABLE IF NOT EXISTS admin_password_attempts (
    bucket_type TEXT NOT NULL,
    bucket_key TEXT NOT NULL,
    failure_count INTEGER NOT NULL DEFAULT 0,
    blocked_until TEXT NULL,
    last_seen_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (bucket_type, bucket_key)
);

CREATE INDEX IF NOT EXISTS idx_admin_password_attempts_last_seen_at
ON admin_password_attempts(last_seen_at);
