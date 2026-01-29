package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"telegram_webapp/internal/bot"
	"telegram_webapp/internal/config"
	"telegram_webapp/internal/db"
	httpServer "telegram_webapp/internal/http"
	"telegram_webapp/internal/http/middleware"
	"telegram_webapp/internal/logger"
	"telegram_webapp/internal/service"
	"telegram_webapp/internal/ton"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Version устанавливается при сборке
var Version = "dev"

func main() {
	cfg := config.Load()

	// Инициализация структурированного логгера
	jsonLogs := os.Getenv("LOG_FORMAT") == "json"
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	logger.Init(logLevel, jsonLogs)
	log := logger.Get()

	service.InitJWT()

	dbPool := db.Connect(cfg.DatabaseURL)
	defer dbPool.Close()

	r := gin.Default()

	// CORS для прода и связи фронта с бэкендом(разные домены)
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	middleware.InitRedisRateLimiter("", "", 0)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	httpServer.RegisterRoutesWithConfig(r, dbPool, cfg.BotToken, Version, cfg)

	// Запуск админ бота ПЕРЕД HTTP сервером чтобы callback был установлен
	var adminBot *bot.AdminBot
	if cfg.AdminBotEnabled && len(cfg.AdminTelegramIDs) > 0 {
		adminService := service.NewAdminService(dbPool)

		// Инициализация TON кошелька для автоматических выводов
		walletMnemonic := os.Getenv("TON_WALLET_MNEMONIC")
		if walletMnemonic != "" {
			network := ton.NetworkMainnet
			if os.Getenv("TON_NETWORK") == "testnet" {
				network = ton.NetworkTestnet
			}
			tonWallet, err := ton.NewWallet(walletMnemonic, network)
			if err != nil {
				log.Error("failed to init TON wallet for withdrawals", "error", err)
			} else {
				adminService.SetWallet(tonWallet)
				log.Info("TON wallet initialized for auto-withdrawals", "address", tonWallet.GetAddress())
			}
		} else {
			log.Warn("TON_WALLET_MNEMONIC not set - withdrawals will be manual only")
		}

		var err error
		adminBot, err = bot.NewAdminBot(cfg.BotToken, adminService, cfg.AdminTelegramIDs)
		if err != nil {
			log.Error("failed to start admin bot", "error", err)
		} else {
			go adminBot.Start()
			log.Info("admin bot started", "admin_ids", cfg.AdminTelegramIDs)

			// Уведомление всем админам бота,если запрашивают вывод
			httpServer.SetWithdrawalNotifyCallback(adminBot.NotifyAdminsNewWithdrawal)
		}
	}

	srv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: r,
	}

	go func() {
		log.Info("server started", "port", cfg.AppPort, "version", Version)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("listen failed", "error", err)
		}
	}()

	// Запуск deposit watcher для автоматической обработки TON депозитов
	var depositWatcher *service.DepositWatcher
	platformWallet := os.Getenv("TON_PLATFORM_WALLET")
	if platformWallet != "" {
		network := ton.NetworkMainnet
		if os.Getenv("TON_NETWORK") == "testnet" {
			network = ton.NetworkTestnet
		}
		tonClient := ton.NewClient(network, os.Getenv("TON_API_KEY"))

		depositWatcher = service.NewDepositWatcher(
			dbPool,
			tonClient,
			platformWallet,
			ton.DepositCheckInterval,
		)

		// Устанавливаем callbacks для уведомлений о депозитах, если админ бот запущен
		if adminBot != nil {
			// Уведомление админов о депозите
			depositWatcher.SetDepositNotifyCallback(adminBot.NotifyAdminsNewDeposit)
			// Уведомление пользователя о депозите
			depositWatcher.SetUserNotifyCallback(func(n service.DepositNotification) {
				adminBot.NotifyUserDeposit(n.TgID, n.AmountTON, n.CoinsCredited, n.TxHash)
			})
			log.Info("deposit watcher: уведомления админов и пользователей включены")
		}

		go depositWatcher.Start()
		log.Info("deposit watcher запущен", "wallet", platformWallet, "interval", ton.DepositCheckInterval)
	} else {
		log.Warn("deposit watcher не запущен: TON_PLATFORM_WALLET не настроен")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")

	// Плавная остановка бота
	if adminBot != nil {
		adminBot.Stop()
	}

	// Плавная остановка deposit watcher
	if depositWatcher != nil {
		depositWatcher.Stop()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("server forced to shutdown", "error", err)
	}

	log.Info("server exited")
}
