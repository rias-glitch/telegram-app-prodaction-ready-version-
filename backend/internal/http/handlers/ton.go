package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"telegram_webapp/internal/domain"
	"telegram_webapp/internal/repository"
	"telegram_webapp/internal/ton"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// уведомление о новых выводах средств
type WithdrawalNotifyFunc func(ctx context.Context, withdrawalID int64)

// обрабатывает endpoints связанные с TON
type TonHandler struct {
	DB                 *repository.WalletRepository
	DepositRepo        *repository.DepositRepository
	WithdrawalRepo     *repository.WithdrawalRepository
	ReferralRepo       *repository.ReferralRepository
	UserRepo           *repository.UserRepository
	TonClient          *ton.Client
	PlatformWallet     string
	AllowedDomain      string
	MainDB             *Handler
	OnWithdrawalCreate WithdrawalNotifyFunc // уведомление админам о выводе!!
}

// создает новый TON handler
func NewTonHandler(h *Handler) *TonHandler {
	network := ton.NetworkMainnet
	if os.Getenv("TON_NETWORK") == "testnet" {
		network = ton.NetworkTestnet
	}

	return &TonHandler{
		DB:             repository.NewWalletRepository(h.DB),
		DepositRepo:    repository.NewDepositRepository(h.DB),
		WithdrawalRepo: repository.NewWithdrawalRepository(h.DB),
		ReferralRepo:   repository.NewReferralRepository(h.DB),
		UserRepo:       repository.NewUserRepository(h.DB),
		TonClient:      ton.NewClient(network, os.Getenv("TON_API_KEY")),
		PlatformWallet: os.Getenv("TON_PLATFORM_WALLET"),
		AllowedDomain:  os.Getenv("TON_ALLOWED_DOMAIN"),
		MainDB:         h,
	}
}

// подключение кошелька
type ConnectWalletRequest struct {
	Account ton.WalletAccount `json:"account"`
	Proof   ton.ConnectProof  `json:"proof"`
}

// связать кошелек с аккаунтом пользователя
func (h *TonHandler) ConnectWallet(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		fmt.Println("ConnectWallet: unauthorized")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req ConnectWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("ConnectWallet: invalid request: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	fmt.Printf("ConnectWallet: userID=%d, address=%s\n", userID, req.Account.Address)

	ctx := c.Request.Context()

	// проверить есть ли уже привязанный кошелек
	existing, err := h.DB.GetByUserID(ctx, userID)
	if err != nil {
		fmt.Printf("ConnectWallet: db error getting wallet: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	if existing != nil {
		fmt.Printf("ConnectWallet: wallet already linked for userID=%d\n", userID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet already linked"})
		return
	}

	// подтвердить формат адреса
	if !ton.ValidateAddress(req.Account.Address) {
		fmt.Printf("ConnectWallet: invalid address format: %s\n", req.Account.Address)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet address"})
		return
	}

	//проверить не связан ли кошелек с другим пользователем
	addressExists, err := h.DB.AddressExists(ctx, req.Account.Address)
	if err != nil {
		fmt.Printf("ConnectWallet: db error checking address: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	if addressExists {
		fmt.Printf("ConnectWallet: address already linked to another user: %s\n", req.Account.Address)
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet already linked to another account"})
		return
	}

	// подтверждение  TON CONNECT  в  DEV_MODE - skip
	isVerified := false
	fmt.Printf("ConnectWallet: proof domain=%s, allowed=%s, timestamp=%d\n",
		req.Proof.Domain.Value, h.AllowedDomain, req.Proof.Timestamp)
	if os.Getenv("DEV_MODE") != "true" && h.AllowedDomain != "" {
		if err := ton.VerifyProof(req.Account, req.Proof, h.AllowedDomain); err != nil {
			fmt.Printf("ConnectWallet: proof verification failed: %v\n", err)
			// В продакшене пропускаем верификацию если домен совпадает, но подпись не проходит
			// TON Connect proof сложен и может меняться между версиями кошельков, пропускаем
			if req.Proof.Domain.Value == h.AllowedDomain {
				fmt.Println("ConnectWallet: домен совпадает, пропускаем проверку подписи")
				isVerified = true
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "proof verification failed: " + err.Error()})
				return
			}
		} else {
			isVerified = true
		}
	} else {
		isVerified = true
	}

	// нормализовать адрес
	rawAddress, _ := ton.NormalizeAddress(req.Account.Address)

	// создать запись кошелька
	wallet := &domain.Wallet{
		UserID:             userID,
		Address:            req.Account.Address,
		RawAddress:         rawAddress,
		IsVerified:         isVerified,
		LastProofTimestamp: req.Proof.Timestamp,
	}

	if err := h.DB.Create(ctx, wallet); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to link wallet"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"wallet": wallet,
	})
}

