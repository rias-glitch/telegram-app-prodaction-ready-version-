package game

import (
	"crypto/rand"
	"errors"
	"math/big"
	"sync"
	"time"
)

type CoinFlipProGame struct {
	ID           string    `json:"id"`
	UserID       int64     `json:"user_id"`
	Bet          int64     `json:"bet"`
	CurrentRound int       `json:"current_round"`
	MaxRounds    int       `json:"max_rounds"`
	Multiplier   float64   `json:"multiplier"`
	Status       string    `json:"status"` // active или cashed_out или lost
	WinAmount    int64     `json:"win_amount"`
	FlipHistory  []bool    `json:"flip_history"` // true = win, false = lose
	CreatedAt    time.Time `json:"created_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	mu           sync.RWMutex
}

const (
	CoinFlipProMaxRounds     = 10
	CoinFlipProStatusActive  = "active"
	CoinFlipProStatusCashedOut = "cashed_out"
	CoinFlipProStatusLost    = "lost"
)

// Множители
var CoinFlipProMultipliers = []float64{
	1.0,   // Пока не подброшена дальше от 1 до 10 раундов
	1.5,
	2.0,
	3.0,
	5.0,
	8.0,
	12.0,
	20.0,
	35.0,
	60.0,
	100.0,
}

// Создание новой игры
func NewCoinFlipProGame(id string, userID int64, bet int64) (*CoinFlipProGame, error) {
	if bet <= 0 {
		return nil, errors.New("bet must be positive")
	}

	return &CoinFlipProGame{
		ID:           id,
		UserID:       userID,
		Bet:          bet,
		CurrentRound: 0,
		MaxRounds:    CoinFlipProMaxRounds,
		Multiplier:   1.0,
		Status:       CoinFlipProStatusActive,
		FlipHistory:  []bool{},
		CreatedAt:    time.Now(),
	}, nil
}

// подброс монеты ! в текущем раунде !
func (g *CoinFlipProGame) Flip() (win bool, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.Status != CoinFlipProStatusActive {
		return false, errors.New("game is not active")
	}

	if g.CurrentRound >= g.MaxRounds {
		return false, errors.New("all rounds completed")
	}

	// 50 на 50 шансы(crypto random)
	n, err := rand.Int(rand.Reader, big.NewInt(2))
	if err != nil {
		return false, err
	}
	win = n.Int64() == 0

	g.FlipHistory = append(g.FlipHistory, win)

	if win {
		g.CurrentRound++
		g.Multiplier = CoinFlipProMultipliers[g.CurrentRound]

		//Автоматически забирается приз если прошло 10 раундов
		if g.CurrentRound >= g.MaxRounds {
			g.Status = CoinFlipProStatusCashedOut
			g.WinAmount = int64(float64(g.Bet) * g.Multiplier)
			now := time.Now()
			g.FinishedAt = &now
		}
	} else {
		// если проиграл,игра завершается
		g.Status = CoinFlipProStatusLost
		g.WinAmount = 0
		now := time.Now()
		g.FinishedAt = &now
	}

	return win, nil
}

// текущий выигрыш
func (g *CoinFlipProGame) CashOut() (int64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.Status != CoinFlipProStatusActive {
		return 0, errors.New("game is not active")
	}

	if g.CurrentRound == 0 {
		return 0, errors.New("must win at least one round before cashing out")
	}

	g.Status = CoinFlipProStatusCashedOut
	g.WinAmount = int64(float64(g.Bet) * g.Multiplier)
	now := time.Now()
	g.FinishedAt = &now

	return g.WinAmount, nil
}

// текущее состояние игры
func (g *CoinFlipProGame) GetState() map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	nextMultiplier := 0.0
	if g.CurrentRound < g.MaxRounds {
		nextMultiplier = CoinFlipProMultipliers[g.CurrentRound+1]
	}

	return map[string]interface{}{
		"id":              g.ID,
		"bet":             g.Bet,
		"current_round":   g.CurrentRound,
		"max_rounds":      g.MaxRounds,
		"multiplier":      g.Multiplier,
		"next_multiplier": nextMultiplier,
		"status":          g.Status,
		"win_amount":      g.WinAmount,
		"potential_win":   int64(float64(g.Bet) * g.Multiplier),
		"flip_history":    g.FlipHistory,
	}
}

// возвращает значение,указывающее, активна ли игра
func (g *CoinFlipProGame) IsActive() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.Status == CoinFlipProStatusActive
}

// чистая прибыль /победа - ставка
func (g *CoinFlipProGame) GetProfit() int64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.Status == CoinFlipProStatusCashedOut {
		return g.WinAmount - g.Bet
	}
	return -g.Bet // если проиграно
}

// Отображение текущего X пользователю
func GetCoinFlipProMultiplierTable() []map[string]interface{} {
	table := make([]map[string]interface{}, CoinFlipProMaxRounds)
	for i := 0; i < CoinFlipProMaxRounds; i++ {
		table[i] = map[string]interface{}{
			"round":      i + 1,
			"multiplier": CoinFlipProMultipliers[i+1],
		}
	}
	return table
}
