-- Add coins and GK rewards to quests
ALTER TABLE quests ADD COLUMN IF NOT EXISTS reward_coins BIGINT DEFAULT 0;
ALTER TABLE quests ADD COLUMN IF NOT EXISTS reward_gk BIGINT DEFAULT 0;

-- Update comments
COMMENT ON COLUMN quests.reward_coins IS 'Coins reward for completing the quest';
COMMENT ON COLUMN quests.reward_gk IS 'GK reward for completing the quest';
