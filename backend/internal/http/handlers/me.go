package handlers

import (
	"net/http"

	"telegram_webapp/internal/repository"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Me(c *gin.Context) {
	uidVal, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	var userID int64
	switch v := uidVal.(type) {
	case int64:
		userID = v
	case float64:
		userID = int64(v)
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user id type"})
		return
	}

	repo := repository.NewUserRepository(h.DB)
	ctx := c.Request.Context()
	user, err := repo.GetByID(ctx, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         user.ID,
		"tg_id":      user.TgID,
		"username":   user.Username,
		"first_name": user.FirstName,
		"created_at": user.CreatedAt,
		"gems":       user.Gems,
		"coins":      user.Coins,
	})
}
