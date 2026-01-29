package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"telegram_webapp/internal/ton"

	"github.com/jackc/pgx/v5/pgxpool"
)

// предоставляет административную статистику и операции
type AdminService struct {
	db     *pgxpool.Pool
	wallet *ton.Wallet
}

// создает новый административный сервис
func NewAdminService(db *pgxpool.Pool) *AdminService {
	return &AdminService{db: db}
}

// устанавливает TON кошелек для автоматических выводов
func (s *AdminService) SetWallet(wallet *ton.Wallet) {
	s.wallet = wallet
}

// представляет статистику платформы
type Stats struct {
	TotalUsers       int64 `json:"total_users"`
	ActiveUsersToday int64 `json:"active_users_today"`
	ActiveUsersWeek  int64 `json:"active_users_week"`
	TotalGamesPlayed int64 `json:"total_games_played"`
	GamesToday       int64 `json:"games_today"`
	TotalGems        int64 `json:"total_gems"`        // общее количество драгоценных камней в обращении
	TotalCoins       int64 `json:"total_coins"`       // общее количество монет в обращении
	TotalWagered     int64 `json:"total_wagered"`     // общая сумма ставок за все время
	WageredToday     int64 `json:"wagered_today"`     // сумма ставок за сегодня
	PendingWithdraws int   `json:"pending_withdraws"` // ожидающие запросы на вывод
	TotalDeposited   int64 `json:"total_deposited"`   // общая сумма депозитов в ton (в драгоценных камнях)
	TotalWithdrawn   int64 `json:"total_withdrawn"`   // общая сумма выводов (в драгоценных камнях)
	// статистика по покупке монет
	CoinsPurchasedToday int64 `json:"coins_purchased_today"`
	CoinsPurchasedWeek  int64 `json:"coins_purchased_week"`
	CoinsPurchasedMonth int64 `json:"coins_purchased_month"`
	CoinsPurchasedTotal int64 `json:"coins_purchased_total"`
}