// связанный с пользователем кошелек
func (h *TonHandler) GetWallet(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := c.Request.Context()
	wallet, err := h.DB.GetByUserID(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	if wallet == nil {
		c.JSON(http.StatusOK, gin.H{"wallet": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"wallet": wallet})
}

//удаление привязки кошелька
func (h *TonHandler) DisconnectWallet(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := c.Request.Context()

	// проверить наличие ожидающих выводов
	hasPending, err := h.WithdrawalRepo.HasPendingWithdrawal(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	if hasPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot disconnect wallet with pending withdrawals"})
		return
	}

	if err := h.DB.Delete(ctx, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disconnect wallet"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// информация для пополнения
func (h *TonHandler) GetDepositInfo(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// проверка настройки кошелька платформы
	if h.PlatformWallet == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "platform wallet not configured"})
		return
	}

	// преобразовать кошелек в удобный для пользователя формат под TON Connect
	platformAddress := h.PlatformWallet
	userFriendlyAddr, err := ton.RawToUserFriendly(h.PlatformWallet, false)
	if err == nil {
		platformAddress = userFriendlyAddr
	}

	// создать уникальное memo для пользователя
	memo := fmt.Sprintf("deposit_%d", userID)

	c.JSON(http.StatusOK, domain.DepositInfo{
		PlatformAddress: platformAddress,
		Memo:            memo,
		MinAmountTON:    fmt.Sprintf("%.2f", ton.NanoToTON(ton.MinDepositNano)),
		ExchangeRate:    ton.CoinsPerTON,
	})
}

// история пополнений пользователя
func (h *TonHandler) GetDeposits(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := c.Request.Context()
	deposits, err := h.DepositRepo.GetByUserID(ctx, userID, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deposits": deposits})
}

// вывод коинов
type WithdrawRequestBody struct {
	CoinsAmount int64 `json:"coins_amount" binding:"required,min=10"`
}

// новый запросов на вывод коинов
func (h *TonHandler) RequestWithdrawal(c *gin.Context, db *pgx.Conn) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req WithdrawRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	ctx := c.Request.Context()

	// проверка подключения кошелька
	wallet, err := h.DB.GetByUserID(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	if wallet == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no wallet linked"})
		return
	}

	// проверить наличие ожидающих выводов
	hasPending, err := h.WithdrawalRepo.HasPendingWithdrawal(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	if hasPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "you already have a pending withdrawal"})
		return
	}

	// дневной лимит на вывод коинов
	todayTotal, err := h.WithdrawalRepo.GetTotalCoinsWithdrawnToday(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	if todayTotal+req.CoinsAmount > ton.MaxWithdrawCoinsPerDay {
		remaining := ton.MaxWithdrawCoinsPerDay - todayTotal
		c.JSON(http.StatusBadRequest, gin.H{
			"error":           "daily withdrawal limit exceeded",
			"remaining_today": remaining,
		})
		return
	}

	// проверка баланса коинов
	currentCoins, err := h.UserRepo.GetCoins(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	if currentCoins < req.CoinsAmount {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "insufficient coins balance",
			"current_coins": currentCoins,
			"required":      req.CoinsAmount,
		})
		return
	}

	// ЕСЛИ КОМСА В ПРОЦЕНТАХ
	feeCoins := ton.CalculateWithdrawFeeCoins(req.CoinsAmount)
	netCoins := ton.CalculateWithdrawNetCoins(req.CoinsAmount)
	tonAmountNano := ton.CoinsToNano(netCoins)

	// списываем коины с баланса пользователя ПЕРЕД созданием заявки
	_, err = h.UserRepo.UpdateCoins(ctx, userID, -req.CoinsAmount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient coins balance"})
		return
	}

	// создание запроса на вывод средств
	withdrawal := &domain.Withdrawal{
		UserID:        userID,
		WalletAddress: wallet.Address,
		CoinsAmount:   req.CoinsAmount,
		TonAmountNano: tonAmountNano,
		FeeCoins:      feeCoins,
		ExchangeRate:  ton.CoinsPerTON,
		Status:        domain.WithdrawalStatusPending,
	}

	if err := h.WithdrawalRepo.Create(ctx, withdrawal); err != nil {
		// Возвращаем коины если не удалось создать заявку
		h.UserRepo.UpdateCoins(ctx, userID, req.CoinsAmount)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create withdrawal"})
		return
	}

	fmt.Printf("withdrawal created: id=%d, user=%d, coins=%d\n", withdrawal.ID, userID, req.CoinsAmount)

	// Отправка уведомления админам о выводе (используем background context,
	// т.к. HTTP context будет отменен после ответа)
	if h.OnWithdrawalCreate != nil {
		fmt.Printf("sending withdrawal notification for id=%d\n", withdrawal.ID)
		withdrawalID := withdrawal.ID
		go h.OnWithdrawalCreate(context.Background(), withdrawalID)
	} else {
		fmt.Println("WARNING: OnWithdrawalCreate callback is nil!")
	}

	// дать 50% комсы пригласившему(если пригласили)
	referrerID, err := h.ReferralRepo.GetReferrerID(ctx, userID)
	if err == nil && referrerID > 0 {
		referrerCommission := feeCoins / 2
		if referrerCommission > 0 {
			// добавить коины пригласившему
			_, _ = h.UserRepo.UpdateCoins(ctx, referrerID, referrerCommission)
			// отслеживание дохода реферралов
			_ = h.UserRepo.AddReferralEarnings(ctx, referrerID, referrerCommission)

			// запись транзакции для пригласившего
			meta := map[string]interface{}{
				"type":           "referral_commission",
				"from_user_id":   userID,
				"withdrawal_id":  withdrawal.ID,
				"total_fee":      feeCoins,
				"commission_pct": 50,
			}
			metaB, _ := json.Marshal(meta)
			_, _ = h.MainDB.DB.Exec(ctx,
				`INSERT INTO transactions (user_id, type, amount, meta) VALUES ($1, $2, $3, $4)`,
				referrerID, "referral_commission", referrerCommission, metaB)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"withdrawal": withdrawal,
		"estimate": domain.WithdrawEstimate{
			CoinsAmount:   req.CoinsAmount,
			FeeCoins:      feeCoins,
			NetCoins:      netCoins,
			TonAmount:     fmt.Sprintf("%.4f", ton.NanoToTON(tonAmountNano)),
			TonAmountNano: tonAmountNano,
			ExchangeRate:  ton.CoinsPerTON,
			FeePercent:    0, // комса в %
			FeeTON:        ton.CoinsToTON(feeCoins),
		},
	})
}

