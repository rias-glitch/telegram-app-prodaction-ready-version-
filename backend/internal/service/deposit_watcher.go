package service

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"telegram_webapp/internal/domain"
	"telegram_webapp/internal/logger"
	"telegram_webapp/internal/repository"
	"telegram_webapp/internal/ton"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DepositNotification содержит информацию о депозите для уведомления
type DepositNotification struct {
	UserID        int64
	Username      string
	TgID          int64
	WalletAddress string
	AmountNano    int64
	AmountTON     float64
	CoinsCredited int64
	TxHash        string
	NewBalance    int64
}

// DepositWatcher отслеживает входящие TON транзакции и начисляет коины
type DepositWatcher struct {
	db              *pgxpool.Pool
	tonClient       *ton.Client
	depositRepo     *repository.DepositRepository
	userRepo        *repository.UserRepository
	walletRepo      *repository.WalletRepository
	platformWallet  string
	lastLt          int64 // последнее обработанное логическое время
	interval        time.Duration
	mu              sync.Mutex
	stop               chan struct{}
	running            bool
	notifyCallback     func(DepositNotification) // callback для уведомления админов о депозите
	userNotifyCallback func(DepositNotification) // callback для уведомления пользователя о депозите
}

// NewDepositWatcher создает новый watcher для депозитов
func NewDepositWatcher(
	db *pgxpool.Pool,
	tonClient *ton.Client,
	platformWallet string,
	interval time.Duration,
) *DepositWatcher {
	return &DepositWatcher{
		db:             db,
		tonClient:      tonClient,
		depositRepo:    repository.NewDepositRepository(db),
		userRepo:       repository.NewUserRepository(db),
		walletRepo:     repository.NewWalletRepository(db),
		platformWallet: platformWallet,
		interval:       interval,
		stop:           make(chan struct{}),
	}
}

// Start запускает watcher в фоновом режиме
func (w *DepositWatcher) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.mu.Unlock()

	log := logger.Get()
	log.Info("запуск deposit watcher", "wallet", w.platformWallet, "interval", w.interval)

	// первоначальная проверка
	w.checkDeposits()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.checkDeposits()
		case <-w.stop:
			log.Info("остановка deposit watcher")
			return
		}
	}
}

// Stop останавливает watcher
func (w *DepositWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running {
		close(w.stop)
		w.running = false
	}
}

// SetDepositNotifyCallback устанавливает callback для уведомлений админов о новых депозитах
func (w *DepositWatcher) SetDepositNotifyCallback(callback func(DepositNotification)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.notifyCallback = callback
}

// SetUserNotifyCallback устанавливает callback для уведомлений пользователя о депозите
func (w *DepositWatcher) SetUserNotifyCallback(callback func(DepositNotification)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.userNotifyCallback = callback
}

// checkDeposits проверяет новые транзакции
func (w *DepositWatcher) checkDeposits() {
	log := logger.Get()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if w.platformWallet == "" {
		log.Warn("deposit watcher: адрес платформы не настроен")
		return
	}

	// получаем последние транзакции
	log.Info("deposit watcher: проверка транзакций", "wallet", w.platformWallet)
	txs, err := w.tonClient.GetTransactions(ctx, w.platformWallet, 50, 0)
	if err != nil {
		log.Error("deposit watcher: ошибка получения транзакций",
			"error", err,
			"wallet", w.platformWallet,
			"hint", "проверьте формат TON_PLATFORM_WALLET (должен быть user-friendly EQxxx или raw 0:xxx)")
		return
	}

	log.Info("deposit watcher: получено транзакций", "count", len(txs))

	// фильтруем входящие транзакции
	incoming := ton.ParseIncomingTransactions(txs, w.platformWallet)
	log.Info("deposit watcher: входящих транзакций", "count", len(incoming))

	for _, tx := range incoming {
		if err := w.processTransaction(ctx, &tx); err != nil {
			log.Error("deposit watcher: ошибка обработки транзакции",
				"hash", tx.Hash,
				"error", err)
		}
	}
}

