package handlers

import (
	"context"
	"net/http"
	"strconv"

	"telegram_webapp/internal/domain"

	"github.com/gin-gonic/gin"
)

// возвращает все активные квесты
func (h *Handler) GetQuests(c *gin.Context) {
	ctx := c.Request.Context()
	quests, err := h.QuestRepo.GetActiveQuests(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get quests"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"quests": quests})
}

// квест с прогрессом пользователя
type QuestWithProgress struct {
	Quest         *domain.Quest `json:"quest"`
	CurrentCount  int           `json:"current_count"`
	TargetCount   int           `json:"target_count"`
	Completed     bool          `json:"completed"`
	RewardClaimed bool          `json:"reward_claimed"`
	Progress      int           `json:"progress"`
	UserQuestID   *int64        `json:"user_quest_id,omitempty"`
}

// возвращает квесты пользователя с прогрессом
func (h *Handler) GetMyQuests(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	ctx := c.Request.Context()

	// Получаем все активные квесты
	allQuests, err := h.QuestRepo.GetActiveQuests(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get quests"})
		return
	}

	// Получаем прогресс пользователя
	userQuests, err := h.QuestRepo.GetUserQuests(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user quests"})
		return
	}

	// map для быстрого поиска прогресса
	progressMap := make(map[int64]*domain.UserQuestWithDetails)
	for _, uq := range userQuests {
		progressMap[uq.QuestID] = uq
	}

	// ответ с прогрессом для каждого квеста
	var result []QuestWithProgress
	for _, q := range allQuests {
		qwp := QuestWithProgress{
			Quest:       q,
			TargetCount: q.TargetCount,
		}

		if uq, exists := progressMap[q.ID]; exists {
			qwp.CurrentCount = uq.CurrentCount
			qwp.Completed = uq.Completed
			qwp.RewardClaimed = uq.RewardClaimed
			qwp.UserQuestID = &uq.ID
			qwp.Progress = uq.Progress(q.TargetCount)
		}

		result = append(result, qwp)
	}

	c.JSON(http.StatusOK, gin.H{"quests": result})
}

// забрать награду за выполненный квест
func (h *Handler) ClaimQuestReward(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	userQuestIDStr := c.Param("id")
	userQuestID, err := strconv.ParseInt(userQuestIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quest id"})
		return
	}

	ctx := c.Request.Context()

	// забрать награду
	reward, err := h.QuestRepo.ClaimReward(ctx, userQuestID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot claim reward"})
		return
	}

	// начислить награды пользователю
	var newGems, newCoins int64
	err = h.DB.QueryRow(ctx,
		`UPDATE users SET gems = gems + $1, coins = coins + $2 WHERE id = $3 RETURNING gems, coins`,
		reward.Gems, reward.Coins, userID,
	).Scan(&newGems, &newCoins)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update balance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reward_gems":  reward.Gems,
		"reward_coins": reward.Coins,
		"reward_gk":    reward.GK,
		"gems":         newGems,
		"coins":        newCoins,
	})
}


//!!!! вызывается после каждой игры для обновления прогресса квестов!!!!
func (h *Handler) updateQuestsAfterGameWithCtx(ctx context.Context, userID int64, gameType string, result string) {
	// получить все активные квесты
	quests, err := h.QuestRepo.GetActiveQuests(ctx)
	if err != nil {
		return
	}

	for _, quest := range quests {
		// проверка на соответствие типа игры
		if quest.GameType != nil && *quest.GameType != "any" && *quest.GameType != gameType {
			continue
		}

		// проверка типа действия
		shouldIncrement := false
		switch quest.ActionType {
		case domain.ActionTypePlay:
			shouldIncrement = true
		case domain.ActionTypeWin:
			shouldIncrement = (result == "win")
		case domain.ActionTypeLose:
			shouldIncrement = (result == "lose")
		}

		if shouldIncrement {
			_ = h.QuestRepo.IncrementProgress(ctx, userID, quest, 1)
		}
	}
}