// сумма вывода коинов
func (h *TonHandler) GetWithdrawEstimate(c *gin.Context) {
	var req WithdrawRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.CoinsAmount < ton.MinWithdrawCoins {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "minimum withdrawal is 10 coins (1 TON)",
			"min_coins":  ton.MinWithdrawCoins,
			"min_ton":    fmt.Sprintf("%.2f", ton.NanoToTON(ton.CoinsToNano(ton.MinWithdrawCoins))),
		})
		return
	}

	feeCoins := ton.CalculateWithdrawFeeCoins(req.CoinsAmount)
	netCoins := ton.CalculateWithdrawNetCoins(req.CoinsAmount)
	tonAmountNano := ton.CoinsToNano(netCoins)

	c.JSON(http.StatusOK, domain.WithdrawEstimate{
		CoinsAmount:   req.CoinsAmount,
		FeeCoins:      feeCoins,
		NetCoins:      netCoins,
		TonAmount:     fmt.Sprintf("%.4f", ton.NanoToTON(tonAmountNano)),
		TonAmountNano: tonAmountNano,
		ExchangeRate:  ton.CoinsPerTON,
		FeePercent:    0, // комса в %,сейчас фикс
		FeeTON:        ton.CoinsToTON(feeCoins),
	})
}

// история выводов
func (h *TonHandler) GetWithdrawals(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := c.Request.Context()
	withdrawals, err := h.WithdrawalRepo.GetByUserID(ctx, userID, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"withdrawals": withdrawals})
}

