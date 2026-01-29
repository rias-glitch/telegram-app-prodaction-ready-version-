package game

import (
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"
)

type RPSGame struct {
	id        string
	players   [2]int64
	moves     map[int64]string
	lastMoves map[int64]string // сохраняем ходы при ничьей для отображения
	round     int
	result    *GameResult
	mu        sync.RWMutex
}

// создает новую игру камень-ножницы-бумага
func NewRPSGame(id string, players [2]int64) *RPSGame {
	return &RPSGame{
		id:      id,
		players: players,
		moves:   make(map[int64]string),
	}
}

func (g *RPSGame) Type() GameType {
	return TypeRPS
}

func (g *RPSGame) Players() [2]int64 {
	return g.players
}

// возвращает таймаут для фазы подготовки (таймаут поиска противника)
func (g *RPSGame) SetupTimeout() time.Duration {
	return 10 * time.Second // Отменяем игру если противник не найден за 10 секунд
}

// устанавливает второго игрока
func (g *RPSGame) SetSecondPlayer(playerID int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.players[1] = playerID
}

// обработка действий игрока во время подготовки
func (g *RPSGame) HandleSetup(playerID int64, data interface{}) error {
	return nil // подготовка не требуется
}

// проверяет завершена ли фаза подготовки
func (g *RPSGame) IsSetupComplete() bool {
	return true // всегда готово
}

// возвращает таймаут для хода
func (g *RPSGame) TurnTimeout() time.Duration {
	return 20 * time.Second
}

// обрабатывает ход игрока
func (g *RPSGame) HandleMove(playerID int64, data interface{}) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Если игрок уже сделал ход, не перезаписываем (бот не должен менять ход игрока)
	if _, ok := g.moves[playerID]; ok {
		if data == nil {
			log.Printf("RPSGame.HandleMove: игрок=%d уже сходил, пропускаем бота", playerID)
			return nil
		}
		return errors.New("уже сделан ход")
	}

	move, ok := data.(string)
	if !ok {
		// Бот делает случайный ход при таймауте
		log.Printf("RPSGame.HandleMove: неверные данные хода, используем случайный ход")
		moves := []string{"rock", "paper", "scissors"}
		move = moves[rand.Intn(3)]
	}

	if move != "rock" && move != "paper" && move != "scissors" {
		return errors.New("неверное значение хода")
	}

	g.moves[playerID] = move
	log.Printf("RPSGame.HandleMove: игра=%s игрок=%d ход=%s ходы=%v", g.id, playerID, move, g.moves)
	return nil
}

// проверяет завершен ли текущий раунд
func (g *RPSGame) IsRoundComplete() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.moves) == 2
}

// проверяет результат игры и определяет победителя
func (g *RPSGame) CheckResult() *GameResult {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.moves) < 2 {
		log.Printf("RPSGame.CheckResult: еще недостаточно ходов: ходы=%v", g.moves)
		return nil
	}

	g.round++
	p1, p2 := g.players[0], g.players[1]
	move1, move2 := g.moves[p1], g.moves[p2]

	log.Printf("RPSGame.CheckResult: раунд=%d p1=%d ход=%s, p2=%d ход=%s", g.round, p1, move1, p2, move2)

	outcome := decide(move1, move2)

	// Кто-то выиграл
	if outcome == "win" {
		log.Printf("RPSGame.CheckResult: p1 побеждает")
		g.result = &GameResult{
			WinnerID: &p1,
			Reason:   "game_complete",
			Details: map[string]interface{}{
				"moves":        g.moves,
				"yourMove":     move1,
				"opponentMove": move2,
			},
		}
		return g.result
	} else if outcome == "lose" {
		log.Printf("RPSGame.CheckResult: p2 побеждает")
		g.result = &GameResult{
			WinnerID: &p2,
			Reason:   "game_complete",
			Details: map[string]interface{}{
				"moves":        g.moves,
				"yourMove":     move2,
				"opponentMove": move1,
			},
		}
		return g.result
	}

	// Ничья - проверяем достигли ли максимального количества раундов
	log.Printf("RPSGame.CheckResult: раунд %d завершился вничью", g.round)

	if g.round >= 5 {
		log.Printf("RPSGame.CheckResult: 5 раундов вничью, игра завершается ничьей")
		g.result = &GameResult{
			WinnerID: nil,
			Reason:   "draw_after_5_rounds",
			Details: map[string]interface{}{
				"rounds": g.round,
			},
		}
		return g.result
	}

	// Переходим к следующему раунду - сохраняем и очищаем ходы
	log.Printf("RPSGame.CheckResult: переходим к раунду %d", g.round+1)
	g.lastMoves = make(map[int64]string)
	for k, v := range g.moves {
		g.lastMoves[k] = v
	}
	g.moves = make(map[int64]string)
	return nil
}

// GetLastMoves возвращает последние ходы при ничьей
func (g *RPSGame) GetLastMoves() map[int64]string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.lastMoves
}

// GetRound возвращает текущий раунд
func (g *RPSGame) GetRound() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.round
}

// проверяет завершена ли игра
func (g *RPSGame) IsFinished() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.result != nil
}

// сериализует состояние игры для конкретного игрока
func (g *RPSGame) SerializeState(playerID int64) interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return map[string]interface{}{
		"type":   "rps",
		"round":  g.round,
		"moves":  len(g.moves),
		"result": g.result,
	}
}

// определяет результат одного раунда камень-ножницы-бумага
func decide(moveA, moveB string) string {
	if moveA == moveB {
		return "draw"
	}

	switch moveA {
	case "rock":
		if moveB == "scissors" {
			return "win"
		}
	case "paper":
		if moveB == "rock" {
			return "win"
		}
	case "scissors":
		if moveB == "paper" {
			return "win"
		}
	}

	return "lose"
}