// возвращает статистику платформы
func (s *AdminService) GetStats(ctx context.Context) (*Stats, error) {
	stats := &Stats{}
	today := time.Now().Truncate(24 * time.Hour)
	weekAgo := today.Add(-7 * 24 * time.Hour)
	monthAgo := today.Add(-30 * 24 * time.Hour)

	// общее количество пользователей
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&stats.TotalUsers)

	// активные пользователи сегодня (сыграли хотя бы одну игру)
	_ = s.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT user_id) FROM game_history WHERE created_at >= $1
	`, today).Scan(&stats.ActiveUsersToday)

	// активные пользователи за эту неделю
	_ = s.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT user_id) FROM game_history WHERE created_at >= $1
	`, weekAgo).Scan(&stats.ActiveUsersWeek)

	// общее количество сыгранных игр
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM game_history`).Scan(&stats.TotalGamesPlayed)

	// игры сегодня
	_ = s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM game_history WHERE created_at >= $1
	`, today).Scan(&stats.GamesToday)

	// общее количество драгоценных камней в обращении
	_ = s.db.QueryRow(ctx, `SELECT COALESCE(SUM(gems), 0) FROM users`).Scan(&stats.TotalGems)

	// общее количество монет в обращении
	_ = s.db.QueryRow(ctx, `SELECT COALESCE(SUM(coins), 0) FROM users`).Scan(&stats.TotalCoins)

	// общая сумма ставок (за все время) - только ставки в монетах
	_ = s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(bet_amount), 0) FROM game_history WHERE currency = 'coins'
	`).Scan(&stats.TotalWagered)

	// сумма ставок сегодня - только ставки в монетах
	_ = s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(bet_amount), 0) FROM game_history WHERE created_at >= $1 AND currency = 'coins'
	`, today).Scan(&stats.WageredToday)

	// ожидающие выводы
	_ = s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM withdrawals WHERE status IN ('pending', 'processing')
	`).Scan(&stats.PendingWithdraws)

	// общая сумма депозитов
	_ = s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(gems_credited), 0) FROM deposits WHERE status = 'confirmed'
	`).Scan(&stats.TotalDeposited)

	// общая сумма выводов
	_ = s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(gems_amount), 0) FROM withdrawals WHERE status IN ('sent', 'completed')
	`).Scan(&stats.TotalWithdrawn)

	// статистика покупки монет (из таблицы депозитов)
	// используем gems_credited т.к. туда сохраняются коины (legacy naming)
	_ = s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(gems_credited), 0) FROM deposits WHERE status = 'confirmed' AND created_at >= $1
	`, today).Scan(&stats.CoinsPurchasedToday)

	_ = s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(gems_credited), 0) FROM deposits WHERE status = 'confirmed' AND created_at >= $1
	`, weekAgo).Scan(&stats.CoinsPurchasedWeek)

	_ = s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(gems_credited), 0) FROM deposits WHERE status = 'confirmed' AND created_at >= $1
	`, monthAgo).Scan(&stats.CoinsPurchasedMonth)

	_ = s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(gems_credited), 0) FROM deposits WHERE status = 'confirmed'
	`).Scan(&stats.CoinsPurchasedTotal)

	return stats, nil
}

// представляет информацию о пользователе для администратора
type UserInfo struct {
	ID          int64     `json:"id"`
	TgID        int64     `json:"tg_id"`
	Username    string    `json:"username"`
	FirstName   string    `json:"first_name"`
	Gems        int64     `json:"gems"`
	Coins       int64     `json:"coins"`
	CreatedAt   time.Time `json:"created_at"`
	GamesPlayed int64     `json:"games_played"`
	TotalWon    int64     `json:"total_won"`
	TotalLost   int64     `json:"total_lost"`
}

// возвращает информацию о пользователе по id или telegram id
func (s *AdminService) GetUser(ctx context.Context, identifier string) (*UserInfo, error) {
	var user UserInfo

	// удаляем @ если присутствует (для поиска по username)
	identifier = strings.TrimPrefix(identifier, "@")

	// пытаемся найти по tg_id сначала (если числовой), затем по username
	var err error
	if tgID, parseErr := strconv.ParseInt(identifier, 10, 64); parseErr == nil {
		// числовой - ищем по tg_id
		err = s.db.QueryRow(ctx, `
			SELECT id, tg_id, COALESCE(username, ''), COALESCE(first_name, ''), gems, COALESCE(coins, 0), created_at
			FROM users
			WHERE tg_id = $1
		`, tgID).Scan(&user.ID, &user.TgID, &user.Username, &user.FirstName, &user.Gems, &user.Coins, &user.CreatedAt)
	} else {
		// не числовой - ищем по username
		err = s.db.QueryRow(ctx, `
			SELECT id, tg_id, COALESCE(username, ''), COALESCE(first_name, ''), gems, COALESCE(coins, 0), created_at
			FROM users
			WHERE LOWER(username) = LOWER($1)
		`, identifier).Scan(&user.ID, &user.TgID, &user.Username, &user.FirstName, &user.Gems, &user.Coins, &user.CreatedAt)
	}

	if err != nil {
		return nil, err
	}

	// получаем статистику игр
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM game_history WHERE user_id = $1`, user.ID).Scan(&user.GamesPlayed)
	_ = s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(win_amount), 0) FROM game_history WHERE user_id = $1 AND win_amount > 0
	`, user.ID).Scan(&user.TotalWon)
	_ = s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(ABS(win_amount)), 0) FROM game_history WHERE user_id = $1 AND win_amount < 0
	`, user.ID).Scan(&user.TotalLost)

	return &user, nil
}

// устанавливает баланс драгоценных камней пользователя
func (s *AdminService) SetUserGems(ctx context.Context, userID int64, gems int64) error {
	_, err := s.db.Exec(ctx, `UPDATE users SET gems = $1 WHERE id = $2`, gems, userID)
	return err
}

// добавляет драгоценные камни к балансу пользователя
func (s *AdminService) AddUserGems(ctx context.Context, userID int64, amount int64) (int64, error) {
	var newBalance int64
	err := s.db.QueryRow(ctx, `
		UPDATE users SET gems = gems + $1 WHERE id = $2 RETURNING gems
	`, amount, userID).Scan(&newBalance)
	return newBalance, err
}

// банит пользователя (устанавливает gems в -1 как маркер)
func (s *AdminService) BanUser(ctx context.Context, userID int64) error {
	_, err := s.db.Exec(ctx, `UPDATE users SET gems = -1 WHERE id = $1`, userID)
	return err
}

// разбанивает пользователя
func (s *AdminService) UnbanUser(ctx context.Context, userID int64) error {
	_, err := s.db.Exec(ctx, `UPDATE users SET gems = 0 WHERE id = $1 AND gems = -1`, userID)
	return err
}

