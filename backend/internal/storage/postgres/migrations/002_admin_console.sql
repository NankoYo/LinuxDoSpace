-- 002_admin_console.sql adds the persistence required by the standalone administrator console.
-- The new tables intentionally keep moderation and operator data separate from OAuth profile data.

CREATE TABLE IF NOT EXISTS user_controls (
    user_id BIGINT PRIMARY KEY,
    is_banned INTEGER NOT NULL DEFAULT 0,
    note TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS email_routes (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL,
    root_domain TEXT NOT NULL,
    prefix TEXT NOT NULL,
    target_email TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(root_domain, prefix),
    FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS admin_applications (
    id BIGSERIAL PRIMARY KEY,
    applicant_user_id BIGINT NOT NULL,
    type TEXT NOT NULL,
    target TEXT NOT NULL,
    reason TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    review_note TEXT NOT NULL DEFAULT '',
    reviewed_by_user_id BIGINT NULL,
    reviewed_at TEXT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (applicant_user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (reviewed_by_user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS redeem_codes (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    target TEXT NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    created_by_user_id BIGINT NOT NULL,
    used_by_user_id BIGINT NULL,
    used_at TEXT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (used_by_user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_user_controls_is_banned ON user_controls(is_banned);
CREATE INDEX IF NOT EXISTS idx_email_routes_owner_user_id ON email_routes(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_admin_applications_status_created_at ON admin_applications(status, created_at);
CREATE INDEX IF NOT EXISTS idx_redeem_codes_created_at ON redeem_codes(created_at);
