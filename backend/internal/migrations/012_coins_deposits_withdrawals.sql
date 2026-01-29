-- Add coins support to deposits
ALTER TABLE deposits ADD COLUMN IF NOT EXISTS coins_credited BIGINT NOT NULL DEFAULT 0;

-- Add coins support to withdrawals
ALTER TABLE withdrawals ADD COLUMN IF NOT EXISTS coins_amount BIGINT NOT NULL DEFAULT 0;
ALTER TABLE withdrawals ADD COLUMN IF NOT EXISTS fee_coins BIGINT NOT NULL DEFAULT 0;

-- Update existing withdrawals to use coins (convert gems_amount to coins_amount)
-- gems_amount / 1000 = coins (since 10000 gems = 1 TON and 10 coins = 1 TON, so 1000 gems = 1 coin)
UPDATE withdrawals SET coins_amount = gems_amount / 1000, fee_coins = fee_gems / 1000 WHERE coins_amount = 0 AND gems_amount > 0;
