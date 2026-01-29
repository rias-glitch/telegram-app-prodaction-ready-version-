package game

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

type MinesGame struct {
	id       string
	players  [2]int64
	boards   map[int64]*Board
	moves    map[int64]int
	round    int
	result   *GameResult
	mu       sync.RWMutex
	// История ходов каждого игрока: playerID -> []MoveResult
	moveHistory map[int64][]MoveResult
	// Результат последнего раунда для отправки клиентам
	lastRoundResult *RoundResult
}

type MoveResult struct {
	Cell   int  `json:"cell"`
	HitMine bool `json:"hit_mine"`
	Round  int  `json:"round"`
}

type RoundResult struct {
	Round       int                  `json:"round"`
	PlayerMoves map[int64]MoveResult `json:"player_moves"`
}

type Board struct {
	mines [12]bool
}

// создает новую игру в мины
func NewMinesGame(id string, players [2]int64) *MinesGame {
	g := &MinesGame{
		id:          id,
		players:     players,
		boards:      make(map[int64]*Board),
		moves:       make(map[int64]int),
		moveHistory: make(map[int64][]MoveResult),
	}
	// Инициализируем пустую историю ходов для обоих игроков
	g.moveHistory[players[0]] = []MoveResult{}
	g.moveHistory[players[1]] = []MoveResult{}
	return g
}

func (g *MinesGame) Type() GameType { return TypeMines }
func (g *MinesGame) Players() [2]int64 { return g.players }
func (g *MinesGame) SetupTimeout() time.Duration { return 10 * time.Second }
func (g *MinesGame) TurnTimeout() time.Duration { return 15 * time.Second }

// устанавливает второго игрока
func (g *MinesGame) SetSecondPlayer(playerID int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.players[1] = playerID
	// Инициализируем историю ходов для нового игрока
	if g.moveHistory[playerID] == nil {
		g.moveHistory[playerID] = []MoveResult{}
	}
}

// проверяет завершена ли фаза подготовки
func (g *MinesGame) IsSetupComplete() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.isSetupCompleteUnlocked()
}

// внутренняя версия проверки подготовки без блокировки
func (g *MinesGame) isSetupCompleteUnlocked() bool {
	// Оба игрока должны иметь доски с расставленными минами
	if len(g.boards) < 2 {
		return false
	}
	for _, board := range g.boards {
		if board == nil {
			return false
		}
		// Проверяем что расставлена хотя бы одна мина
		hasMine := false
		for _, m := range board.mines {
			if m {
				hasMine = true
				break
			}
		}
		if !hasMine {
			return false
		}
	}
	return true
}

// обрабатывает действие игрока во время подготовки
func (g *MinesGame) HandleSetup(playerID int64, data interface{}) error {
	return g.HandleMove(playerID, data)  // Подготовка = тот же HandleMove
}

// обрабатывает ход игрока (как подготовку, так и игровой процесс)
func (g *MinesGame) HandleMove(playerID int64, data interface{}) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	log.Printf("MinesGame.HandleMove: player=%d data=%v dataType=%T setupComplete=%v", playerID, data, data, g.isSetupCompleteUnlocked())

	// Фаза подготовки - расстановка мин
	if !g.isSetupCompleteUnlocked() {
		// Если игрок уже расставил мины, не перезаписываем (бот не должен менять ход игрока)
		if g.boards[playerID] != nil && data == nil {
			log.Printf("MinesGame.HandleMove: player=%d already placed mines, skipping bot", playerID)
			return nil
		}

		positions, ok := data.([]int)
		if !ok || len(positions) != 4 {
			log.Printf("MinesGame.HandleMove: invalid setup data, using bot positions")
			// Бот расставляет мины случайно
			positions = []int{}
			used := make(map[int]bool)
			for len(positions) < 4 {
				pos := rand.Intn(12) + 1
				if !used[pos] {
					used[pos] = true
					positions = append(positions, pos)
				}
			}
		}

		board := &Board{}
		for _, pos := range positions {
			if pos >= 1 && pos <= 12 {
				board.mines[pos-1] = true
			}
		}
		g.boards[playerID] = board
		log.Printf("MinesGame.HandleMove: player=%d placed mines at %v, boards=%d", playerID, positions, len(g.boards))
		return nil
	}

	// Игровая фаза - выбор ячейки на доске противника
	// Если игрок уже сделал ход в этом раунде, не перезаписываем (бот не должен менять ход игрока)
	if _, alreadyMoved := g.moves[playerID]; alreadyMoved && data == nil {
		log.Printf("MinesGame.HandleMove: player=%d already moved this round, skipping bot", playerID)
		return nil
	}

	position, ok := data.(int)
	if !ok || position < 1 || position > 12 {
		log.Printf("MinesGame.HandleMove: invalid move data, using random position")
		// Бот выбирает случайную клетку
		position = rand.Intn(12) + 1
	}

	g.moves[playerID] = position
	log.Printf("MinesGame.HandleMove: player=%d selected cell %d, moves=%v", playerID, position, g.moves)
	return nil
}

