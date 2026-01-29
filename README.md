# Telegram WebApp Gaming Platform

Полноценная игровая платформа для Telegram Mini Apps с PvE/PvP играми, системой квестов, рефералами и интеграцией TON Blockchain.

## Стек технологий

**Backend:** Go 1.24 + Gin + PostgreSQL + WebSocket
**Frontend:** React 18 + Vite 5 + Tailwind CSS
**Blockchain:** TON Connect + tonutils-go
**Deploy:** Docker + Render.com

---

## Возможности

### Игры

| Игра | Режим | Описание |
|------|-------|----------|
| Coin Flip | PvE | Классический 50/50, множитель x2 |
| Coin Flip Pro | PvE | Серия ставок с кэшаутом |
| Rock Paper Scissors | PvE / PvP | Камень-ножницы-бумага |
| Mines | PvE / PvP | Поле 4x3, найди безопасные клетки |
| Mines Pro | PvE | Поле 5x5, настраиваемые мины (1-24), кэшаут |
| Dice | PvE | Кости с режимами exact/low/high |
| Wheel | PvE | Колесо фортуны с множителями x0.1 - x10 |
| Case | Solo | Лутбоксы с призами |

### PvP (WebSocket)

- Автоматический матчмейкинг по ставке и валюте
- Таймеры ходов (15-20 сек)
- Синхронизация в реальном времени
- История всех матчей

### Валюты

| Валюта | Тип | Получение |
|--------|-----|-----------|
| Gems | F2P | Игры, квесты, рефералы |
| Coins | Premium | Депозит TON (1 TON = 10 Coins) |
| GK | Progression | Квесты, рефералы |

### Квесты

- **Daily** — ежедневные задания
- **Weekly** — еженедельные
- **One-time** — одноразовые

Типы заданий: сыграть N игр, выиграть, проиграть, потратить/заработать gems

### Рефералы

- Уникальный реферальный код для каждого игрока
- Реферальная ссылка для Telegram
- 50% от выигрышей приглашённых
- Статистика заработка

### TON Blockchain

- TON Connect для подключения кошелька
- Депозиты с автоматическим зачислением
- Выводы с комиссией 5%
- Уведомления админам о транзакциях

---

## Структура проекта

```
telegram-webapp/
├── backend/
│   ├── cmd/
│   │   ├── app/              # Основной сервер
│   │   ├── migrate_apply/    # Миграции БД
│   │   └── ws_smoke/         # Тесты WebSocket
│   ├── internal/
│   │   ├── bot/              # Telegram Admin Bot
│   │   ├── config/           # Конфигурация
│   │   ├── db/               # PostgreSQL
│   │   ├── domain/           # Модели данных
│   │   ├── game/             # Игровая логика
│   │   ├── http/
│   │   │   ├── handlers/     # API (40+ endpoints)
│   │   │   └── middleware/   # JWT, Rate Limiting
│   │   ├── repository/       # Data Access Layer
│   │   ├── service/          # Бизнес-логика
│   │   ├── telegram/         # Telegram Auth
│   │   ├── ton/              # TON интеграция
│   │   └── ws/               # WebSocket PvP
│   └── migrations/           # 18 SQL миграций
│
└── frontend/
    ├── src/
    │   ├── components/
    │   │   ├── games/        # Компоненты игр
    │   │   └── ui/           # UI Kit
    │   ├── pages/            # Страницы
    │   ├── hooks/            # useAuth, useProfile, useWebSocket
    │   └── api/              # HTTP клиент
    └── vite.config.js
```

---

## API Endpoints

### Аутентификация
```
POST /api/v1/auth              # Telegram initData → JWT
```

### Профиль
```
GET  /api/v1/me                # Информация о пользователе
GET  /api/v1/profile           # Профиль + баланс
GET  /api/v1/profile/:id       # Публичный профиль
POST /api/v1/profile/balance   # Изменить баланс
```

### Игры (PvE)
```
POST /api/v1/game/coinflip     # Coin Flip
POST /api/v1/game/rps          # Rock Paper Scissors
POST /api/v1/game/mines        # Mines Simple
POST /api/v1/game/dice         # Dice
POST /api/v1/game/wheel        # Wheel of Fortune
POST /api/v1/game/case         # Lootbox
```

### Mines Pro
```
POST /api/v1/game/mines-pro/start    # Начать игру
POST /api/v1/game/mines-pro/reveal   # Открыть ячейку
POST /api/v1/game/mines-pro/cashout  # Забрать выигрыш
GET  /api/v1/game/mines-pro/state    # Текущее состояние
```

