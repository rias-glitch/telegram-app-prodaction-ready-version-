-- Add GK currency and character level to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS gk BIGINT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS character_level INT NOT NULL DEFAULT 1;

-- Track referral earnings from withdrawal commissions
ALTER TABLE users ADD COLUMN IF NOT EXISTS referral_earnings BIGINT NOT NULL DEFAULT 0;

-- Table to track claimed GK rewards for referral milestones
CREATE TABLE IF NOT EXISTS gk_rewards_claimed (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    threshold INT NOT NULL,
    reward BIGINT NOT NULL,
    claimed_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, threshold)
);

-- Index for leaderboard queries (monthly)
CREATE INDEX IF NOT EXISTS idx_users_gems ON users(gems DESC);
CREATE INDEX IF NOT EXISTS idx_game_history_created_at ON game_history(created_at);
CREATE INDEX IF NOT EXISTS idx_gk_rewards_user ON gk_rewards_claimed(user_id);
