package domain

import "time"

type User struct {
	ID              int64     `db:"id" json:"id"`
	TgID            int64     `db:"tg_id" json:"tg_id"`
	Username        string    `db:"username" json:"username"`
	FirstName       string    `db:"first_name" json:"first_name"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	Gems            int64     `db:"gems" json:"gems"`                         // f2p
	Coins           int64     `db:"coins" json:"coins"`                       // донат
	GK              int64     `db:"gk" json:"gk"`                             // прокачка
	CharacterLevel  int       `db:"character_level" json:"character_level"`   // прокачка
	ReferralEarnings int64    `db:"referral_earnings" json:"referral_earnings"` // 50% комка,которая лутается с рефералла
}

// Валюты для игр
type Currency string

const (
	CurrencyGems  Currency = "gems"
	CurrencyCoins Currency = "coins"
)

// Курс коинов к TON
const (
	CoinsPerTON       = 10    //
	WithdrawalFeePct  = 5     // 5% комка на вывод !сейчас стоит 0,1TON на любой вывод
	MinWithdrawCoins  = 10    //  мин кол-во коинов на вывод
)
