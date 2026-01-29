package handlers

import (
	"telegram_webapp/internal/repository"
	"telegram_webapp/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HandlerConfig содержит конфигурацию для обработчика
type HandlerConfig struct {
	MinBet int64
	MaxBet int64
}

type Handler struct {
	DB                 *pgxpool.Pool
	BotToken           string
	GameHistoryRepo    *repository.GameHistoryRepository
	QuestRepo          *repository.QuestRepository
	TransactionRepo    *repository.TransactionRepository
	UserRepo           *repository.UserRepository
	MinesProService    *service.MinesProService
	CoinFlipProService *service.CoinFlipProService
	GameService        *service.GameService
	AuditService       *service.AuditService
}
// Прием зависимостей на вход
func NewHandler(db *pgxpool.Pool, botToken string) *Handler {
	return &Handler{
		DB:                 db,
		BotToken:           botToken,
		GameHistoryRepo:    repository.NewGameHistoryRepository(db),
		QuestRepo:          repository.NewQuestRepository(db),
		TransactionRepo:    repository.NewTransactionRepository(db),
		UserRepo:           repository.NewUserRepository(db),
		MinesProService:    service.NewMinesProService(db),
		CoinFlipProService: service.NewCoinFlipProService(db),
		GameService:        service.NewGameService(db),
		AuditService:       service.NewAuditService(db),
	}
}

// Создает handler с пользовательским конфигом
func NewHandlerWithConfig(db *pgxpool.Pool, botToken string, cfg HandlerConfig) *Handler {
	return &Handler{
		DB:                 db,
		BotToken:           botToken,
		GameHistoryRepo:    repository.NewGameHistoryRepository(db),
		QuestRepo:          repository.NewQuestRepository(db),
		TransactionRepo:    repository.NewTransactionRepository(db),
		UserRepo:           repository.NewUserRepository(db),
		MinesProService:    service.NewMinesProService(db),
		CoinFlipProService: service.NewCoinFlipProService(db),
		GameService:        service.NewGameServiceWithLimits(db, cfg.MinBet, cfg.MaxBet),
		AuditService:       service.NewAuditService(db),
	}
}

// getUserID извлекает user_id из контекста Gin
func getUserID(c interface{ Get(string) (any, bool) }) (int64, bool) {
	uidVal, ok := c.Get("user_id")
	if !ok {
		return 0, false
	}
	switch v := uidVal.(type) {
	case int64:
		return v, true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}
