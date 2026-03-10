-- 005_email_targets.sql stores per-user forwarding destinations so LinuxDoSpace
-- can bind one external inbox to exactly one local user and remember the
-- Cloudflare verification lifecycle across requests.

CREATE TABLE IF NOT EXISTS email_targets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_user_id INTEGER NOT NULL,
    email TEXT NOT NULL UNIQUE,
    cloudflare_address_id TEXT NOT NULL DEFAULT '',
    verified_at TEXT NULL,
    last_verification_sent_at TEXT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_email_targets_owner_user_id ON email_targets(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_email_targets_verified_at ON email_targets(verified_at);
