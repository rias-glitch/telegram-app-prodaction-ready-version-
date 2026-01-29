-- TON Withdrawals
-- Tracks outgoing TON payments from platform to users

CREATE TABLE IF NOT EXISTS withdrawals (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wallet_address VARCHAR(100) NOT NULL,

    -- Gems deducted from user account
    gems_amount INT NOT NULL,

    -- TON amount to send (in nanoTON)
    ton_amount_nano BIGINT NOT NULL,

    -- Fee deducted (in gems)
    fee_gems INT DEFAULT 0,

    -- Exchange rate at time of withdrawal
    exchange_rate INT NOT NULL,

    -- Withdrawal status
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN (
        'pending',      -- User requested, awaiting processing
        'processing',   -- Being processed by system
        'sent',         -- Transaction sent to blockchain
        'completed',    -- Confirmed on blockchain
        'failed',       -- Failed to process
        'cancelled'     -- Cancelled by user or admin
    )),

    -- TON blockchain transaction details (filled after sending)
    tx_hash VARCHAR(100),
    tx_lt BIGINT,

    -- Admin notes (for manual review)
    admin_notes TEXT,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    CONSTRAINT min_withdrawal_amount CHECK (gems_amount >= 1000)
);

CREATE INDEX IF NOT EXISTS idx_withdrawals_user_id ON withdrawals(user_id);
CREATE INDEX IF NOT EXISTS idx_withdrawals_status ON withdrawals(status);
CREATE INDEX IF NOT EXISTS idx_withdrawals_tx_hash ON withdrawals(tx_hash);
