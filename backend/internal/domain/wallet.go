package domain

import "time"

// Подключенный TON  кошелек
type Wallet struct {
	ID                 int64     `db:"id" json:"id"`
	UserID             int64     `db:"user_id" json:"user_id"`
	Address            string    `db:"address" json:"address"`
	RawAddress         string    `db:"raw_address" json:"raw_address,omitempty"`
	LinkedAt           time.Time `db:"linked_at" json:"linked_at"`
	IsVerified         bool      `db:"is_verified" json:"is_verified"`
	LastProofTimestamp int64     `db:"last_proof_timestamp" json:"last_proof_timestamp,omitempty"`
}

// Входящий перевод - пополнение
type Deposit struct {
	ID            int64         `db:"id" json:"id"`
	UserID        int64         `db:"user_id" json:"user_id"`
	WalletAddress string        `db:"wallet_address" json:"wallet_address"`
	AmountNano    int64         `db:"amount_nano" json:"amount_nano"`
	GemsCredited  int64         `db:"gems_credited" json:"gems_credited"` //заменена на коины,но можно чето придумать новое,оставлено ЧТОБЫ НЕ ЛОМАТЬ
	CoinsCredited int64         `db:"coins_credited" json:"coins_credited"` // текущие коины,И ТОЛЬКО КОИНЫ МОЖНО ВЫВЕСТИ
	ExchangeRate  int           `db:"exchange_rate" json:"exchange_rate"`
	TxHash        string        `db:"tx_hash" json:"tx_hash"`
	TxLt          int64         `db:"tx_lt" json:"tx_lt,omitempty"`
	Status        DepositStatus `db:"status" json:"status"`
	Memo          string        `db:"memo" json:"memo,omitempty"`
	CreatedAt     time.Time     `db:"created_at" json:"created_at"`
	ConfirmedAt   *time.Time    `db:"confirmed_at" json:"confirmed_at,omitempty"`
	Processed     bool          `db:"processed" json:"processed"`
}

// Статус обработки пополнения
type DepositStatus string

const (
	DepositStatusPending   DepositStatus = "pending"
	DepositStatusConfirmed DepositStatus = "confirmed"
	DepositStatusFailed    DepositStatus = "failed"
	DepositStatusExpired   DepositStatus = "expired"
)

// Исходящий вывод,размен коинов на  TON и отправка - вывод
type Withdrawal struct {
	ID            int64            `db:"id" json:"id"`
	UserID        int64            `db:"user_id" json:"user_id"`
	WalletAddress string           `db:"wallet_address" json:"wallet_address"`
	CoinsAmount   int64            `db:"coins_amount" json:"coins_amount"` // сумма в коинах
	TonAmountNano int64            `db:"ton_amount_nano" json:"ton_amount_nano"`
	FeeCoins      int64            `db:"fee_coins" json:"fee_coins"` // фиксированная комиссия
	ExchangeRate  int              `db:"exchange_rate" json:"exchange_rate"` // коины за тон
	Status        WithdrawalStatus `db:"status" json:"status"`
	TxHash        string           `db:"tx_hash" json:"tx_hash,omitempty"`
	TxLt          int64            `db:"tx_lt" json:"tx_lt,omitempty"`
	AdminNotes    string           `db:"admin_notes" json:"admin_notes,omitempty"`
	CreatedAt     time.Time        `db:"created_at" json:"created_at"`
	ProcessedAt   *time.Time       `db:"processed_at" json:"processed_at,omitempty"`
	CompletedAt   *time.Time       `db:"completed_at" json:"completed_at,omitempty"`
	// !!НЕ ИСПОЛЬЗУЕТСЯ, оставлено для работоспособности и дальнейшей модификации
	GemsAmount int64 `db:"gems_amount" json:"gems_amount,omitempty"`
	FeeGems    int64 `db:"fee_gems" json:"fee_gems,omitempty"`
}

// Статус вывода
type WithdrawalStatus string

const (
	WithdrawalStatusPending    WithdrawalStatus = "pending"
	WithdrawalStatusProcessing WithdrawalStatus = "processing"
	WithdrawalStatusSent       WithdrawalStatus = "sent"
	WithdrawalStatusCompleted  WithdrawalStatus = "completed"
	WithdrawalStatusFailed     WithdrawalStatus = "failed"
	WithdrawalStatusCancelled  WithdrawalStatus = "cancelled"
)

// Информация возвращается пользователю,когда он хочет вкинуть донат
type DepositInfo struct {
	PlatformAddress string `json:"platform_address"`
	Memo            string `json:"memo"` //НЕ ЗАБЫВАТЬ УКАЗЫВАТЬ memo ибо не обработается
	MinAmountTON    string `json:"min_amount_ton"`
	ExchangeRate    int    `json:"exchange_rate"` // 10 коинов = 1 TON
}

// Платежка ОТ пользователя
type WithdrawRequest struct {
	CoinsAmount int64 `json:"coins_amount" binding:"required,min=10"` // минималка 10 коинов
}

// Отображение того,что получит пользователь
type WithdrawEstimate struct {
	CoinsAmount   int64   `json:"coins_amount"`
	FeeCoins      int64   `json:"fee_coins"`       //комка снимается с коинов
	NetCoins      int64   `json:"net_coins"`
	TonAmount     string  `json:"ton_amount"`
	TonAmountNano int64   `json:"ton_amount_nano"`
	ExchangeRate  int     `json:"exchange_rate"`   // 10 коинов за TON
	FeePercent    float64 `json:"fee_percent"`     // !!!!!!ЕСЛИ ИЗМЕНИТЬ КОМКУ НА ПРОЦЕНТНУЮ(сейчас не используется, стоит фиксированная)
	FeeTON        float64 `json:"fee_ton"`
}
