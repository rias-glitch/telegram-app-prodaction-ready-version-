-- Универсальная таблица истории всех игр (PvP, PvE, кейсы)
-- Заменяет старую таблицу games

-- Переименовываем старую таблицу (сохраняем данные на всякий случай)
-- Только если games существует и games_old не существует
DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'games')
       AND NOT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'games_old') THEN
        ALTER TABLE games RENAME TO games_old;
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS game_history (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Тип игры и режим
    game_type TEXT NOT NULL,      -- 'rps', 'mines', 'coinflip', 'case'
    mode TEXT NOT NULL,           -- 'pvp', 'pve', 'solo'

    -- Противник (NULL для PvE/solo игр)
    opponent_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    room_id TEXT,                 -- ID комнаты для PvP матчей

    -- Результат
    result TEXT NOT NULL,         -- 'win', 'lose', 'draw'
    bet_amount BIGINT NOT NULL DEFAULT 0,
    win_amount BIGINT NOT NULL DEFAULT 0,  -- итоговый выигрыш (отрицательный = проигрыш)

    -- Детали игры в JSON (ходы, выпавший приз кейса и т.д.)
    details JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Индексы для быстрых запросов
CREATE INDEX IF NOT EXISTS idx_game_history_user_id ON game_history(user_id);
CREATE INDEX IF NOT EXISTS idx_game_history_user_created ON game_history(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_game_history_game_type ON game_history(game_type);
CREATE INDEX IF NOT EXISTS idx_game_history_mode ON game_history(mode);
CREATE INDEX IF NOT EXISTS idx_game_history_result ON game_history(result);
CREATE INDEX IF NOT EXISTS idx_game_history_created_at ON game_history(created_at DESC);

-- Индекс для статистики за последний месяц (частичный индекс)
CREATE INDEX IF NOT EXISTS idx_game_history_stats ON game_history(user_id, result, created_at)
    ;

-- Комментарии для документации
COMMENT ON TABLE game_history IS 'История всех игр пользователей (PvP, PvE, кейсы)';
COMMENT ON COLUMN game_history.game_type IS 'Тип игры: rps, mines, coinflip, case';
COMMENT ON COLUMN game_history.mode IS 'Режим: pvp (против игрока), pve (против бота), solo (одиночная)';
COMMENT ON COLUMN game_history.result IS 'Результат: win, lose, draw';
COMMENT ON COLUMN game_history.win_amount IS 'Чистый выигрыш (может быть отрицательным при проигрыше)';
COMMENT ON COLUMN game_history.details IS 'JSON с деталями: ходы игроков, выпавший приз и т.д.';
