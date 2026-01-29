package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"telegram_webapp/internal/game"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// управляет активными играми Mines Pro
type MinesProService struct {
	db          *pgxpool.Pool
	activeGames map[int64]*game.MinesPvEGame // userID -> game
	mu          sync.RWMutex
}

// создает новый сервис Mines Pro
func NewMinesProService(db *pgxpool.Pool) *MinesProService {
	s := &MinesProService{
		db:          db,
		activeGames: make(map[int64]*game.MinesPvEGame),
	}

	// запускаем горутину для очистки устаревших игр
	go s.cleanupExpiredGames()

	return s
}

// начинает новую игру Mines Pro
func (s *MinesProService) StartGame(ctx context.Context, userID int64, bet int64, minesCount int) (*game.MinesPvEGame, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// проверяем, есть ли у пользователя уже активная игра
	if existing, ok := s.activeGames[userID]; ok && existing.IsActive() {
		return nil, errors.New("у вас уже есть активная игра")
	}

	// начинаем транзакцию
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// проверяем и списываем баланс
	var balance int64
	if err := tx.QueryRow(ctx, `SELECT gems FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&balance); err != nil {
		return nil, err
	}
	if balance < bet {
		return nil, errors.New("недостаточно средств")
	}

	if _, err := tx.Exec(ctx, `UPDATE users SET gems = gems - $1 WHERE id=$2`, bet, userID); err != nil {
		return nil, err
	}

	// создаем игру
	gameID := uuid.New().String()[:8]
	g, err := game.NewMinesPvEGame(gameID, userID, bet, minesCount)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	s.activeGames[userID] = g
	return g, nil
}

// возвращает активную игру пользователя
func (s *MinesProService) GetActiveGame(userID int64) *game.MinesPvEGame {
	s.mu.RLock()
	defer s.mu.RUnlock()

	g, ok := s.activeGames[userID]
	if !ok || !g.IsActive() {
		return nil
	}
	return g
}

// открывает ячейку в активной игре пользователя
func (s *MinesProService) RevealCell(ctx context.Context, userID int64, cell int) (hitMine bool, g *game.MinesPvEGame, err error) {
	s.mu.Lock()
	g, ok := s.activeGames[userID]
	if !ok || !g.IsActive() {
		s.mu.Unlock()
		return false, nil, errors.New("нет активной игры")
	}
	s.mu.Unlock()

	hitMine, err = g.Reveal(cell)
	if err != nil {
		return false, g, err
	}

	// если игра завершена (взорвалась или все открыто), очищаем
	if !g.IsActive() {
		s.mu.Lock()
		delete(s.activeGames, userID)
		s.mu.Unlock()

		// если автоматически вывели (все открыто), начисляем выигрыш
		if g.Status == game.MinesProStatusCashedOut {
			_, _ = s.db.Exec(ctx, `UPDATE users SET gems = gems + $1 WHERE id=$2`, g.WinAmount, userID)
		}
	}

	return hitMine, g, nil
}

// выводит средства из активной игры пользователя
func (s *MinesProService) CashOut(ctx context.Context, userID int64) (*game.MinesPvEGame, error) {
	s.mu.Lock()
	g, ok := s.activeGames[userID]
	if !ok || !g.IsActive() {
		s.mu.Unlock()
		return nil, errors.New("нет активной игры")
	}
	s.mu.Unlock()

	winAmount, err := g.CashOut()
	if err != nil {
		return g, err
	}

	// начисляем выигрыш
	if _, err := s.db.Exec(ctx, `UPDATE users SET gems = gems + $1 WHERE id=$2`, winAmount, userID); err != nil {
		return g, err
	}

	// очищаем
	s.mu.Lock()
	delete(s.activeGames, userID)
	s.mu.Unlock()

	return g, nil
}

// удаляет игры старше 1 часа
func (s *MinesProService) cleanupExpiredGames() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for userID, g := range s.activeGames {
			// игры старше 1 часа считаются заброшенными
			if now.Sub(g.CreatedAt) > time.Hour {
				delete(s.activeGames, userID)
			}
		}
		s.mu.Unlock()
	}
}

// возвращает количество активных игр
func (s *MinesProService) GetActiveGamesCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.activeGames)
}