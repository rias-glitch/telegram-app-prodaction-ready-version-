package handlers

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"telegram_webapp/internal/domain"
	"telegram_webapp/internal/repository"
	"telegram_webapp/internal/service"
	"telegram_webapp/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func (h *Handler) WS(hub *ws.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Проверка JWT токена
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token required"})
			return
		}

		userID, err := service.ParseJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// получаем тип игры(по умолчанию rps)
		gameType := c.Query("game")
		if gameType == "" {
			gameType = "rps"
		}

		// получить сумму ставки и валюты из запроса
		var betAmount int64
		if betStr := c.Query("bet"); betStr != "" {
			if parsed, err := strconv.ParseInt(betStr, 10, 64); err == nil {
				betAmount = parsed
			}
		}
		currency := c.Query("currency")
		if currency == "" {
			currency = "gems" // валюта по умолчанию
		}

		// списываем ставку с баланса пользователя при входе в игру
		if betAmount > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			userRepo := repository.NewUserRepository(h.DB)
			user, err := userRepo.GetByID(ctx, userID)
			if err != nil || user == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "user not found"})
				return
			}

			// проверяем баланс
			var balance int64
			if currency == string(domain.CurrencyCoins) {
				balance = user.Coins
			} else {
				balance = user.Gems
			}

			if balance < betAmount {
				c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient balance"})
				return
			}

			// списываем ставку
			if currency == string(domain.CurrencyCoins) {
				if _, err := userRepo.UpdateCoins(ctx, userID, -betAmount); err != nil {
					log.Printf("WS: failed to deduct coins: %v", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reserve bet"})
					return
				}
				log.Printf("WS: deducted %d coins from user=%d", betAmount, userID)
			} else {
				if _, err := userRepo.UpdateGems(ctx, userID, -betAmount); err != nil {
					log.Printf("WS: failed to deduct gems: %v", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reserve bet"})
					return
				}
				log.Printf("WS: deducted %d gems from user=%d", betAmount, userID)
			}
		}

		allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				if allowedOrigin == "" {
					return true
				}
				return r.Header.Get("Origin") == allowedOrigin
			},
		}

		// обновление вебсокета
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Println("ws upgrade error:", err)
			// возвращаем ставку если upgrade не удался
			if betAmount > 0 {
				ctx := context.Background()
				userRepo := repository.NewUserRepository(h.DB)
				if currency == string(domain.CurrencyCoins) {
					userRepo.UpdateCoins(ctx, userID, betAmount)
				} else {
					userRepo.UpdateGems(ctx, userID, betAmount)
				}
				log.Printf("WS: refunded %d %s to user=%d after upgrade failure", betAmount, currency, userID)
			}
			return
		}

		// создание клиента с типом игры,суммой ставки и валюты
		client := ws.NewClient(userID, conn, hub, gameType, betAmount, currency)

		go client.Run()
	}
}
