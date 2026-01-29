-- Cleanup duplicate quests that were created by repeated migration runs

-- Delete duplicate quests, keeping only the one with the smallest ID for each unique combination
DELETE FROM quests
WHERE id NOT IN (
    SELECT MIN(id)
    FROM quests
    GROUP BY quest_type, title, action_type, target_count
);

-- Add unique constraint to prevent future duplicates
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'quests_unique_definition'
    ) THEN
        ALTER TABLE quests ADD CONSTRAINT quests_unique_definition
        UNIQUE (quest_type, title, action_type, target_count);
    END IF;
END $$;

-- Log cleanup result
DO $$
DECLARE
    quest_count INT;
BEGIN
    SELECT COUNT(*) INTO quest_count FROM quests;
    RAISE NOTICE 'Quest cleanup complete. Remaining quests: %', quest_count;
END $$;
