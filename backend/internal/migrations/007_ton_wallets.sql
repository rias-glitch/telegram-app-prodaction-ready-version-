-- интеграция TON кошелька
-- связывает пользовательские аккаунты с адресами TON кошельков

CREATE TABLE IF NOT EXISTS wallets (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    address VARCHAR(100) NOT NULL,
    -- адреса TON могут быть в разных форматах (raw, bounceable, non-bounceable)
    -- храним сырой формат для консистентности
    raw_address VARCHAR(100),
    linked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_verified BOOLEAN DEFAULT FALSE,
    -- для верификации TON Connect proof
    last_proof_timestamp BIGINT,
    UNIQUE(user_id),
    UNIQUE(address)
);

CREATE INDEX IF NOT EXISTS idx_wallets_user_id ON wallets(user_id);
CREATE INDEX IF NOT EXISTS idx_wallets_address ON wallets(address);