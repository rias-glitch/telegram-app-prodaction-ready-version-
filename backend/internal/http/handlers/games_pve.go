package handlers

import (
	"net/http"

	"telegram_webapp/internal/domain"
	"telegram_webapp/internal/game"
	"telegram_webapp/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// DiceRequest представляет запрос игры в кости (1-6)
type DiceRequest struct {
	Bet    int64  `json:"bet" binding:"required,min=1"`
	Target int    `json:"target"` // Обязательно для режима "exact", игнорируется для диапазонных режимов
	Mode   string `json:"mode" binding:"required,oneof=exact low high"`
}

// DiceResponse представляет ответ игры в кости (1-6)
type DiceResponse struct {
	Target     int     `json:"target"`
	Result     int     `json:"result"`
	Mode       string  `json:"mode"`
	Multiplier float64 `json:"multiplier"`
	WinChance  float64 `json:"win_chance"`
	Won        bool    `json:"won"`
	WinAmount  int64   `json:"win_amount"`
	Gems       int64   `json:"gems"`
}

// Dice обрабатывает эндпоинт игры в кости
func (h *Handler) Dice(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	var req DiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Валидация режима
	if req.Mode != game.DiceModeExact && req.Mode != game.DiceModeLow && req.Mode != game.DiceModeHigh {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mode must be 'exact', 'low', or 'high'"})
		return
	}

	// Валидация цели для режима exact
	if req.Mode == game.DiceModeExact {
		if req.Target < game.DiceMinTarget || req.Target > game.DiceMaxTarget {
			c.JSON(http.StatusBadRequest, gin.H{"error": "target must be between 1 and 6 for exact mode"})
			return
		}
	}

	ctx := c.Request.Context()

	// Начало транзакции
	tx, err := h.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Блокировка и проверка баланса
	var balance int64
	if err := tx.QueryRow(ctx, `SELECT gems FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&balance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	if balance < req.Bet {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient balance"})
		return
	}

	// Списание ставки
	if _, err := tx.Exec(ctx, `UPDATE users SET gems = gems - $1 WHERE id=$2`, req.Bet, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	// Играем в игру (кости 1-6 с режимом)
	diceGame := game.NewDiceGame(req.Target, req.Mode)
	diceGame.Roll()

	// Расчёт выигрыша
	winAmount := diceGame.CalculateWinAmount(req.Bet)
	if winAmount > 0 {
		if _, err := tx.Exec(ctx, `UPDATE users SET gems = gems + $1 WHERE id=$2`, winAmount, userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
	}

	// Запись транзакции
	netAmount := winAmount - req.Bet
	meta := diceGame.ToDetails()
	meta["bet"] = req.Bet
	meta["win_amount"] = winAmount
	txRecord := &domain.Transaction{
		UserID: userID,
		Type:   "dice",
		Amount: netAmount,
		Meta:   meta,
	}
	if err := h.TransactionRepo.CreateWithTx(ctx, tx, txRecord); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	// Получение нового баланса
	var newBalance int64
	if err := tx.QueryRow(ctx, `SELECT gems FROM users WHERE id=$1`, userID).Scan(&newBalance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	// Запись истории игры
	var gameResult domain.GameResult
	if diceGame.Won {
		gameResult = domain.GameResultWin
	} else {
		gameResult = domain.GameResultLose
	}
	go h.RecordGameResult(userID, domain.GameTypeDice, domain.GameModePVE, gameResult, req.Bet, netAmount, meta)

	c.JSON(http.StatusOK, DiceResponse{
		Target:     diceGame.Target,
		Result:     diceGame.Result,
		Mode:       diceGame.Mode,
		Multiplier: diceGame.Multiplier,
		WinChance:  diceGame.WinChance(),
		Won:        diceGame.Won,
		WinAmount:  winAmount,
		Gems:       newBalance,
	})
}

// DiceInfo возвращает конфигурацию игры в кости (1-6)
func (h *Handler) DiceInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"min_target": game.DiceMinTarget, // 1
		"max_target": game.DiceMaxTarget, // 6
		"sides":      game.DiceSides,     // 6
		"modes": []gin.H{
			{
				"mode":        game.DiceModeExact,
				"name":        "Exact Number",
				"description": "Pick a specific number (1-6)",
				"multiplier":  game.DiceMultiplierExact,
				"win_chance":  16.67,
			},
			{
				"mode":        game.DiceModeLow,
				"name":        "Low (1-3)",
				"description": "Win if dice shows 1, 2, or 3",
				"multiplier":  game.DiceMultiplierRange,
				"win_chance":  50.0,
			},
			{
				"mode":        game.DiceModeHigh,
				"name":        "High (4-6)",
				"description": "Win if dice shows 4, 5, or 6",
				"multiplier":  game.DiceMultiplierRange,
				"win_chance":  50.0,
			},
		},
	})
}

// WheelRequest представляет запрос игры в колесо фортуны
type WheelRequest struct {
	Bet int64 `json:"bet" binding:"required,min=1"`
}

// WheelResponse представляет ответ игры в колесо фортуны
type WheelResponse struct {
	SegmentID  int     `json:"segment_id"`
	Multiplier float64 `json:"multiplier"`
	Color      string  `json:"color"`
	Label      string  `json:"label"`
	SpinAngle  float64 `json:"spin_angle"`
	Bet        int64   `json:"bet"`
	WinAmount  int64   `json:"win_amount"`
	Gems       int64   `json:"gems"`
}

// Wheel обрабатывает эндпоинт игры в колесо фортуны
func (h *Handler) Wheel(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	var req WheelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Начало транзакции
	tx, err := h.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Блокировка и проверка баланса
	var balance int64
	if err := tx.QueryRow(ctx, `SELECT gems FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&balance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	if balance < req.Bet {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient balance"})
		return
	}

	// Списание ставки
	if _, err := tx.Exec(ctx, `UPDATE users SET gems = gems - $1 WHERE id=$2`, req.Bet, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	// Играем в игру
	wheelGame := game.NewWheelGame()
	result := wheelGame.Spin()

	// Расчёт выигрыша
	winAmount := wheelGame.CalculateWinAmount(req.Bet)
	if winAmount > 0 {
		if _, err := tx.Exec(ctx, `UPDATE users SET gems = gems + $1 WHERE id=$2`, winAmount, userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
	}

	// Запись транзакции
	netAmount := winAmount - req.Bet
	meta := wheelGame.ToDetails()
	meta["bet"] = req.Bet
	meta["win_amount"] = winAmount
	txRecord := &domain.Transaction{
		UserID: userID,
		Type:   "wheel",
		Amount: netAmount,
		Meta:   meta,
	}
	if err := h.TransactionRepo.CreateWithTx(ctx, tx, txRecord); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	// Получение нового баланса
	var newBalance int64
	if err := tx.QueryRow(ctx, `SELECT gems FROM users WHERE id=$1`, userID).Scan(&newBalance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	// Запись истории игры
	var gameResult domain.GameResult
	if result.Multiplier >= 1.0 {
		gameResult = domain.GameResultWin
	} else {
		gameResult = domain.GameResultLose
	}
	go h.RecordGameResult(userID, domain.GameTypeWheel, domain.GameModePVE, gameResult, req.Bet, netAmount, meta)

	c.JSON(http.StatusOK, WheelResponse{
		SegmentID:  result.ID,
		Multiplier: result.Multiplier,
		Color:      result.Color,
		Label:      result.Label,
		SpinAngle:  wheelGame.SpinAngle,
		Bet:        req.Bet,
		WinAmount:  winAmount,
		Gems:       newBalance,
	})
}

// WheelInfo возвращает конфигурацию колеса для фронтенда
func (h *Handler) WheelInfo(c *gin.Context) {
	wheelGame := game.NewWheelGame()

	c.JSON(http.StatusOK, gin.H{
		"segments":        wheelGame.Segments,
		"expected_return": wheelGame.GetExpectedReturn(),
	})
}

// ============ MINES PRO ============

// MinesProStartRequest представляет запрос на начало игры
type MinesProStartRequest struct {
	Bet        int64 `json:"bet" binding:"required,min=1"`
	MinesCount int   `json:"mines_count" binding:"required,min=1,max=24"`
}

// MinesProRevealRequest представляет запрос на открытие ячейки
type MinesProRevealRequest struct {
	Cell *int `json:"cell" binding:"required,min=0,max=24"`
}

// MinesProStart запускает новую игру Mines Pro
func (h *Handler) MinesProStart(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	var req MinesProStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	g, err := h.MinesProService.StartGame(ctx, userID, req.Bet, req.MinesCount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, g.GetState())
}

// MinesProReveal открывает ячейку в активной игре
func (h *Handler) MinesProReveal(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	var req MinesProRevealRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if req.Cell == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cell is required"})
		return
	}

	ctx := c.Request.Context()
	hitMine, g, err := h.MinesProService.RevealCell(ctx, userID, *req.Cell)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	state := g.GetState()
	state["hit_mine"] = hitMine

	// Записываем игру если завершена
	if !g.IsActive() {
		var result domain.GameResult
		if g.Status == game.MinesProStatusCashedOut {
			result = domain.GameResultWin
		} else {
			result = domain.GameResultLose
		}
		go h.RecordGameResult(userID, domain.GameTypeMinesPro, domain.GameModePVE, result, g.Bet, g.GetProfit(), g.ToDetails())

		// Запись транзакции
		meta := g.ToDetails()
		meta["bet"] = g.Bet
		meta["win_amount"] = g.WinAmount
		txRecord := &domain.Transaction{
			UserID: userID,
			Type:   "mines_pro",
			Amount: g.GetProfit(),
			Meta:   meta,
		}
		_ = h.TransactionRepo.Create(ctx, txRecord)
	}

	// Получение текущего баланса
	user, _ := repository.NewUserRepository(h.DB).GetByID(ctx, userID)
	var balance int64
	if user != nil {
		balance = user.Gems
	}
	state["gems"] = balance

	c.JSON(http.StatusOK, state)
}

// MinesProCashOut забирает выигрыш в активной игре
func (h *Handler) MinesProCashOut(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	ctx := c.Request.Context()
	g, err := h.MinesProService.CashOut(ctx, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Запись истории игры
	go h.RecordGameResult(userID, domain.GameTypeMinesPro, domain.GameModePVE, domain.GameResultWin, g.Bet, g.GetProfit(), g.ToDetails())

	// Запись транзакции
	meta := g.ToDetails()
	meta["bet"] = g.Bet
	meta["win_amount"] = g.WinAmount
	txRecord := &domain.Transaction{
		UserID: userID,
		Type:   "mines_pro",
		Amount: g.GetProfit(),
		Meta:   meta,
	}
	_ = h.TransactionRepo.Create(ctx, txRecord)

	// Получение текущего баланса
	user, _ := repository.NewUserRepository(h.DB).GetByID(ctx, userID)
	var balance int64
	if user != nil {
		balance = user.Gems
	}

	state := g.GetState()
	state["gems"] = balance

	c.JSON(http.StatusOK, state)
}

// MinesProState возвращает текущее состояние игры
func (h *Handler) MinesProState(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	g := h.MinesProService.GetActiveGame(userID)
	if g == nil {
		c.JSON(http.StatusOK, gin.H{"active": false})
		return
	}

	state := g.GetState()
	state["active"] = true
	c.JSON(http.StatusOK, state)
}

// MinesProInfo возвращает конфигурацию игры
func (h *Handler) MinesProInfo(c *gin.Context) {
	// Таблицы множителей для разного количества мин
	tables := make(map[int][]float64)
	for mines := 1; mines <= 24; mines++ {
		tables[mines] = game.MultiplierTable(mines)
	}

	c.JSON(http.StatusOK, gin.H{
		"board_size":        game.MinesProBoardSize,
		"min_mines":         game.MinesProMinMines,
		"max_mines":         game.MinesProMaxMines,
		"multiplier_tables": tables,
	})
}

// ============ COINFLIP PRO ============

// CoinFlipProStartRequest представляет запрос на начало игры
type CoinFlipProStartRequest struct {
	Bet int64 `json:"bet" binding:"required,min=1"`
}

// CoinFlipProStart запускает новую игру CoinFlip Pro
func (h *Handler) CoinFlipProStart(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	var req CoinFlipProStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	g, err := h.CoinFlipProService.StartGame(ctx, userID, req.Bet)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, g.GetState())
}

// CoinFlipProFlip выполняет бросок монеты в активной игре
func (h *Handler) CoinFlipProFlip(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	ctx := c.Request.Context()
	win, g, err := h.CoinFlipProService.Flip(ctx, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	state := g.GetState()
	state["flip_win"] = win

	// Записываем игру если завершена
	if !g.IsActive() {
		var result domain.GameResult
		if g.Status == game.CoinFlipProStatusCashedOut {
			result = domain.GameResultWin
		} else {
			result = domain.GameResultLose
		}
		details := map[string]interface{}{
			"rounds":       g.CurrentRound,
			"multiplier":   g.Multiplier,
			"flip_history": g.FlipHistory,
		}
		go h.RecordGameResult(userID, domain.GameTypeCoinflip, domain.GameModePVE, result, g.Bet, g.GetProfit(), details)

		// Запись транзакции
		meta := details
		meta["bet"] = g.Bet
		meta["win_amount"] = g.WinAmount
		txRecord := &domain.Transaction{
			UserID: userID,
			Type:   "coinflip_pro",
			Amount: g.GetProfit(),
			Meta:   meta,
		}
		_ = h.TransactionRepo.Create(ctx, txRecord)
	}

	// Получение текущего баланса
	user, _ := repository.NewUserRepository(h.DB).GetByID(ctx, userID)
	var balance int64
	if user != nil {
		balance = user.Gems
	}
	state["gems"] = balance

	c.JSON(http.StatusOK, state)
}

// CoinFlipProCashOut забирает выигрыш в активной игре
func (h *Handler) CoinFlipProCashOut(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	ctx := c.Request.Context()
	g, err := h.CoinFlipProService.CashOut(ctx, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Запись истории игры
	details := map[string]interface{}{
		"rounds":       g.CurrentRound,
		"multiplier":   g.Multiplier,
		"flip_history": g.FlipHistory,
	}
	go h.RecordGameResult(userID, domain.GameTypeCoinflip, domain.GameModePVE, domain.GameResultWin, g.Bet, g.GetProfit(), details)

	// Запись транзакции
	meta := details
	meta["bet"] = g.Bet
	meta["win_amount"] = g.WinAmount
	txRecord := &domain.Transaction{
		UserID: userID,
		Type:   "coinflip_pro",
		Amount: g.GetProfit(),
		Meta:   meta,
	}
	_ = h.TransactionRepo.Create(ctx, txRecord)

	// Получение текущего баланса
	user, _ := repository.NewUserRepository(h.DB).GetByID(ctx, userID)
	var balance int64
	if user != nil {
		balance = user.Gems
	}

	state := g.GetState()
	state["gems"] = balance

	c.JSON(http.StatusOK, state)
}

// CoinFlipProState возвращает текущее состояние игры
func (h *Handler) CoinFlipProState(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	g := h.CoinFlipProService.GetActiveGame(userID)
	if g == nil {
		c.JSON(http.StatusOK, gin.H{"active": false})
		return
	}

	state := g.GetState()
	state["active"] = true
	c.JSON(http.StatusOK, state)
}

// CoinFlipProInfo возвращает конфигурацию игры
func (h *Handler) CoinFlipProInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"max_rounds":  game.CoinFlipProMaxRounds,
		"multipliers": game.GetCoinFlipProMultiplierTable(),
	})
}

