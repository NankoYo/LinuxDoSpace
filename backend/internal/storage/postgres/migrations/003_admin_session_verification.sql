-- 003_admin_session_verification.sql adds a nullable timestamp that marks when
-- one administrator session completed the extra password verification step.
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS admin_verified_at TEXT;
