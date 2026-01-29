package handlers

import (
	"net/http"
	"strconv"
	"time"

	"telegram_webapp/internal/repository"

	"github.com/gin-gonic/gin"
)

// Получение профиля юзера по id
func (h *Handler) Profile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	repo := repository.NewUserRepository(h.DB)
	ctx := c.Request.Context()
	user, err := repo.GetByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Получить статистику пользователя за последний месяц
	since := time.Now().AddDate(0, -1, 0)
	stats, _ := h.GameHistoryRepo.GetUserStats(ctx, id, since)

	c.JSON(http.StatusOK, gin.H{
		"id":         user.ID,
		"tg_id":      user.TgID,
		"username":   user.Username,
		"first_name": user.FirstName,
		"created_at": user.CreatedAt,
		"gems":       user.Gems,
		"coins":      user.Coins,
		"stats":      stats,
	})
}
// Получение истории игр текущего пользователя
func (h *Handler) MyGames(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	ctx := c.Request.Context()

	// История игр
	games, err := h.GameHistoryRepo.GetByUser(ctx, userID, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get games"})
		return
	}

	// Получение статистики за последний месяц
	since := time.Now().AddDate(0, -1, 0)
	stats, _ := h.GameHistoryRepo.GetUserStats(ctx, userID, since)

	c.JSON(http.StatusOK, gin.H{"games": games, "stats": stats})
}

func (h *Handler) TopUsers(c *gin.Context) {
	ctx := c.Request.Context()

	// Использовать GameHistoryRepo для получения месячной статистики !!
	top, err := h.GameHistoryRepo.GetTopUsers(ctx, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get top users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"top": top})
}

// бонус 10к гемов для юзеров,чей баланс 0 (единоразово)
func (h *Handler) ClaimBonus(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	ctx := c.Request.Context()
	_, err := h.DB.Exec(ctx, `UPDATE users SET gems = gems + 10000 WHERE id = $1 AND gems < 100`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to claim bonus"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Bonus claimed!"})
}