// возвращает лучших пользователей по количеству драгоценных камней
func (s *AdminService) GetTopUsers(ctx context.Context, limit int) ([]UserInfo, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, tg_id, username, first_name, gems, created_at
		FROM users
		WHERE gems >= 0
		ORDER BY gems DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UserInfo
	for rows.Next() {
		var u UserInfo
		if err := rows.Scan(&u.ID, &u.TgID, &u.Username, &u.FirstName, &u.Gems, &u.CreatedAt); err != nil {
			continue
		}
		users = append(users, u)
	}
	return users, nil
}

// возвращает последние игры
func (s *AdminService) GetRecentGames(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	rows, err := s.db.Query(ctx, `
		SELECT gh.id, gh.user_id, u.username, gh.game_type, gh.mode, gh.result,
		       gh.bet_amount, gh.win_amount, gh.created_at
		FROM game_history gh
		JOIN users u ON u.id = gh.user_id
		ORDER BY gh.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []map[string]interface{}
	for rows.Next() {
		var id, userID, betAmount, winAmount int64
		var username, gameType, mode, result string
		var createdAt time.Time

		if err := rows.Scan(&id, &userID, &username, &gameType, &mode, &result, &betAmount, &winAmount, &createdAt); err != nil {
			continue
		}

		games = append(games, map[string]interface{}{
			"id":         id,
			"user_id":    userID,
			"username":   username,
			"game_type":  gameType,
			"mode":       mode,
			"result":     result,
			"bet_amount": betAmount,
			"win_amount": winAmount,
			"created_at": createdAt,
		})
	}
	return games, nil
}

// представляет ожидающий вывод средств
type PendingWithdrawal struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	Username      string    `json:"username"`
	WalletAddress string    `json:"wallet_address"`
	CoinsAmount   int64     `json:"coins_amount"`
	TonAmount     string    `json:"ton_amount"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// возвращает ожидающие запросы на вывод средств
func (s *AdminService) GetPendingWithdrawals(ctx context.Context) ([]PendingWithdrawal, error) {
	rows, err := s.db.Query(ctx, `
		SELECT w.id, w.user_id, COALESCE(u.username, u.first_name, ''), w.wallet_address, w.coins_amount,
		       w.ton_amount_nano, w.status, w.created_at
		FROM withdrawals w
		JOIN users u ON u.id = w.user_id
		WHERE w.status IN ('pending', 'processing')
		ORDER BY w.created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var withdrawals []PendingWithdrawal
	for rows.Next() {
		var w PendingWithdrawal
		var tonNano int64
		if err := rows.Scan(&w.ID, &w.UserID, &w.Username, &w.WalletAddress,
			&w.CoinsAmount, &tonNano, &w.Status, &w.CreatedAt); err != nil {
			continue
		}
		w.TonAmount = fmt.Sprintf("%.4f TON", float64(tonNano)/1e9)
		withdrawals = append(withdrawals, w)
	}
	return withdrawals, nil
}

// ApproveWithdrawalResult результат одобрения вывода
type ApproveWithdrawalResult struct {
	TxHash    string
	AutoSent  bool
	TonAmount float64
	UserTgID  int64 // telegram ID пользователя для уведомления
}

// одобряет вывод средств и автоматически отправляет TON (если настроен кошелек)
func (s *AdminService) ApproveWithdrawal(ctx context.Context, id int64, manualTxHash string) (*ApproveWithdrawalResult, error) {
	// Получаем информацию о выводе вместе с raw_address из wallets и tg_id пользователя
	var walletAddress string
	var rawAddress *string
	var tonAmountNano int64
	var status string
	var userTgID int64
	err := s.db.QueryRow(ctx, `
		SELECT w.wallet_address, wal.raw_address, w.ton_amount_nano, w.status, u.tg_id
		FROM withdrawals w
		LEFT JOIN users u ON u.id = w.user_id
		LEFT JOIN wallets wal ON wal.user_id = u.id
		WHERE w.id = $1
	`, id).Scan(&walletAddress, &rawAddress, &tonAmountNano, &status, &userTgID)
	if err != nil {
		return nil, fmt.Errorf("вывод не найден: %w", err)
	}

	if status != "pending" && status != "processing" {
		return nil, fmt.Errorf("вывод уже обработан (статус: %s)", status)
	}

	result := &ApproveWithdrawalResult{
		TonAmount: float64(tonAmountNano) / 1e9,
		UserTgID:  userTgID,
	}

	// Определяем адрес для отправки: предпочитаем raw_address (формат 0:hex)
	sendAddress := walletAddress
	if rawAddress != nil && *rawAddress != "" {
		sendAddress = *rawAddress
	}

	// Если есть кошелек - отправляем автоматически
	if s.wallet != nil && manualTxHash == "" {
		// Отправляем TON
		sendResult, err := s.wallet.SendTON(ctx, sendAddress, uint64(tonAmountNano), fmt.Sprintf("Withdrawal #%d", id))
		if err != nil {
			return nil, fmt.Errorf("ошибка отправки TON: %w", err)
		}
		result.TxHash = sendResult.TxHash
		result.AutoSent = true
	} else {
		// Ручной режим - используем переданный хэш
		if manualTxHash == "" {
			manualTxHash = fmt.Sprintf("manual_%d_%d", id, time.Now().Unix())
		}
		result.TxHash = manualTxHash
		result.AutoSent = false
	}

	// Обновляем статус вывода
	_, err = s.db.Exec(ctx, `
		UPDATE withdrawals
		SET status = 'sent', tx_hash = $2, processed_at = NOW()
		WHERE id = $1 AND status IN ('pending', 'processing')
	`, id, result.TxHash)
	if err != nil {
		return nil, fmt.Errorf("ошибка обновления статуса: %w", err)
	}

	return result, nil
}

// RejectWithdrawalResult результат отклонения вывода
type RejectWithdrawalResult struct {
	UserTgID    int64 // telegram ID пользователя для уведомления
	CoinsAmount int64 // сумма возвращённых коинов
}

// отклоняет вывод средств и возвращает коины
func (s *AdminService) RejectWithdrawal(ctx context.Context, id int64, reason string) (*RejectWithdrawalResult, error) {
	// начинаем транзакцию
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// получаем информацию о выводе вместе с tg_id пользователя
	var userID, coinsAmount, userTgID int64
	err = tx.QueryRow(ctx, `
		SELECT w.user_id, w.coins_amount, u.tg_id
		FROM withdrawals w
		JOIN users u ON u.id = w.user_id
		WHERE w.id = $1 AND w.status = 'pending'
	`, id).Scan(&userID, &coinsAmount, &userTgID)
	if err != nil {
		return nil, err
	}

	// возвращаем коины
	_, err = tx.Exec(ctx, `UPDATE users SET coins = coins + $1 WHERE id = $2`, coinsAmount, userID)
	if err != nil {
		return nil, err
	}

	// обновляем статус вывода
	_, err = tx.Exec(ctx, `
		UPDATE withdrawals SET status = 'cancelled', admin_notes = $2 WHERE id = $1
	`, id, reason)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &RejectWithdrawalResult{
		UserTgID:    userTgID,
		CoinsAmount: coinsAmount,
	}, nil
}

// отправляет сообщение всем пользователям (возвращает количество)
// примечание: это только сохраняет сообщение - фактическая отправка происходит через бота
func (s *AdminService) GetAllUserTgIDs(ctx context.Context) ([]int64, error) {
	query := `SELECT tg_id FROM users WHERE tg_id IS NOT NULL ORDER BY id`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("ошибка сканирования: %w", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации строк: %w", err)
	}

	return ids, nil
}

// представляет запись об одной игре
type GameRecord struct {
	ID        int64     `json:"id"`
	GameType  string    `json:"game_type"`
	Mode      string    `json:"mode"`
	Result    string    `json:"result"`
	BetAmount int64     `json:"bet_amount"`
	WinAmount int64     `json:"win_amount"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
}

// возвращает последние игры пользователя по telegram id
func (s *AdminService) GetUserGamesByTgID(ctx context.Context, tgID int64, currency string, limit int) ([]GameRecord, error) {
	rows, err := s.db.Query(ctx, `
		SELECT gh.id, gh.game_type, gh.mode, gh.result, gh.bet_amount, gh.win_amount,
		       COALESCE(gh.currency, 'gems') as currency, gh.created_at
		FROM game_history gh
		JOIN users u ON u.id = gh.user_id
		WHERE u.tg_id = $1 AND COALESCE(gh.currency, 'gems') = $2
		ORDER BY gh.created_at DESC
		LIMIT $3
	`, tgID, currency, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []GameRecord
	for rows.Next() {
		var g GameRecord
		if err := rows.Scan(&g.ID, &g.GameType, &g.Mode, &g.Result, &g.BetAmount, &g.WinAmount, &g.Currency, &g.CreatedAt); err != nil {
			continue
		}
		games = append(games, g)
	}
	return games, nil
}

// представляет статистику побед пользователя
type UserGameStats struct {
	UserID    int64  `json:"user_id"`
	TgID      int64  `json:"tg_id"`
	Username  string `json:"username"`
	GemsWins  int64  `json:"gems_wins"`
	CoinsWins int64  `json:"coins_wins"`
}

// возвращает пользователей, отсортированных по количеству побед
func (s *AdminService) GetTopUsersByWins(ctx context.Context, limit int) ([]UserGameStats, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
			u.id,
			u.tg_id,
			COALESCE(u.username, u.first_name) as username,
			COALESCE(SUM(CASE WHEN gh.result = 'win' AND COALESCE(gh.currency, 'gems') = 'gems' THEN 1 ELSE 0 END), 0) as gems_wins,
			COALESCE(SUM(CASE WHEN gh.result = 'win' AND gh.currency = 'coins' THEN 1 ELSE 0 END), 0) as coins_wins
		FROM users u
		LEFT JOIN game_history gh ON gh.user_id = u.id
		GROUP BY u.id, u.tg_id, u.username, u.first_name
		HAVING COALESCE(SUM(CASE WHEN gh.result = 'win' THEN 1 ELSE 0 END), 0) > 0
		ORDER BY (COALESCE(SUM(CASE WHEN gh.result = 'win' AND COALESCE(gh.currency, 'gems') = 'gems' THEN 1 ELSE 0 END), 0) +
		          COALESCE(SUM(CASE WHEN gh.result = 'win' AND gh.currency = 'coins' THEN 1 ELSE 0 END), 0)) DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []UserGameStats
	for rows.Next() {
		var s UserGameStats
		if err := rows.Scan(&s.UserID, &s.TgID, &s.Username, &s.GemsWins, &s.CoinsWins); err != nil {
			continue
		}
		stats = append(stats, s)
	}
	return stats, nil
}

// добавляет монеты к балансу пользователя по tg_id
func (s *AdminService) AddUserCoins(ctx context.Context, tgID int64, amount int64) (int64, error) {
	var newBalance int64
	err := s.db.QueryRow(ctx, `
		UPDATE users SET coins = coins + $1 WHERE tg_id = $2 RETURNING coins
	`, amount, tgID).Scan(&newBalance)
	return newBalance, err
}

// добавляет GK к балансу пользователя по tg_id
func (s *AdminService) AddUserGK(ctx context.Context, tgID int64, amount int64) (int64, error) {
	var newBalance int64
	err := s.db.QueryRow(ctx, `
		UPDATE users SET gk = gk + $1 WHERE tg_id = $2 RETURNING gk
	`, amount, tgID).Scan(&newBalance)
	return newBalance, err
}

// возвращает информацию о пользователе по telegram id
func (s *AdminService) GetUserByTgID(ctx context.Context, tgID int64) (*UserInfo, error) {
	var user UserInfo
	err := s.db.QueryRow(ctx, `
		SELECT id, tg_id, username, first_name, gems, created_at
		FROM users
		WHERE tg_id = $1
	`, tgID).Scan(&user.ID, &user.TgID, &user.Username, &user.FirstName, &user.Gems, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// представляет реферальную статистику для пользователя
type ReferralStat struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	Count     int    `json:"count"`
}

// возвращает пользователей с количеством их рефералов
func (s *AdminService) GetReferralStats(ctx context.Context, limit int) ([]ReferralStat, error) {
	rows, err := s.db.Query(ctx, `
		SELECT u.id, COALESCE(u.username, ''), COALESCE(u.first_name, ''), COUNT(r.id) as ref_count
		FROM users u
		LEFT JOIN referrals r ON r.referrer_id = u.id
		GROUP BY u.id, u.username, u.first_name
		HAVING COUNT(r.id) > 0
		ORDER BY ref_count DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []ReferralStat
	for rows.Next() {
		var s ReferralStat
		if err := rows.Scan(&s.UserID, &s.Username, &s.FirstName, &s.Count); err != nil {
			continue
		}
		stats = append(stats, s)
	}
	return stats, nil
}

// представляет пользователя в списке пользователей
type UserListItem struct {
	ID        int64  `json:"id"`
	TgID      int64  `json:"tg_id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	Gems      int64  `json:"gems"`
	Coins     int64  `json:"coins"`
}

// возвращает всех пользователей с пагинацией
func (s *AdminService) GetAllUsers(ctx context.Context, limit, offset int) ([]UserListItem, int, error) {
	var total int
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)

	rows, err := s.db.Query(ctx, `
		SELECT id, tg_id, COALESCE(username, ''), COALESCE(first_name, ''), gems, COALESCE(coins, 0)
		FROM users
		ORDER BY id DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []UserListItem
	for rows.Next() {
		var u UserListItem
		if err := rows.Scan(&u.ID, &u.TgID, &u.Username, &u.FirstName, &u.Gems, &u.Coins); err != nil {
			continue
		}
		users = append(users, u)
	}
	return users, total, nil
}

// возвращает пользователя по username (без @)
func (s *AdminService) GetUserByUsername(ctx context.Context, username string) (*UserInfo, error) {
	// удаляем @ если присутствует
	username = strings.TrimPrefix(username, "@")

	var user UserInfo
	err := s.db.QueryRow(ctx, `
		SELECT id, tg_id, COALESCE(username, ''), COALESCE(first_name, ''), gems, COALESCE(coins, 0), created_at
		FROM users
		WHERE LOWER(username) = LOWER($1)
	`, username).Scan(&user.ID, &user.TgID, &user.Username, &user.FirstName, &user.Gems, &user.Coins, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// преобразует @username или tg_id во внутренний id пользователя
func (s *AdminService) ResolveUserIdentifier(ctx context.Context, identifier string) (int64, error) {
	// удаляем @ если присутствует
	identifier = strings.TrimPrefix(identifier, "@")

	var userID int64

	// сначала пытаемся распарсить как число (tg_id)
	if tgID, err := strconv.ParseInt(identifier, 10, 64); err == nil {
		err = s.db.QueryRow(ctx, `SELECT id FROM users WHERE tg_id = $1`, tgID).Scan(&userID)
		if err == nil {
			return userID, nil
		}
	}

	// пытаемся как username
	err := s.db.QueryRow(ctx, `SELECT id FROM users WHERE LOWER(username) = LOWER($1)`, identifier).Scan(&userID)
	return userID, err
}

// преобразует @username или tg_id в telegram id пользователя
func (s *AdminService) ResolveToTgID(ctx context.Context, identifier string) (int64, error) {
	// удаляем @ если присутствует
	identifier = strings.TrimPrefix(identifier, "@")

	var tgID int64

	// сначала пытаемся распарсить как число (уже tg_id)
	if parsedTgID, err := strconv.ParseInt(identifier, 10, 64); err == nil {
		// проверяем что такой пользователь существует
		err = s.db.QueryRow(ctx, `SELECT tg_id FROM users WHERE tg_id = $1`, parsedTgID).Scan(&tgID)
		if err == nil {
			return tgID, nil
		}
	}

	// пытаемся как username
	err := s.db.QueryRow(ctx, `SELECT tg_id FROM users WHERE LOWER(username) = LOWER($1)`, identifier).Scan(&tgID)
	return tgID, err
}

// возвращает детали вывода для уведомления
type WithdrawalNotification struct {
	ID            int64
	UserID        int64
	Username      string
	TgID          int64
	WalletAddress string
	CoinsAmount   int64
	TonAmount     float64
}

// возвращает информацию о выводе для административного уведомления
func (s *AdminService) GetWithdrawalNotification(ctx context.Context, withdrawalID int64) (*WithdrawalNotification, error) {
	var w WithdrawalNotification
	var tonNano int64
	err := s.db.QueryRow(ctx, `
		SELECT w.id, w.user_id, COALESCE(u.username, u.first_name, ''), u.tg_id,
		       w.wallet_address, w.coins_amount, w.ton_amount_nano
		FROM withdrawals w
		JOIN users u ON u.id = w.user_id
		WHERE w.id = $1
	`, withdrawalID).Scan(&w.ID, &w.UserID, &w.Username, &w.TgID, &w.WalletAddress, &w.CoinsAmount, &tonNano)
	if err != nil {
		return nil, err
	}
	w.TonAmount = float64(tonNano) / 1e9
	return &w, nil
}

// представляет информацию о квесте для администратора
type QuestInfo struct {
	ID          int64  `json:"id"`
	QuestType   string `json:"quest_type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ActionType  string `json:"action_type"`
	TargetCount int    `json:"target_count"`
	RewardGems  int64  `json:"reward_gems"`
	RewardCoins int64  `json:"reward_coins"`
	RewardGK    int64  `json:"reward_gk"`
	IsActive    bool   `json:"is_active"`
}

// возвращает все квесты для администратора
func (s *AdminService) GetAllQuests(ctx context.Context) ([]QuestInfo, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, quest_type, title, COALESCE(description, ''), action_type,
		       target_count, reward_gems, COALESCE(reward_coins, 0), COALESCE(reward_gk, 0), is_active
		FROM quests
		ORDER BY is_active DESC, quest_type, sort_order, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quests []QuestInfo
	for rows.Next() {
		var q QuestInfo
		if err := rows.Scan(&q.ID, &q.QuestType, &q.Title, &q.Description, &q.ActionType,
			&q.TargetCount, &q.RewardGems, &q.RewardCoins, &q.RewardGK, &q.IsActive); err != nil {
			continue
		}
		quests = append(quests, q)
	}
	return quests, nil
}

// создает новый квест
func (s *AdminService) CreateQuest(ctx context.Context, questType, title, description, actionType string, targetCount int, rewardGems, rewardCoins, rewardGK int64) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO quests (quest_type, title, description, action_type, target_count, reward_gems, reward_coins, reward_gk, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true)
		RETURNING id
	`, questType, title, description, actionType, targetCount, rewardGems, rewardCoins, rewardGK).Scan(&id)
	return id, err
}

// удаляет квест по id
func (s *AdminService) DeleteQuest(ctx context.Context, id int64) error {
	// сначала удаляем весь прогресс пользователей для этого квеста
	_, err := s.db.Exec(ctx, `DELETE FROM user_quests WHERE quest_id = $1`, id)
	if err != nil {
		return err
	}
	// затем удаляем сам квест
	_, err = s.db.Exec(ctx, `DELETE FROM quests WHERE id = $1`, id)
	return err
}

// переключает активный статус квеста
func (s *AdminService) ToggleQuestActive(ctx context.Context, id int64) (bool, error) {
	var newStatus bool
	err := s.db.QueryRow(ctx, `
		UPDATE quests SET is_active = NOT is_active WHERE id = $1 RETURNING is_active
	`, id).Scan(&newStatus)
	return newStatus, err
}

// возвращает общее количество квестов
func (s *AdminService) GetQuestCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM quests`).Scan(&count)
	return count, err
}

// представляет информацию о депозите
type DepositInfo struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	Username      string    `json:"username"`
	TgID          int64     `json:"tg_id"`
	WalletAddress string    `json:"wallet_address"`
	AmountNano    int64     `json:"amount_nano"`
	CoinsCredited int64     `json:"coins_credited"`
	TxHash        string    `json:"tx_hash"`
	Status        string    `json:"status"`
	Memo          string    `json:"memo"`
	CreatedAt     time.Time `json:"created_at"`
}

// возвращает последние депозиты
func (s *AdminService) GetRecentDeposits(ctx context.Context, limit int) ([]DepositInfo, error) {
	rows, err := s.db.Query(ctx, `
		SELECT d.id, d.user_id, COALESCE(u.username, u.first_name, ''), u.tg_id,
		       d.wallet_address, d.amount_nano, d.gems_credited, d.tx_hash, d.status,
		       COALESCE(d.memo, ''), d.created_at
		FROM deposits d
		JOIN users u ON u.id = d.user_id
		ORDER BY d.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deposits []DepositInfo
	for rows.Next() {
		var d DepositInfo
		if err := rows.Scan(&d.ID, &d.UserID, &d.Username, &d.TgID, &d.WalletAddress,
			&d.AmountNano, &d.CoinsCredited, &d.TxHash, &d.Status, &d.Memo, &d.CreatedAt); err != nil {
			continue
		}
		deposits = append(deposits, d)
	}
	return deposits, nil
}

// ManualDepositResult результат ручного депозита
type ManualDepositResult struct {
	ID            int64
	CoinsCredited int64
}

// создает ручной депозит (для админа)
// принимает txHash, tgID пользователя и сумму в TON
func (s *AdminService) CreateManualDeposit(ctx context.Context, txHash string, userTgID int64, tonAmount float64) (*ManualDepositResult, error) {
	// конвертация: 1 TON = 1000 coins
	amountNano := int64(tonAmount * 1e9)
	coinsCredited := int64(tonAmount * 1000)

	// получаем внутренний ID пользователя по tg_id
	var userID int64
	err := s.db.QueryRow(ctx, `SELECT id FROM users WHERE tg_id = $1`, userTgID).Scan(&userID)
	if err != nil {
		return nil, fmt.Errorf("пользователь с TG ID %d не найден", userTgID)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// создаем запись депозита
	// используем gems_credited т.к. туда сохраняются коины (legacy naming)
	var depositID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO deposits (user_id, wallet_address, amount_nano, gems_credited, exchange_rate, tx_hash, tx_lt, status, memo, processed)
		VALUES ($1, 'manual_admin', $2, $3, 1000, $4, 0, 'confirmed', 'manual_admin', true)
		RETURNING id
	`, userID, amountNano, coinsCredited, txHash).Scan(&depositID)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания депозита: %w", err)
	}

	// начисляем коины пользователю
	_, err = tx.Exec(ctx, `UPDATE users SET coins = coins + $1 WHERE id = $2`, coinsCredited, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка начисления коинов: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &ManualDepositResult{
		ID:            depositID,
		CoinsCredited: coinsCredited,
	}, nil
}

// WithdrawalHistoryItem представляет запись вывода для истории
type WithdrawalHistoryItem struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	Username      string    `json:"username"`
	WalletAddress string    `json:"wallet_address"`
	CoinsAmount   int64     `json:"coins_amount"`
	TonAmount     string    `json:"ton_amount"`
	Status        string    `json:"status"`
	TxHash        string    `json:"tx_hash"`
	CreatedAt     time.Time `json:"created_at"`
}

// возвращает историю всех выводов
func (s *AdminService) GetWithdrawalsHistory(ctx context.Context, limit int) ([]WithdrawalHistoryItem, error) {
	rows, err := s.db.Query(ctx, `
		SELECT w.id, w.user_id, COALESCE(u.username, u.first_name, ''), w.wallet_address, w.coins_amount,
		       w.ton_amount_nano, w.status, COALESCE(w.tx_hash, ''), w.created_at
		FROM withdrawals w
		JOIN users u ON u.id = w.user_id
		ORDER BY w.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var withdrawals []WithdrawalHistoryItem
	for rows.Next() {
		var w WithdrawalHistoryItem
		var tonNano int64
		if err := rows.Scan(&w.ID, &w.UserID, &w.Username, &w.WalletAddress,
			&w.CoinsAmount, &tonNano, &w.Status, &w.TxHash, &w.CreatedAt); err != nil {
			continue
		}
		w.TonAmount = fmt.Sprintf("%.4f TON", float64(tonNano)/1e9)
		withdrawals = append(withdrawals, w)
	}
	return withdrawals, nil
}

// DepositHistoryItem представляет запись депозита для истории
type DepositHistoryItem struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	Username      string    `json:"username"`
	TgID          int64     `json:"tg_id"`
	AmountNano    int64     `json:"amount_nano"`
	CoinsCredited int64     `json:"coins_credited"`
	Status        string    `json:"status"`
	TxHash        string    `json:"tx_hash"`
	CreatedAt     time.Time `json:"created_at"`
}

// возвращает историю всех депозитов
func (s *AdminService) GetDepositsHistory(ctx context.Context, limit int) ([]DepositHistoryItem, error) {
	rows, err := s.db.Query(ctx, `
		SELECT d.id, d.user_id, COALESCE(u.username, u.first_name, ''), u.tg_id,
		       d.amount_nano, d.gems_credited, d.status, d.tx_hash, d.created_at
		FROM deposits d
		JOIN users u ON u.id = d.user_id
		ORDER BY d.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deposits []DepositHistoryItem
	for rows.Next() {
		var d DepositHistoryItem
		if err := rows.Scan(&d.ID, &d.UserID, &d.Username, &d.TgID,
			&d.AmountNano, &d.CoinsCredited, &d.Status, &d.TxHash, &d.CreatedAt); err != nil {
			continue
		}
		deposits = append(deposits, d)
	}
	return deposits, nil
}