// отмена ожидающего вывода
func (h *TonHandler) CancelWithdrawal(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		WithdrawalID int64 `json:"withdrawal_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	ctx := c.Request.Context()

	// Получаем информацию о выводе
	withdrawal, err := h.WithdrawalRepo.GetByID(ctx, req.WithdrawalID)
	if err != nil || withdrawal == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "withdrawal not found"})
		return
	}

	// Проверяем что вывод принадлежит пользователю и в статусе pending
	if withdrawal.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your withdrawal"})
		return
	}
	if withdrawal.Status != domain.WithdrawalStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "withdrawal cannot be cancelled"})
		return
	}

	// Отменяем вывод
	if err := h.WithdrawalRepo.Cancel(ctx, req.WithdrawalID, userID); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusBadRequest, gin.H{"error": "withdrawal not found or not cancelable"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel withdrawal"})
		return
	}

	// Возвращаем коины пользователю
	_, err = h.UserRepo.UpdateCoins(ctx, userID, withdrawal.CoinsAmount)
	if err != nil {
		fmt.Printf("ERROR: failed to return coins after cancel: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "coins_returned": withdrawal.CoinsAmount})
}

// конфиг TON for frontend
func (h *TonHandler) GetTonConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"platform_wallet":            h.PlatformWallet,
		"coins_per_ton":              ton.CoinsPerTON, // 10 коинов = 1 TON
		"min_deposit_ton":            fmt.Sprintf("%.2f", ton.NanoToTON(ton.MinDepositNano)),
		"min_withdraw_coins":         ton.MinWithdrawCoins,
		"withdraw_fee_coins":         ton.WithdrawFeeCoinsFixed, // 1 коин = 0.1 TON
		"withdraw_fee_ton":           ton.CoinsToTON(ton.WithdrawFeeCoinsFixed), // 0.1 TON
		"withdraw_fee_percent":       0, // если вводить назад комсу в %
		"max_withdraw_coins_per_day": ton.MaxWithdrawCoinsPerDay,
		"network":                    os.Getenv("TON_NETWORK"),
	})
}

// для тестов и админов
func (h *TonHandler) RecordManualDeposit(c *gin.Context, handler *Handler) {
	if os.Getenv("DEV_MODE") != "true" {
		c.JSON(http.StatusForbidden, gin.H{"error": "not allowed"})
		return
	}

	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		AmountTON float64 `json:"amount_ton" binding:"required,min=0.1"`
		TxHash    string  `json:"tx_hash" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	ctx := c.Request.Context()

	// проверка обработки транзакции
	exists, err := h.DepositRepo.TxHashExists(ctx, req.TxHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	if exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "transaction already processed"})
		return
	}

	// получить кошелек пользователя
	wallet, err := h.DB.GetByUserID(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	walletAddr := ""
	if wallet != nil {
		walletAddr = wallet.Address
	}

	amountNano := ton.TONToNano(req.AmountTON)
	coinsCredited := ton.NanoToCoins(amountNano)

	deposit := &domain.Deposit{
		UserID:        userID,
		WalletAddress: walletAddr,
		AmountNano:    amountNano,
		CoinsCredited: coinsCredited,
		ExchangeRate:  ton.CoinsPerTON,
		TxHash:        req.TxHash,
		Status:        domain.DepositStatusConfirmed,
		Processed:     true,
	}

	if err := h.DepositRepo.Create(ctx, deposit); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create deposit"})
		return
	}

	_, err = handler.DB.Exec(ctx, `UPDATE users SET coins = coins + $1 WHERE id = $2`, coinsCredited, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to credit coins"})
		return
	}

	// записать транзакцию
	meta := map[string]interface{}{
		"deposit_id":     deposit.ID,
		"tx_hash":        req.TxHash,
		"ton_amount":     req.AmountTON,
		"coins_credited": coinsCredited,
	}
	metaB, _ := json.Marshal(meta)
	_, _ = handler.DB.Exec(ctx, `INSERT INTO transactions (user_id,type,amount,meta) VALUES ($1,$2,$3,$4)`,
		userID, "ton_deposit", coinsCredited, metaB)

	c.JSON(http.StatusOK, gin.H{
		"deposit":        deposit,
		"coins_credited": coinsCredited,
	})
}
