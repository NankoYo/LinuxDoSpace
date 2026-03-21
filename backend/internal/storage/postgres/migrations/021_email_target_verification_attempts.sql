-- 021_email_target_verification_attempts.sql stores outbound verification-send
-- attempts so resend limits survive process restarts and multi-instance deploys.

CREATE TABLE IF NOT EXISTS email_target_verification_attempts (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    prepared_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_email_target_verification_attempts_owner_prepared
ON email_target_verification_attempts(owner_user_id, prepared_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_email_target_verification_attempts_email_prepared
ON email_target_verification_attempts(email, prepared_at DESC, id DESC);
