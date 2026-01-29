package service

import (
	"context"

	"telegram_webapp/internal/domain"
	"telegram_webapp/internal/logger"
	"telegram_webapp/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// обрабатывает логирование аудита
type AuditService struct {
	repo *repository.AuditRepository
}

// создает новый сервис аудита
func NewAuditService(db *pgxpool.Pool) *AuditService {
	return &AuditService{
		repo: repository.NewAuditRepository(db),
	}
}

// создает новую запись в журнале аудита
func (s *AuditService) Log(ctx context.Context, userID int64, action, category string, details map[string]interface{}) {
	log := &domain.AuditLog{
		UserID:   userID,
		Action:   action,
		Category: category,
		Details:  details,
	}

	if err := s.repo.Create(ctx, log); err != nil {
		logger.Error("не удалось создать запись аудита", "error", err, "action", action, "user_id", userID)
	}
}

// создает запись аудита с информацией о запросе (ip, user-agent)
func (s *AuditService) LogWithRequest(ctx context.Context, userID int64, action, category, ip, userAgent string, details map[string]interface{}) {
	log := &domain.AuditLog{
		UserID:    userID,
		Action:    action,
		Category:  category,
		Details:   details,
		IP:        ip,
		UserAgent: userAgent,
	}

	if err := s.repo.Create(ctx, log); err != nil {
		logger.Error("не удалось создать запись аудита", "error", err, "action", action, "user_id", userID)
	}
}

// логирует игровое действие
func (s *AuditService) LogGame(ctx context.Context, userID int64, gameType string, bet, result int64, win bool, details map[string]interface{}) {
	action := domain.AuditActionGameLose
	if win {
		action = domain.AuditActionGameWin
	}

	if details == nil {
		details = make(map[string]interface{})
	}
	details["game_type"] = gameType
	details["bet"] = bet
	details["result"] = result
	details["win"] = win

	s.Log(ctx, userID, action, domain.AuditCategoryGame, details)
}

// логирует действие депозита
func (s *AuditService) LogDeposit(ctx context.Context, userID int64, amount int64, txHash string, details map[string]interface{}) {
	if details == nil {
		details = make(map[string]interface{})
	}
	details["amount"] = amount
	details["tx_hash"] = txHash

	s.Log(ctx, userID, domain.AuditActionDeposit, domain.AuditCategoryPayment, details)
}

// логирует запрос на вывод средств
func (s *AuditService) LogWithdrawRequest(ctx context.Context, userID int64, amount int64, walletAddress string) {
	details := map[string]interface{}{
		"amount":         amount,
		"wallet_address": walletAddress,
	}

	s.Log(ctx, userID, domain.AuditActionWithdrawRequest, domain.AuditCategoryWithdrawal, details)
}

// логирует подтверждение вывода средств
func (s *AuditService) LogWithdrawApprove(ctx context.Context, userID, withdrawalID int64, txHash string) {
	details := map[string]interface{}{
		"withdrawal_id": withdrawalID,
		"tx_hash":       txHash,
	}

	s.Log(ctx, userID, domain.AuditActionWithdrawApprove, domain.AuditCategoryWithdrawal, details)
}

// логирует отклонение вывода средств
func (s *AuditService) LogWithdrawReject(ctx context.Context, userID, withdrawalID int64, reason string) {
	details := map[string]interface{}{
		"withdrawal_id": withdrawalID,
		"reason":        reason,
	}

	s.Log(ctx, userID, domain.AuditActionWithdrawReject, domain.AuditCategoryWithdrawal, details)
}

// логирует действие администратора
func (s *AuditService) LogAdminAction(ctx context.Context, adminID int64, action string, targetUserID int64, details map[string]interface{}) {
	if details == nil {
		details = make(map[string]interface{})
	}
	details["admin_id"] = adminID
	details["target_user_id"] = targetUserID

	s.Log(ctx, targetUserID, action, domain.AuditCategoryAdmin, details)
}

// логирует вход пользователя
func (s *AuditService) LogLogin(ctx context.Context, userID int64, ip, userAgent string) {
	s.LogWithRequest(ctx, userID, domain.AuditActionLogin, domain.AuditCategoryAuth, ip, userAgent, nil)
}

// логирует изменение баланса
func (s *AuditService) LogBalanceChange(ctx context.Context, userID int64, change int64, reason string, details map[string]interface{}) {
	action := domain.AuditActionBalanceCredit
	if change < 0 {
		action = domain.AuditActionBalanceDebit
	}

	if details == nil {
		details = make(map[string]interface{})
	}
	details["change"] = change
	details["reason"] = reason

	s.Log(ctx, userID, action, domain.AuditCategoryBalance, details)
}

// возвращает записи аудита для пользователя
func (s *AuditService) GetUserAuditLogs(ctx context.Context, userID int64, limit int) ([]*domain.AuditLog, error) {
	return s.repo.GetByUserID(ctx, userID, limit)
}

// возвращает последние записи аудита
func (s *AuditService) GetRecentLogs(ctx context.Context, limit int) ([]*domain.AuditLog, error) {
	return s.repo.GetRecent(ctx, limit)
}

// возвращает записи аудита по категории
func (s *AuditService) GetLogsByCategory(ctx context.Context, category string, limit int) ([]*domain.AuditLog, error) {
	return s.repo.GetByCategory(ctx, category, limit)
}