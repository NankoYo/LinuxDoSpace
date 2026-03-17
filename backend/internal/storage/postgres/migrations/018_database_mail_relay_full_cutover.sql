-- 018_database_mail_relay_full_cutover.sql completes the move away from
-- Cloudflare Email Routing destination addresses. Email-target verification is
-- now owned by LinuxDoSpace itself, and normal mailbox forwarding also needs a
-- database-backed per-user daily usage counter.

ALTER TABLE email_targets
ADD COLUMN IF NOT EXISTS verification_token_hash TEXT NOT NULL DEFAULT '';

ALTER TABLE email_targets
ADD COLUMN IF NOT EXISTS verification_expires_at TEXT NULL;

CREATE INDEX IF NOT EXISTS idx_email_targets_verification_token_hash
ON email_targets(verification_token_hash);

CREATE TABLE IF NOT EXISTS mail_forward_daily_usage (
    user_id BIGINT NOT NULL,
    usage_date TEXT NOT NULL,
    used_count BIGINT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (user_id, usage_date),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_mail_forward_daily_usage_usage_date
ON mail_forward_daily_usage(usage_date, user_id);
