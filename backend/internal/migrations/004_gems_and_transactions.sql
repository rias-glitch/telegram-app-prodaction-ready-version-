-- Add gems balance to users and transactions table
ALTER TABLE users ADD COLUMN IF NOT EXISTS gems BIGINT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    type TEXT NOT NULL,
    amount BIGINT NOT NULL,
    meta JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Add foreign key to users and useful indexes for querying history and recent activity
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'transactions_user_fk'
    ) THEN
        ALTER TABLE transactions
            ADD CONSTRAINT transactions_user_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_transactions_user_created_at ON transactions (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions (type);
CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions (created_at DESC);
