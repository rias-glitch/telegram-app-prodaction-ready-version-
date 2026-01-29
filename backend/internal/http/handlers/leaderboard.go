package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// список 100 лучших игроков
func (h *Handler) GetLeaderboard(c *gin.Context) {
	top, err := h.UserRepo.GetMonthlyTop(c.Request.Context(), 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get leaderboard"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"leaderboard": top,
		"period":      "monthly",
	})
}

// рейтинг пользователя в топе
func (h *Handler) GetMyRank(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	rank, winsCount, err := h.UserRepo.GetUserRank(c.Request.Context(), userID)
	if err != nil {
		// если не сыграно ни одной игры ранг - 0
		c.JSON(http.StatusOK, gin.H{
			"rank":       0,
			"wins_count": 0,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rank":       rank,
		"wins_count": winsCount,
	})
}
