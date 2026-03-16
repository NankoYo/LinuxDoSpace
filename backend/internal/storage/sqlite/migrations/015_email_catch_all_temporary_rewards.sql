-- 015_email_catch_all_temporary_rewards.sql adds the temporary PoW reward
-- bucket kept alongside the permanent purchased remaining-count pool.
-- Temporary rewards expire at Shanghai-local midnight and therefore need their
-- own nullable expiry timestamp instead of mutating the permanent ledger.

ALTER TABLE email_catch_all_access
ADD COLUMN temporary_reward_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE email_catch_all_access
ADD COLUMN temporary_reward_expires_at TEXT NULL;
