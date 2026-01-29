package config

import (
	"os"
	"strconv"
	"strings"

	"telegram_webapp/internal/logger"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort          string
	DatabaseURL      string
	BotToken         string
	BotUsername      string
	WebAppShortName  string // short_name из BotFather для Web App
	JWTSecret        string
	AdminTelegramIDs []int64 // добавить в env tg id админов бота
	AdminBotEnabled  bool

	// Лимиты игр
	MaxBet         int64
	MinBet         int64
	GameRateLimit  int
	GameRateWindow int
}

// Загрузка конфига из env
func Load() *Config {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Fatal("DATABASE_URL is not set")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Fatal("JWT_SECRET is not set")
	}

	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		logger.Fatal("BOT_TOKEN is not set")
	}

	botUsername := os.Getenv("BOT_USERNAME")
	if botUsername == "" {
		botUsername = "hard_mine_playbot" // ! если не установлено в env !
	}

	webAppShortName := os.Getenv("WEBAPP_SHORT_NAME")
	if webAppShortName == "" {
		webAppShortName = "app" // short_name из BotFather
	}

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	// Проверка тг id админов !! ЧЕРЕЗ ЗАПЯТУЮ В ENV !!
	var adminIDs []int64
	adminIDsStr := os.Getenv("ADMIN_TELEGRAM_IDS")
	if adminIDsStr != "" {
		for _, idStr := range strings.Split(adminIDsStr, ",") {
			idStr = strings.TrimSpace(idStr)
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				adminIDs = append(adminIDs, id)
			}
		}
	}

	adminBotEnabled := os.Getenv("ADMIN_BOT_ENABLED") == "true"

	// Game limits (по умолчанию)
	maxBet := int64(100000) //максимум 100к
	if v := os.Getenv("MAX_BET"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			maxBet = n
		}
	}

	minBet := int64(10) //мин ставка 10
	if v := os.Getenv("MIN_BET"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			minBet = n
		}
	}

	gameRateLimit := 60 // макс действий за ->
	if v := os.Getenv("GAME_RATE_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			gameRateLimit = n
		}
	}

	gameRateWindow := 60 // -> 60 секунд
	if v := os.Getenv("GAME_RATE_WINDOW"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			gameRateWindow = n
		}
	}

	return &Config{
		AppPort:          port,
		DatabaseURL:      dbURL,
		BotToken:         botToken,
		BotUsername:      botUsername,
		WebAppShortName:  webAppShortName,
		JWTSecret:        jwtSecret,
		AdminTelegramIDs: adminIDs,
		AdminBotEnabled:  adminBotEnabled,
		MaxBet:           maxBet,
		MinBet:           minBet,
		GameRateLimit:    gameRateLimit,
		GameRateWindow:   gameRateWindow,
	}
}
