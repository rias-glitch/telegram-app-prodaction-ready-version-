-- Система квестов/заданий с наградами

-- Шаблоны заданий (создаются администратором)
CREATE TABLE IF NOT EXISTS quests (
    id BIGSERIAL PRIMARY KEY,

    -- Тип задания
    quest_type TEXT NOT NULL,         -- 'daily', 'weekly', 'one_time'

    -- Описание
    title TEXT NOT NULL,
    description TEXT,

    -- Условия выполнения
    game_type TEXT,                   -- 'rps', 'mines', 'coinflip', 'case', 'any' (NULL = любая)
    action_type TEXT NOT NULL,        -- 'play', 'win', 'lose', 'spend_gems', 'earn_gems'
    target_count INT NOT NULL,        -- сколько нужно выполнить (5 игр, 3 победы)

    -- Награда
    reward_gems BIGINT NOT NULL,

    -- Статус
    is_active BOOLEAN NOT NULL DEFAULT true,
    sort_order INT NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Прогресс пользователя по заданиям
CREATE TABLE IF NOT EXISTS user_quests (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    quest_id BIGINT NOT NULL REFERENCES quests(id) ON DELETE CASCADE,

    -- Прогресс
    current_count INT NOT NULL DEFAULT 0,
    completed BOOLEAN NOT NULL DEFAULT false,
    reward_claimed BOOLEAN NOT NULL DEFAULT false,

    -- Временные метки
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    reward_claimed_at TIMESTAMPTZ,

    -- Период для daily/weekly (начало текущего периода)
    period_start DATE NOT NULL DEFAULT CURRENT_DATE,

    -- Уникальность: один квест на пользователя на период
    UNIQUE(user_id, quest_id, period_start)
);

-- Индексы
CREATE INDEX IF NOT EXISTS idx_quests_active ON quests(is_active, sort_order);
CREATE INDEX IF NOT EXISTS idx_quests_type ON quests(quest_type);

CREATE INDEX IF NOT EXISTS idx_user_quests_user ON user_quests(user_id);
CREATE INDEX IF NOT EXISTS idx_user_quests_quest ON user_quests(quest_id);
CREATE INDEX IF NOT EXISTS idx_user_quests_active ON user_quests(user_id, completed) WHERE completed = false;
CREATE INDEX IF NOT EXISTS idx_user_quests_period ON user_quests(user_id, period_start);

-- Комментарии
COMMENT ON TABLE quests IS 'Шаблоны заданий с наградами';
COMMENT ON TABLE user_quests IS 'Прогресс пользователей по заданиям';
COMMENT ON COLUMN quests.quest_type IS 'daily - ежедневное, weekly - еженедельное, one_time - разовое';
COMMENT ON COLUMN quests.action_type IS 'play - сыграть, win - победить, spend_gems - потратить, earn_gems - заработать';
COMMENT ON COLUMN user_quests.period_start IS 'Начало периода для daily/weekly заданий (для сброса прогресса)';

-- Начальные задания
INSERT INTO quests (quest_type, title, description, game_type, action_type, target_count, reward_gems, sort_order) VALUES
    -- Ежедневные
    ('daily', 'Первые шаги', 'Сыграй 3 игры в любом режиме', 'any', 'play', 3, 25, 1),
    ('daily', 'Активный игрок', 'Сыграй 10 игр в любом режиме', 'any', 'play', 10, 75, 2),
    ('daily', 'Победитель', 'Одержи 3 победы', 'any', 'win', 3, 100, 3),
    ('daily', 'Минёр', 'Сыграй 5 игр в Mines', 'mines', 'play', 5, 50, 4),
    ('daily', 'Камень-ножницы-бумага', 'Сыграй 5 игр в RPS', 'rps', 'play', 5, 50, 5),
    ('daily', 'Испытай удачу', 'Открой 3 кейса', 'case', 'play', 3, 30, 6),

    -- Еженедельные
    ('weekly', 'Марафонец', 'Сыграй 50 игр за неделю', 'any', 'play', 50, 300, 10),
    ('weekly', 'Чемпион недели', 'Одержи 25 побед за неделю', 'any', 'win', 25, 500, 11),
    ('weekly', 'Коллекционер', 'Открой 20 кейсов за неделю', 'case', 'play', 20, 200, 12),

    -- Разовые (для новичков)
    ('one_time', 'Добро пожаловать!', 'Сыграй свою первую игру', 'any', 'play', 1, 100, 100),
    ('one_time', 'Первая победа', 'Одержи свою первую победу', 'any', 'win', 1, 200, 101),
    ('one_time', 'Первый кейс', 'Открой свой первый кейс', 'case', 'play', 1, 50, 102),
    ('one_time', 'Опытный игрок', 'Сыграй 100 игр', 'any', 'play', 100, 1000, 103),
    ('one_time', 'Легенда', 'Одержи 50 побед', 'any', 'win', 50, 2000, 104);
