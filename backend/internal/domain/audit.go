package domain

import "time"

// Логирование мастхев важных действий
type AuditLog struct {
	ID        int64                  `db:"id" json:"id"`
	UserID    int64                  `db:"user_id" json:"user_id"`
	Action    string                 `db:"action" json:"action"`
	Category  string                 `db:"category" json:"category"`
	Details   map[string]interface{} `db:"details" json:"details"`
	IP        string                 `db:"ip" json:"ip,omitempty"`
	UserAgent string                 `db:"user_agent" json:"user_agent,omitempty"`
	CreatedAt time.Time              `db:"created_at" json:"created_at"`
}

// Категории совершенных действий
const (
	AuditCategoryAuth       = "auth"
	AuditCategoryGame       = "game"
	AuditCategoryPayment    = "payment"
	AuditCategoryBalance    = "balance"
	AuditCategoryAdmin      = "admin"
	AuditCategoryWithdrawal = "withdrawal"
)

const (
	// Авторизация
	AuditActionLogin  = "login"
	AuditActionLogout = "logout"

	// Игры
	AuditActionGameStart  = "game_start"
	AuditActionGameEnd    = "game_end"
	AuditActionGameBet    = "game_bet"
	AuditActionGameWin    = "game_win"
	AuditActionGameLose   = "game_lose"

	// Платежки по  coins
	AuditActionDeposit         = "deposit"
	AuditActionWithdrawRequest = "withdraw_request"
	AuditActionWithdrawApprove = "withdraw_approve"
	AuditActionWithdrawReject  = "withdraw_reject"
	AuditActionWithdrawCancel  = "withdraw_cancel"

	// Баланс
	AuditActionBalanceCredit = "balance_credit"
	AuditActionBalanceDebit  = "balance_debit"
	AuditActionBonusClaim    = "bonus_claim"

	// Действия админов
	AuditActionAdminSetGems  = "admin_set_gems"
	AuditActionAdminAddGems  = "admin_add_gems"
	AuditActionAdminBanUser  = "admin_ban_user"
	AuditActionAdminUnbanUser = "admin_unban_user"
)
