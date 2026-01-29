-- Add currency column to game_history for tracking which currency was bet
ALTER TABLE game_history ADD COLUMN IF NOT EXISTS currency VARCHAR(10) NOT NULL DEFAULT 'gems';

-- Create index for filtering by currency
CREATE INDEX IF NOT EXISTS idx_game_history_currency ON game_history(currency);
