-- Add coins column to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS coins BIGINT NOT NULL DEFAULT 0;

-- Index for users with coins (for withdrawals, top players etc)
CREATE INDEX IF NOT EXISTS idx_users_coins ON users(coins) WHERE coins > 0;