### CoinFlip Pro
```
POST /api/v1/game/coinflip-pro/start    # Начать серию
POST /api/v1/game/coinflip-pro/flip     # Следующий флип
POST /api/v1/game/coinflip-pro/cashout  # Забрать выигрыш
```

### История и статистика
```
GET  /api/v1/me/games          # История игр
GET  /api/v1/top               # Топ-50 игроков
GET  /api/v1/game/limits       # Лимиты ставок
```

### Квесты
```
GET  /api/v1/quests            # Все квесты
GET  /api/v1/me/quests         # Мои квесты + прогресс
POST /api/v1/quests/:id/claim  # Забрать награду
```

### Рефералы
```
GET  /api/v1/referral/code     # Мой код
GET  /api/v1/referral/stats    # Статистика
GET  /api/v1/referral/link     # Ссылка для шеринга
POST /api/v1/referral/apply    # Применить код
```

### TON
```
GET  /api/v1/ton/config              # TON Connect конфиг
POST /api/v1/ton/wallet/connect      # Подключить кошелёк
DELETE /api/v1/ton/wallet            # Отключить
GET  /api/v1/ton/deposit/info        # Адрес для депозита
GET  /api/v1/ton/deposits            # История депозитов
POST /api/v1/ton/withdraw            # Запрос на вывод
GET  /api/v1/ton/withdrawals         # История выводов
```

### WebSocket (PvP)
```
GET /ws?token=<JWT>&game=rps&bet=100&currency=gems
```

### Health
```
GET /health    # DB check + version
GET /healthz   # Liveness
GET /readyz    # Readiness
GET /metrics   # Prometheus
```

---

## WebSocket Protocol

### Client → Server
```json
{ "type": "move", "value": "rock" }        // RPS
{ "type": "setup", "value": [1,2,3,4] }    // Mines: расстановка мин
{ "type": "move", "value": 5 }             // Mines: выбор клетки
```

### Server → Client
```json
{ "type": "ready" }
{ "type": "matched", "payload": { "room_id": "...", "opponent": {...} } }
{ "type": "start", "payload": { "timestamp": ... } }
{ "type": "setup_complete" }
{ "type": "round_result", "payload": { "your_move": 5, "your_hit": false } }
{ "type": "round_draw" }
{ "type": "result", "payload": { "you": "win", "reason": "...", "win_amount": 200 } }
```

---

## База данных

### Основные таблицы

- **users** — пользователи (gems, coins, gk, level)
- **game_history** — история всех игр
- **quests** — квесты
- **user_quests** — прогресс квестов
- **referrals** — реферальные связи
- **ton_wallets** — подключённые кошельки
- **ton_deposits** — депозиты
- **ton_withdrawals** — выводы
- **audit_logs** — логи действий

---

## Запуск

### Требования
- Go 1.24+
- PostgreSQL 15+
- Node.js 18+

### Backend
```bash
cd backend
cp .env.example .env  # настроить переменные
go mod download
go run cmd/migrate_apply/main.go
go run cmd/app/main.go
```

### Frontend
```bash
cd frontend
npm install
npm run dev
```

### Docker
```bash
docker-compose up -d
```

---

## Переменные окружения

### Обязательные
| Переменная | Описание |
|------------|----------|
| `DATABASE_URL` | PostgreSQL connection string |
| `JWT_SECRET` | Секрет для JWT |
| `BOT_TOKEN` | Telegram Bot Token |

### Опциональные
| Переменная | Default | Описание |
|------------|---------|----------|
| `APP_PORT` | 8080 | Порт сервера |
| `MIN_BET` | 10 | Мин. ставка |
| `MAX_BET` | 100000 | Макс. ставка |
| `GAME_RATE_LIMIT` | 60 | Игр в минуту |
| `API_RATE_LIMIT` | 10 | Запросов в минуту |
| `REDIS_URL` | — | Redis для rate limiting |
| `DEV_MODE` | false | Режим разработки |
| `LOG_FORMAT` | text | json для structured logs |
| `ADMIN_TELEGRAM_IDS` | — | ID админов (через запятую) |
| `ADMIN_BOT_ENABLED` | false | Включить админ-бота |

### TON (опционально)
| Переменная | Описание |
|------------|----------|
| `TON_API_KEY` | API ключ tonapi.io |
| `TON_PLATFORM_WALLET` | Адрес кошелька платформы |
| `TON_WALLET_MNEMONIC` | Мнемоника для авто-выводов |
| `TON_NETWORK` | mainnet / testnet |

---

## Статистика

| Метрика | Значение |
|---------|----------|
| API Endpoints | 40+ |
| PvE игр | 7 |
| PvP игр | 2 |
| Валют | 3 (Gems, Coins, GK) |
| SQL миграций | 18 |
| Go файлов | 77 |

---

## Лицензия

MIT