// processTransaction обрабатывает одну транзакцию
func (w *DepositWatcher) processTransaction(ctx context.Context, tx *ton.Transaction) error {
	log := logger.Get()

	// проверяем не обрабатывали ли уже
	exists, err := w.depositRepo.TxHashExists(ctx, tx.Hash)
	if err != nil {
		return fmt.Errorf("ошибка проверки хэша: %w", err)
	}
	if exists {
		return nil // уже обработано
	}

	var userID int64
	var memo string

	// Способ 1: извлекаем userID из memo
	memo = ton.ExtractMemo(tx)
	if memo != "" {
		parsedID, err := parseUserIDFromMemo(memo)
		if err == nil && parsedID > 0 {
			userID = parsedID
			log.Debug("deposit watcher: userID найден в memo",
				"userID", userID,
				"memo", memo,
				"hash", tx.Hash)
		}
	}

	// Способ 2: fallback - ищем по адресу отправителя (привязанный кошелёк)
	if userID == 0 && tx.InMsg != nil && tx.InMsg.Source != nil && tx.InMsg.Source.Address != "" {
		sourceAddress := tx.InMsg.Source.Address
		log.Info("deposit watcher: поиск пользователя по адресу",
			"sourceAddress", sourceAddress,
			"hash", tx.Hash)

		// Собираем все возможные форматы адреса для поиска
		addressVariants := []string{sourceAddress}

		// Raw формат
		if normalized, err := ton.NormalizeAddress(sourceAddress); err == nil && normalized != "" {
			addressVariants = append(addressVariants, normalized)
		}

		// User-friendly non-bounceable (UQ...)
		if uf, err := ton.RawToUserFriendly(sourceAddress, false); err == nil && uf != "" {
			addressVariants = append(addressVariants, uf)
		}

		// User-friendly bounceable (EQ...)
		if uf, err := ton.RawToUserFriendly(sourceAddress, true); err == nil && uf != "" {
			addressVariants = append(addressVariants, uf)
		}

		log.Info("deposit watcher: варианты адресов для поиска",
			"variants", addressVariants,
			"hash", tx.Hash)

		// Ищем кошелек по любому из форматов
		wallet, err := w.walletRepo.GetByAnyAddress(ctx, addressVariants)

		if err == nil && wallet != nil {
			userID = wallet.UserID
			memo = fmt.Sprintf("wallet_%d", userID)
			log.Info("deposit watcher: userID найден по адресу кошелька",
				"userID", userID,
				"walletAddress", sourceAddress,
				"dbAddress", wallet.Address,
				"hash", tx.Hash)
		} else {
			log.Warn("deposit watcher: кошелек не найден в БД",
				"sourceAddress", sourceAddress,
				"variants", addressVariants,
				"hash", tx.Hash)
		}
	}

	// если userID не найден ни одним способом - пропускаем
	if userID == 0 {
		sourceAddr := ""
		if tx.InMsg != nil && tx.InMsg.Source != nil {
			sourceAddr = tx.InMsg.Source.Address
		}
		log.Debug("deposit watcher: не удалось идентифицировать пользователя",
			"memo", memo,
			"source", sourceAddr,
			"hash", tx.Hash)
		return nil
	}

	// проверяем существует ли пользователь
	user, err := w.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		log.Warn("deposit watcher: пользователь не найден",
			"userID", userID,
			"hash", tx.Hash)
		return nil
	}

	// получаем сумму
	amountNano := tx.InMsg.Value
	if amountNano < ton.MinDepositNano {
		log.Debug("deposit watcher: сумма меньше минимальной",
			"amount", amountNano,
			"min", ton.MinDepositNano)
		return nil
	}

	// конвертируем в коины
	coinsCredited := ton.NanoToCoins(amountNano)

	log.Info("deposit watcher: обнаружен новый депозит",
		"userID", userID,
		"amountNano", amountNano,
		"amountTON", ton.NanoToTON(amountNano),
		"coins", coinsCredited,
		"hash", tx.Hash)

	// получаем адрес отправителя
	sourceAddress := ""
	if tx.InMsg != nil && tx.InMsg.Source != nil {
		sourceAddress = tx.InMsg.Source.Address
	}

	// создаем запись депозита
	// GemsCredited используется для обратной совместимости с БД (колонка gems_credited)
	// но фактически хранит coins
	deposit := &domain.Deposit{
		UserID:        userID,
		WalletAddress: sourceAddress,
		AmountNano:    amountNano,
		GemsCredited:  coinsCredited, // сохраняем в gems_credited для совместимости с БД
		CoinsCredited: coinsCredited, // также в CoinsCredited для JSON ответов
		ExchangeRate:  ton.CoinsPerTON,
		TxHash:        tx.Hash,
		TxLt:          tx.Lt,
		Status:        domain.DepositStatusConfirmed,
		Memo:          memo,
		Processed:     true,
	}

	if err := w.depositRepo.Create(ctx, deposit); err != nil {
		return fmt.Errorf("ошибка создания депозита: %w", err)
	}

	// начисляем коины пользователю
	newBalance, err := w.userRepo.UpdateCoins(ctx, userID, coinsCredited)
	if err != nil {
		log.Error("deposit watcher: ошибка начисления коинов",
			"userID", userID,
			"coins", coinsCredited,
			"error", err)
		return fmt.Errorf("ошибка начисления коинов: %w", err)
	}

	log.Info("deposit watcher: депозит успешно обработан",
		"userID", userID,
		"coins", coinsCredited,
		"newBalance", newBalance,
		"depositID", deposit.ID,
		"hash", tx.Hash)

	// формируем уведомление
	notification := DepositNotification{
		UserID:        userID,
		Username:      user.Username,
		TgID:          user.TgID,
		WalletAddress: sourceAddress,
		AmountNano:    amountNano,
		AmountTON:     ton.NanoToTON(amountNano),
		CoinsCredited: coinsCredited,
		TxHash:        tx.Hash,
		NewBalance:    newBalance,
	}

	// отправляем уведомление админам
	if w.notifyCallback != nil {
		go w.notifyCallback(notification)
	}

	// отправляем уведомление пользователю
	if w.userNotifyCallback != nil {
		go w.userNotifyCallback(notification)
	}

	return nil
}

// parseUserIDFromMemo извлекает userID из memo
// поддерживает форматы: "deposit_123", "123", "user_123"
func parseUserIDFromMemo(memo string) (int64, error) {
	memo = strings.TrimSpace(memo)

	// формат deposit_123 или user_123
	re := regexp.MustCompile(`(?:deposit_|user_)?(\d+)`)
	matches := re.FindStringSubmatch(memo)
	if len(matches) < 2 {
		return 0, fmt.Errorf("неверный формат memo: %s", memo)
	}

	userID, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("ошибка парсинга userID: %w", err)
	}

	if userID <= 0 {
		return 0, fmt.Errorf("некорректный userID: %d", userID)
	}

	return userID, nil
}
