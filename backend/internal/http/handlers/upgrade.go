package handlers

import (
	"net/http"

	"telegram_webapp/internal/repository"

	"github.com/gin-gonic/gin"
)

// обрабатывает запросы на повышение лвл персонажа
type UpgradeHandler struct {
	userRepo     *repository.UserRepository
	referralRepo *repository.ReferralRepository
}

// создает новый handler для 'прокачки'
func NewUpgradeHandler(userRepo *repository.UserRepository, referralRepo *repository.ReferralRepository) *UpgradeHandler {
	return &UpgradeHandler{
		userRepo:     userRepo,
		referralRepo: referralRepo,
	}
}

// стоимость перехода на след уровень
var UpgradeCosts = map[int]int64{
	2:  100,   // сколько нужно gk для левел апа
	3:  250,
	4:  500,
	5:  1000,
	6:  2000,
	7:  4000,
	8:  7500,
	9:  12000,
	10: 20000,
}

// вознаграждение в gk за реферралов
var ReferralGKRewards = map[int]int64{
	1:  50,
	2:  100,
	3:  200,
	4:  350,
	5:  500,
	10: 1500,
	25: 5000,
	50: 12000,
	100: 30000,
}

// возвращает инфо о системе прокачки
func (h *UpgradeHandler) GetUpgradeInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"costs":            UpgradeCosts,
		"referral_rewards": ReferralGKRewards,
		"max_level":        10,
	})
}

// возвращает текущий статус прокачки пользователя
func (h *UpgradeHandler) GetMyUpgradeStatus(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	//получение данных пользователя
	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	// получить статистику рефералов
	stats, err := h.referralRepo.GetReferralStats(c.Request.Context(), userID)
	if err != nil {
		stats = &repository.ReferralStats{}
	}

	// стоимость для апгрейда на след лвл
	var nextLevelCost int64
	if user.CharacterLevel < 10 {
		nextLevelCost = UpgradeCosts[user.CharacterLevel+1]
	}

	// расчет наград GK
	claimedRewards, err := h.referralRepo.GetClaimedGKRewards(c.Request.Context(), userID)
	if err != nil {
		claimedRewards = []int{}
	}

	// доступные награды для получения
	availableRewards := []struct {
		Threshold int   `json:"threshold"`
		Reward    int64 `json:"reward"`
	}{}

	for threshold, reward := range ReferralGKRewards {
		if stats.TotalReferrals >= threshold {
			claimed := false
			for _, c := range claimedRewards {
				if c == threshold {
					claimed = true
					break
				}
			}
			if !claimed {
				availableRewards = append(availableRewards, struct {
					Threshold int   `json:"threshold"`
					Reward    int64 `json:"reward"`
				}{Threshold: threshold, Reward: reward})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"gk":                user.GK,
		"character_level":   user.CharacterLevel,
		"next_level_cost":   nextLevelCost,
		"total_referrals":   stats.TotalReferrals,
		"referral_earnings": user.ReferralEarnings,
		"available_rewards": availableRewards,
		"costs":             UpgradeCosts,
		"referral_rewards":  ReferralGKRewards,
	})
}

type UpgradeRequest struct {
	TargetLevel int `json:"target_level" binding:"required,min=2,max=10"`
}

// повышает уровень
func (h *UpgradeHandler) UpgradeCharacter(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req UpgradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// получить текущий лвл
	currentLevel, err := h.userRepo.GetCharacterLevel(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get level"})
		return
	}

	// проверка на валид обновления
	if req.TargetLevel != currentLevel+1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "can only upgrade to next level"})
		return
	}

	// получить стоимость
	cost, ok := UpgradeCosts[req.TargetLevel]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target level"})
		return
	}

	// обновить
	err = h.userRepo.UpgradeCharacter(c.Request.Context(), userID, req.TargetLevel, cost)
	if err != nil {
		if err == repository.ErrInsufficientFunds {
			c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient GK"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upgrade"})
		return
	}

	// получить новый баланс
	newGK, _ := h.userRepo.GetGK(c.Request.Context(), userID)

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"new_level":       req.TargetLevel,
		"gk":              newGK,
		"next_level_cost": UpgradeCosts[req.TargetLevel+1],
	})
}

type ClaimRewardRequest struct {
	Threshold int `json:"threshold" binding:"required"`
}

// получает GK при достижении первого порога
func (h *UpgradeHandler) ClaimReferralReward(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req ClaimRewardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// проверить есть ли вознаграждение у порога
	reward, ok := ReferralGKRewards[req.Threshold]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid threshold"})
		return
	}

	// получить статистику пригласившего
	stats, err := h.referralRepo.GetReferralStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get stats"})
		return
	}

	// достаточно ли реферралов у пользователя
	if stats.TotalReferrals < req.Threshold {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not enough referrals"})
		return
	}

	// забрана ли уже награда
	claimed, err := h.referralRepo.IsGKRewardClaimed(c.Request.Context(), userID, req.Threshold)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check claim status"})
		return
	}
	if claimed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reward already claimed"})
		return
	}

	// забрать награду
	err = h.referralRepo.ClaimGKReward(c.Request.Context(), userID, req.Threshold, reward)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to claim reward"})
		return
	}

	// получить новый баланс GK
	newGK, _ := h.userRepo.GetGK(c.Request.Context(), userID)

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"reward":    reward,
		"gk":        newGK,
		"threshold": req.Threshold,
	})
}
