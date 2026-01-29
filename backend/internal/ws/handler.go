package ws

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

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// содержит зависимости для обработки WebSocket
type WSHandler struct {
	Hub      *Hub
	UserRepo *repository.UserRepository
}

func NewWSHandler(hub *Hub, userRepo *repository.UserRepository) *WSHandler {
	return &WSHandler{
		Hub:      hub,
		UserRepo: userRepo,
	}
}

func (h *WSHandler) HandleWS() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "токен обязателен"})
			return
		}

		userID, err := service.ParseJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "неверный токен"})
			return
		}

		// получаем тип игры из query (по умолчанию: rps)
		gameType := c.Query("game")
		if gameType == "" {
			gameType = "rps"
		}

		// получаем сумму ставки из query (по умолчанию: 0 для бесплатной игры)
		betAmount := int64(0)
		if betStr := c.Query("bet"); betStr != "" {
			if bet, err := strconv.ParseInt(betStr, 10, 64); err == nil && bet > 0 {
				betAmount = bet
			}
		}

		// получаем валюту из query (по умолчанию: gems)
		currency := c.Query("currency")
		if currency != string(domain.CurrencyCoins) {
			currency = string(domain.CurrencyGems)
		}

		// проверяем, что у пользователя достаточно средств для ставки
		if betAmount > 0 && h.UserRepo != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			user, err := h.UserRepo.GetByID(ctx, userID)
			if err != nil || user == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "пользователь не найден"})
				return
			}

			var balance int64
			if currency == string(domain.CurrencyCoins) {
				balance = user.Coins
			} else {
				balance = user.Gems
			}

			if balance < betAmount {
				c.JSON(http.StatusBadRequest, gin.H{"error": "недостаточно средств"})
				return
			}

			// списываем ставку с баланса пользователя (резервируем её)
			if currency == string(domain.CurrencyCoins) {
				if _, err := h.UserRepo.UpdateCoins(ctx, userID, -betAmount); err != nil {
					log.Printf("HandleWS: не удалось списать монеты: %v", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось зарезервировать ставку"})
					return
				}
			} else {
				if _, err := h.UserRepo.UpdateGems(ctx, userID, -betAmount); err != nil {
					log.Printf("HandleWS: не удалось списать драгоценные камни: %v", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось зарезервировать ставку"})
					return
				}
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

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Println("ошибка обновления ws:", err)
			// возвращаем ставку, если обновление WebSocket не удалось
			if betAmount > 0 && h.UserRepo != nil {
				ctx := context.Background()
				if currency == string(domain.CurrencyCoins) {
					h.UserRepo.UpdateCoins(ctx, userID, betAmount)
				} else {
					h.UserRepo.UpdateGems(ctx, userID, betAmount)
				}
			}
			return
		}

		// создаем клиента и запускаем его обработчики и матчмейкинг
		client := NewClient(userID, conn, h.Hub, gameType, betAmount, currency)
		go client.Run()
	}
}

// устаревший обработчик для обратной совместимости (без ставок)
func HandleWS(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "токен обязателен"})
			return
		}

		userID, err := service.ParseJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "неверный токен"})
			return
		}

		// получаем тип игры из query (по умолчанию: rps)
		gameType := c.Query("game")
		if gameType == "" {
			gameType = "rps"
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

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Println("ошибка обновления ws:", err)
			return
		}

		// создаем клиента и запускаем его обработчики и матчмейкинг (бесплатная игра)
		client := NewClient(userID, conn, hub, gameType, 0, string(domain.CurrencyGems))
		go client.Run()
	}
}