package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"telegram_webapp/internal/domain"
	"telegram_webapp/internal/repository"
	"telegram_webapp/internal/service"

	"github.com/gin-gonic/gin"
)


type AuthRequest struct {
	InitData string `json:"init_data"`
}

func (h *Handler) Auth(c *gin.Context) {
	var req AuthRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	// DEV MODE: пропускаем валидацию
	if os.Getenv("DEV_MODE") == "true" {
		// Парсим ID из init_data (или используем дефолт)
		var userId int64 = 12345

		// Пытаемся распарсить
		if strings.Contains(req.InitData, "\"id\":") {
			start := strings.Index(req.InitData, "\"id\":") + 5
			end := start
			for end < len(req.InitData) && req.InitData[end] >= '0' && req.InitData[end] <= '9' {
				end++
			}
			if parsed, err := strconv.ParseInt(req.InitData[start:end], 10, 64); err == nil {
				userId = parsed
			}
		}

		repo := repository.NewUserRepository(h.DB)
		ctx := c.Request.Context()

		user, err := repo.GetByTgID(ctx, userId)
		if err != nil {
			user = &domain.User{
				TgID:      userId,
				Username:  fmt.Sprintf("testuser%d", userId),
				FirstName: "Test",
			}
			if err := repo.Create(ctx, user); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
				return
			}
		}

		token, err := service.GenerateJWT(user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
			"user": gin.H{
				"id":         user.ID,
				"tg_id":      user.TgID,
				"username":   user.Username,
				"first_name": user.FirstName,
				"gems":       user.Gems,
			},
		})
		return
	}

	// дефолтная валидация для прода
	if len(req.InitData) > 4096 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "init_data too long"})
		return
	}

	values, ok := service.ValidateTelegramInitData(req.InitData, h.BotToken)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or stale telegram data"})
		return
	}

	userRaw := values.Get("user")
	if userRaw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user not found"})
		return
	}

	userValues, _ := url.ParseQuery("user=" + userRaw)
	userJSON := userValues.Get("user")

	var tgUser struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		FirstName string `json:"first_name"`
	}

	if err := json.Unmarshal([]byte(userJSON), &tgUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user json"})
		return
	}

	repo := repository.NewUserRepository(h.DB)
	ctx := c.Request.Context()

	isNewUser := false
	user, err := repo.GetByTgID(ctx, tgUser.ID)
	if err != nil {
		isNewUser = true
		user = &domain.User{
			TgID:      tgUser.ID,
			Username:  tgUser.Username,
			FirstName: tgUser.FirstName,
		}

		if err := repo.Create(ctx, user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}
	}

	// Обработка реферральной ссылки из (параметра startapp)
	startParam := values.Get("start_param")
	if startParam != "" && strings.HasPrefix(startParam, "ref_") && isNewUser {
		refCode := strings.TrimPrefix(startParam, "ref_")
		if refCode != "" {
			// применить для нового пользователя
			referralRepo := repository.NewReferralRepository(h.DB)
			referrerID, err := referralRepo.GetUserByReferralCode(ctx, refCode)
			if err == nil && referrerID != user.ID {
				// засчитать рефку
				_ = referralRepo.CreateReferral(ctx, referrerID, user.ID)
			}
		}
	}

	token, err := service.GenerateJWT(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":         user.ID,
			"tg_id":      user.TgID,
			"username":   user.Username,
			"first_name": user.FirstName,
			"gems":       user.Gems,
		},
	})
}
