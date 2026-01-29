package game

import (
	"crypto/rand"
	"errors"
	"math"
	"math/big"
	"sync"
	"time"
)

// MinesPvEGame представляет одиночную игру в сапёра
// Игрок может открывать несколько ячеек и кэшировать в любое время
type MinesPvEGame struct {
	ID             string    `json:"id"`
	UserID         int64     `json:"user_id"`
	BoardSize      int       `json:"board_size"`      // По умолчанию 25 (5x5)
	MinesCount     int       `json:"mines_count"`     // 1-24 мин
	Bet            int64     `json:"bet"`
	Mines          []int     `json:"-"`               // Позиции мин (скрыты от клиента)
	RevealedCells  []int     `json:"revealed_cells"`  // Открытые игроком ячейки
	Multiplier     float64   `json:"multiplier"`      // Текущий множитель
	NextMultiplier float64   `json:"next_multiplier"` // Множитель если следующая ячейка безопасна
	Status         string    `json:"status"`          // active, cashed_out, exploded
	WinAmount      int64     `json:"win_amount"`      // Выигранная сумма (0 при взрыве)
	CreatedAt      time.Time `json:"created_at"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
	mu             sync.RWMutex
}

const (
	MinesProBoardSize       = 25 // 5x5 сетка
	MinesProMinMines        = 1
	MinesProMaxMines        = 24
	MinesProStatusActive    = "active"
	MinesProStatusCashedOut = "cashed_out"
	MinesProStatusExploded  = "exploded"
)

// создает новую игру Mines Pro
func NewMinesPvEGame(id string, userID int64, bet int64, minesCount int) (*MinesPvEGame, error) {
	if minesCount < MinesProMinMines || minesCount > MinesProMaxMines {
		return nil, errors.New("количество мин должно быть от 1 до 24")
	}
	if bet <= 0 {
		return nil, errors.New("ставка должна быть положительной")
	}

	g := &MinesPvEGame{
		ID:            id,
		UserID:        userID,
		BoardSize:     MinesProBoardSize,
		MinesCount:    minesCount,
		Bet:           bet,
		RevealedCells: []int{},
		Multiplier:    1.0,
		Status:        MinesProStatusActive,
		CreatedAt:     time.Now(),
	}

	// Генерируем случайные позиции мин
	g.Mines = g.generateMines()

	// Рассчитываем начальный следующий множитель
	g.NextMultiplier = g.calculateNextMultiplier()

	return g, nil
}

// генерирует случайные позиции мин
func (g *MinesPvEGame) generateMines() []int {
	mines := make([]int, 0, g.MinesCount)
	used := make(map[int]bool)

	for len(mines) < g.MinesCount {
		max := big.NewInt(int64(g.BoardSize))
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			n = big.NewInt(int64(len(mines))) // Запасной вариант
		}
		pos := int(n.Int64())
		if !used[pos] {
			used[pos] = true
			mines = append(mines, pos)
		}
	}

	return mines
}

//  рассчитывает множитель на основе открытых безопасных ячеек
func (g *MinesPvEGame) calculateMultiplier() float64 {
	if len(g.RevealedCells) == 0 {
		return 1.0
	}

	totalCells := g.BoardSize
	safeCells := totalCells - g.MinesCount
	revealed := len(g.RevealedCells)

	// Прозрачная формула:
	// Множитель = произведение (totalRemaining / safeRemaining) для каждого открытия
	multiplier := 1.0
	for i := 0; i < revealed; i++ {
		totalRemaining := float64(totalCells - i)
		safeRemaining := float64(safeCells - i)
		if safeRemaining <= 0 {
			break
		}
		multiplier *= totalRemaining / safeRemaining
	}

	// Округление до 2 десятичных знаков
	return math.Floor(multiplier*100) / 100
}

// рассчитывает каким будет множитель если следующая ячейка безопасна
func (g *MinesPvEGame) calculateNextMultiplier() float64 {
	totalCells := g.BoardSize
	safeCells := totalCells - g.MinesCount
	revealed := len(g.RevealedCells)

	// Если все безопасные ячейки открыты, больше ходов нет
	if revealed >= safeCells {
		return g.Multiplier
	}

	// Рассчитываем множитель после еще одного открытия
	multiplier := 1.0
	for i := 0; i <= revealed; i++ {
		totalRemaining := float64(totalCells - i)
		safeRemaining := float64(safeCells - i)
		if safeRemaining <= 0 {
			break
		}
		multiplier *= totalRemaining / safeRemaining
	}

	return math.Floor(multiplier*100) / 100
}

// Reveal открывает ячейку на поле
func (g *MinesPvEGame) Reveal(cell int) (hitMine bool, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.Status != MinesProStatusActive {
		return false, errors.New("игра не активна")
	}

	if cell < 0 || cell >= g.BoardSize {
		return false, errors.New("неверная позиция ячейки")
	}

	// Проверяем если уже открыта
	for _, c := range g.RevealedCells {
		if c == cell {
			return false, errors.New("ячейка уже открыта")
		}
	}

	// Проверяем если попали на мину
	for _, m := range g.Mines {
		if m == cell {
			// ВЗРЫВ!
			g.Status = MinesProStatusExploded
			g.WinAmount = 0
			now := time.Now()
			g.FinishedAt = &now
			return true, nil
		}
	}

	// Безопасная ячейка
	g.RevealedCells = append(g.RevealedCells, cell)
	g.Multiplier = g.calculateMultiplier()
	g.NextMultiplier = g.calculateNextMultiplier()

	// Проверяем если все безопасные ячейки открыты (авто кэшаут)
	safeCells := g.BoardSize - g.MinesCount
	if len(g.RevealedCells) >= safeCells {
		g.Status = MinesProStatusCashedOut
		g.WinAmount = int64(float64(g.Bet) * g.Multiplier)
		now := time.Now()
		g.FinishedAt = &now
	}

	return false, nil
}

// кэширует текущий выигрыш
func (g *MinesPvEGame) CashOut() (int64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.Status != MinesProStatusActive {
		return 0, errors.New("игра не активна")
	}

	if len(g.RevealedCells) == 0 {
		return 0, errors.New("нужно открыть хотя бы одну ячейку перед кэшаутом")
	}

	g.Status = MinesProStatusCashedOut
	g.WinAmount = int64(float64(g.Bet) * g.Multiplier)
	now := time.Now()
	g.FinishedAt = &now

	return g.WinAmount, nil
}

// возвращает текущее состояние игры (безопасно для клиента)
func (g *MinesPvEGame) GetState() map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	state := map[string]interface{}{
		"id":              g.ID,
		"board_size":      g.BoardSize,
		"mines_count":     g.MinesCount,
		"bet":             g.Bet,
		"revealed_cells":  g.RevealedCells,
		"multiplier":      g.Multiplier,
		"next_multiplier": g.NextMultiplier,
		"status":          g.Status,
		"win_amount":      g.WinAmount,
		"potential_win":   int64(float64(g.Bet) * g.Multiplier),
	}

	// Показываем мины только если игра окончена
	if g.Status != MinesProStatusActive {
		state["mines"] = g.Mines
	}

	return state
}

// активна ли игра
func (g *MinesPvEGame) IsActive() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.Status == MinesProStatusActive
}

// возвращает чистую прибыль (выигрыш - ставка)
func (g *MinesPvEGame) GetProfit() int64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.Status == MinesProStatusCashedOut {
		return g.WinAmount - g.Bet
	}
	return -g.Bet // Проиграл
}

//  возвращает детали игры для хранения
func (g *MinesPvEGame) ToDetails() map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return map[string]interface{}{
		"board_size":     g.BoardSize,
		"mines_count":    g.MinesCount,
		"mines":          g.Mines,
		"revealed_cells": g.RevealedCells,
		"multiplier":     g.Multiplier,
		"status":         g.Status,
	}
}

// возвращает таблицу множителей для разных количеств открытий
func MultiplierTable(minesCount int) []float64 {
	boardSize := MinesProBoardSize
	safeCells := boardSize - minesCount

	table := make([]float64, safeCells)

	for reveals := 1; reveals <= safeCells; reveals++ {
		multiplier := 1.0
		for i := 0; i < reveals; i++ {
			totalRemaining := float64(boardSize - i)
			safeRemaining := float64(safeCells - i)
			multiplier *= totalRemaining / safeRemaining
		}
		table[reveals-1] = math.Floor(multiplier*100) / 100
	}

	return table
}