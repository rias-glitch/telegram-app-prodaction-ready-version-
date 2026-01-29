-- депозиты в TON
-- отслеживает входящие платежи в TON и зачисленные гемы

CREATE TABLE IF NOT EXISTS deposits (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wallet_address VARCHAR(100) NOT NULL,

    -- сумма в nanoTON (1 TON = 10^9 nanoTON)
    amount_nano BIGINT NOT NULL,

    -- гемы зачисленные на аккаунт пользователя (старое сейчас не юз - донат за коины)
    gems_credited INT NOT NULL,

    -- курс обмена на момент депозита (гемов за TON)
    exchange_rate INT NOT NULL,

    -- детали транзакции блокчейна TON
    tx_hash VARCHAR(100) UNIQUE NOT NULL,
    tx_lt BIGINT,  -- логическое время

    -- статус депозита
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'failed', 'expired')),

    -- опциональное примечание для идентификации депозитов
    memo VARCHAR(100),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    confirmed_at TIMESTAMP WITH TIME ZONE,

    -- для идемпотентности - предотвращение двойного зачисления
    processed BOOLEAN DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_deposits_user_id ON deposits(user_id);
CREATE INDEX IF NOT EXISTS idx_deposits_tx_hash ON deposits(tx_hash);
CREATE INDEX IF NOT EXISTS idx_deposits_status ON deposits(status);
CREATE INDEX IF NOT EXISTS idx_deposits_wallet ON deposits(wallet_address);