-- 010_pow_challenges.sql stores user-bound proof-of-work puzzles together with
-- the immutable reward parameters that were generated for each challenge.

CREATE TABLE IF NOT EXISTS pow_challenges (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    benefit_key TEXT NOT NULL,
    resource_key TEXT NOT NULL,
    scope TEXT NOT NULL DEFAULT '',
    difficulty INTEGER NOT NULL,
    base_reward INTEGER NOT NULL,
    reward_quantity INTEGER NOT NULL,
    reward_unit TEXT NOT NULL,
    challenge_token TEXT NOT NULL UNIQUE,
    salt_hex TEXT NOT NULL,
    argon2_variant TEXT NOT NULL,
    argon2_memory_kib INTEGER NOT NULL,
    argon2_iterations INTEGER NOT NULL,
    argon2_parallelism INTEGER NOT NULL,
    argon2_hash_length INTEGER NOT NULL,
    status TEXT NOT NULL,
    solution_nonce TEXT NOT NULL DEFAULT '',
    solution_hash_hex TEXT NOT NULL DEFAULT '',
    claimed_at TEXT NULL,
    superseded_at TEXT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_pow_challenges_user_status_created
ON pow_challenges(user_id, status, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_pow_challenges_user_claimed_at
ON pow_challenges(user_id, claimed_at, id DESC);
