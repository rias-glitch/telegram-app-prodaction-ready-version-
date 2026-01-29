-- Fix withdrawals table for coins-based withdrawals
-- gems_amount is deprecated but needs a default for backwards compatibility

-- Add default value to gems_amount so INSERT works without it
ALTER TABLE withdrawals ALTER COLUMN gems_amount SET DEFAULT 0;

-- Drop the old constraint that requires gems_amount >= 1000
-- (now we use coins_amount instead)
ALTER TABLE withdrawals DROP CONSTRAINT IF EXISTS min_withdrawal_amount;

-- Add new constraint for coins-based minimum (10 coins = 1 TON)
ALTER TABLE withdrawals DROP CONSTRAINT IF EXISTS min_withdrawal_coins;
ALTER TABLE withdrawals ADD CONSTRAINT min_withdrawal_coins CHECK (coins_amount >= 10 OR gems_amount >= 1000);
