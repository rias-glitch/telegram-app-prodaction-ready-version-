package handlers

import (
	"net/http"

	"telegram_webapp/internal/repository"

	"github.com/gin-gonic/gin"
)

// Обработка запросов с рефками
type ReferralHandler struct {
	repo            *repository.ReferralRepository
	botUsername     string
	webAppShortName string
}

// создает новый handler для реф ссылок
func NewReferralHandler(repo *repository.ReferralRepository, botUsername, webAppShortName string) *ReferralHandler {
	return &ReferralHandler{repo: repo, botUsername: botUsername, webAppShortName: webAppShortName}
}

// возвращает реферральный код пользователя
func (h *ReferralHandler) GetReferralCode(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	code, err := h.repo.GetOrCreateReferralCode(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get referral code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": code})
}

// статистика реферралов пользователя
func (h *ReferralHandler) GetReferralStats(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	stats, err := h.repo.GetReferralStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get stats"})
		return
	}

	referrals, err := h.repo.GetReferralsByUser(c.Request.Context(), userID)
	if err != nil {
		referrals = []repository.Referral{}
	}

	c.JSON(http.StatusOK, gin.H{
		"stats":     stats,
		"referrals": referrals,
	})
}

// Применяет реферральный код для текущего юзера
type ApplyReferralRequest struct {
	Code string `json:"code" binding:"required"`
}

func (h *ReferralHandler) ApplyReferralCode(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req ApplyReferralRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	// был ли юзер уже приглашен?
	isReferred, err := h.repo.IsReferred(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check referral status"})
		return
	}

	if isReferred {
		c.JSON(http.StatusBadRequest, gin.H{"error": "already referred"})
		return
	}

	// найти приглашенного
	referrerID, err := h.repo.GetUserByReferralCode(c.Request.Context(), req.Code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid referral code"})
		return
	}

	// не засчитывать создателя рефки
	if referrerID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot use your own code"})
		return
	}

	// Создать реферрала
	err = h.repo.CreateReferral(c.Request.Context(), referrerID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to apply referral"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "referral applied successfully"})
}

// полная рефка для -> поделиться
func (h *ReferralHandler) GetReferralLink(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	code, err := h.repo.GetOrCreateReferralCode(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get referral code"})
		return
	}

	// https://t.me/bot_username/webapp_short_name?startapp=ref_CODE
	// Открывает сразу веб аппку напрямую
	link := "https://t.me/" + h.botUsername + "/" + h.webAppShortName + "?startapp=ref_" + code

	c.JSON(http.StatusOK, gin.H{
		"code": code,
		"link": link,
	})
}
