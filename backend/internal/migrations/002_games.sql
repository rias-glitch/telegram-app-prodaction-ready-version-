CREATE TABLE IF NOT EXISTS games (
    id BIGSERIAL PRIMARY KEY,
    room_id TEXT,
    player_a_id BIGINT NOT NULL,
    player_b_id BIGINT NOT NULL,
    moves JSONB,
    winner_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