// проверяет завершен ли текущий раунд
func (g *MinesGame) IsRoundComplete() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	complete := len(g.moves) == 2
	log.Printf("MinesGame.IsRoundComplete: moves=%v count=%d complete=%v", g.moves, len(g.moves), complete)
	return complete
}

// проверяет результат игры и определяет победителя
func (g *MinesGame) CheckResult() *GameResult {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.moves) < 2 {
		log.Printf("MinesGame.CheckResult: waiting for moves (have %d)", len(g.moves))
		return nil
	}

	g.round++

	p1, p2 := g.players[0], g.players[1]

	log.Printf("MinesGame.CheckResult: round=%d p1=%d pos=%d, p2=%d pos=%d", g.round, p1, g.moves[p1], p2, g.moves[p2])

	hit1 := g.boards[p2].mines[g.moves[p1]-1]
	hit2 := g.boards[p1].mines[g.moves[p2]-1]

	log.Printf("MinesGame.CheckResult: p1_hit=%v p2_hit=%v", hit1, hit2)

	// Сохраняем историю ходов для обоих игроков
	move1 := MoveResult{Cell: g.moves[p1], HitMine: hit1, Round: g.round}
	move2 := MoveResult{Cell: g.moves[p2], HitMine: hit2, Round: g.round}
	g.moveHistory[p1] = append(g.moveHistory[p1], move1)
	g.moveHistory[p2] = append(g.moveHistory[p2], move2)

	// Сохраняем результат последнего раунда для клиентов
	g.lastRoundResult = &RoundResult{
		Round: g.round,
		PlayerMoves: map[int64]MoveResult{
			p1: move1,
			p2: move2,
		},
	}

	// Один игрок подорвался
	if hit1 && !hit2 {
		log.Printf("MinesGame.CheckResult: p2 wins (p1 hit mine)")
		g.result = &GameResult{WinnerID: &p2, Reason: "opponent_hit_mine", Details: g.getResultDetails()}
		return g.result
	}
	if hit2 && !hit1 {
		log.Printf("MinesGame.CheckResult: p1 wins (p2 hit mine)")
		g.result = &GameResult{WinnerID: &p1, Reason: "opponent_hit_mine", Details: g.getResultDetails()}
		return g.result
	}

	// Оба игрока подорвались - ничья в раунде, но игра продолжается
	if hit1 && hit2 {
		log.Printf("MinesGame.CheckResult: both hit mines, round draw")
	}

	// Прошло 5 раундов
	if g.round >= 5 {
		log.Printf("MinesGame.CheckResult: draw (5 rounds)")
		g.result = &GameResult{WinnerID: nil, Reason: "draw", Details: g.getResultDetails()}
		return g.result
	}

	// Продолжаем игру
	log.Printf("MinesGame.CheckResult: continue to round %d", g.round+1)
	g.moves = make(map[int64]int)
	return nil
}

// возвращает детали результата игры
func (g *MinesGame) getResultDetails() map[string]interface{} {
	return map[string]interface{}{
		"rounds":      g.round,
		"moveHistory": g.moveHistory,
	}
}

// возвращает результат последнего раунда (для отправки клиентам)
func (g *MinesGame) GetLastRoundResult() *RoundResult {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.lastRoundResult
}

// возвращает историю ходов для игрока
func (g *MinesGame) GetMoveHistory(playerID int64) []MoveResult {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.moveHistory[playerID]
}

// возвращает текущий номер раунда
func (g *MinesGame) GetRound() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.round
}

// проверяет завершена ли игра
func (g *MinesGame) IsFinished() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.result != nil
}

// сериализует состояние игры для конкретного игрока
func (g *MinesGame) SerializeState(playerID int64) interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return map[string]interface{}{
		"type":   "mines",
		"round":  g.round,
		"result": g.result,
	}
}