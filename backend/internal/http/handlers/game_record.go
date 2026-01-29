package handlers

import (
	"context"
	"time"

	"telegram_webapp/internal/domain"
)

// Записывает результат игры в историю и обновляет квесты
func (h *Handler) RecordGameResult(userID int64, gameType domain.GameType, mode domain.GameMode, result domain.GameResult, betAmount, winAmount int64, details map[string]interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Запись в game_history
	gh := &domain.GameHistory{
		UserID:    userID,
		GameType:  gameType,
		Mode:      mode,
		Result:    result,
		BetAmount: betAmount,
		WinAmount: winAmount,
		Details:   details,
	}
	_ = h.GameHistoryRepo.Create(ctx, gh)

	// Обновляем прогресс квестов с тем же контекстом
	h.updateQuestsAfterGameWithCtx(ctx, userID, string(gameType), string(result))
}

// Записывает результат PvP игры для обоих игроков
func (h *Handler) RecordPVPGameResult(playerA, playerB int64, gameType domain.GameType, roomID string, winnerID *int64, betAmount int64, details map[string]interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Записываем для игрока A
	var resultA domain.GameResult
	var winAmountA int64
	if winnerID == nil {
		resultA = domain.GameResultDraw
		winAmountA = 0
	} else if *winnerID == playerA {
		resultA = domain.GameResultWin
		winAmountA = betAmount
	} else {
		resultA = domain.GameResultLose
		winAmountA = -betAmount
	}

	ghA := &domain.GameHistory{
		UserID:     playerA,
		GameType:   gameType,
		Mode:       domain.GameModePVP,
		OpponentID: &playerB,
		RoomID:     &roomID,
		Result:     resultA,
		BetAmount:  betAmount,
		WinAmount:  winAmountA,
		Details:    details,
	}
	_ = h.GameHistoryRepo.Create(ctx, ghA)

	// Записываем для игрока B
	var resultB domain.GameResult
	var winAmountB int64
	if winnerID == nil {
		resultB = domain.GameResultDraw
		winAmountB = 0
	} else if *winnerID == playerB {
		resultB = domain.GameResultWin
		winAmountB = betAmount
	} else {
		resultB = domain.GameResultLose
		winAmountB = -betAmount
	}

	ghB := &domain.GameHistory{
		UserID:     playerB,
		GameType:   gameType,
		Mode:       domain.GameModePVP,
		OpponentID: &playerA,
		RoomID:     &roomID,
		Result:     resultB,
		BetAmount:  betAmount,
		WinAmount:  winAmountB,
		Details:    details,
	}
	_ = h.GameHistoryRepo.Create(ctx, ghB)

	// Обновляем квесты для обоих с тем же контекстом
	h.updateQuestsAfterGameWithCtx(ctx, playerA, string(gameType), string(resultA))
	h.updateQuestsAfterGameWithCtx(ctx, playerB, string(gameType), string(resultB))
}
