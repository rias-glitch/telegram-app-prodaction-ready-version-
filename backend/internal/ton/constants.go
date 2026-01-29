package ton

import "time"

const (
	// коэффициент обмена: сколько монет за 1 TON
	// 1 TON = 10 монет (премиум валюта)
	CoinsPerTON = 10

	// сохраняется для обратной совместимости (бесплатная валюта, не выводится)
	// 1 TON = 10000 гемов (только для справки, драгоценные камни нельзя купить за TON)
	GemsPerTON = 10000

	// наименьшая единица TON (1 TON = 10^9 наноTON)
	NanoTON = 1_000_000_000

	// минимальная сумма депозита в наноTON (1 TON = 10 монет)
	MinDepositNano = 1_000_000_000

	// минимальная сумма вывода в монетах (10 монет = 1 TON)
	MinWithdrawCoins = 10

	// фиксированная комиссия платформы на вывод (1 монета = 0.1 TON)
	WithdrawFeeCoinsFixed = 1

	// сохраняется для обратной совместимости, но не используется (заменена фиксированной комиссией)
	WithdrawFeePercent = 5

	// максимальный вывод в день в монетах (1000 монет = 100 TON)
	MaxWithdrawCoinsPerDay = 1000

	// количество необходимых подтверждений для депозита
	DepositConfirmations = 1

	// время жизни доказательства TON Connect
	ProofTTL = 15 * time.Minute

	// интервал проверки новых депозитов
	DepositCheckInterval = 30 * time.Second

	// интервал обработки ожидающих выводов
	WithdrawProcessInterval = 1 * time.Minute
)

// представляет тип сети TON
type Network string

const (
	NetworkMainnet Network = "mainnet"
	NetworkTestnet Network = "testnet"
)

// конечные точки TON API
const (
	TonAPIMainnet = "https://tonapi.io/v2"
	TonAPITestnet = "https://testnet.tonapi.io/v2"

	TonCenterMainnet = "https://toncenter.com/api/v2"
	TonCenterTestnet = "https://testnet.toncenter.com/api/v2"
)

// конвертирует TON в наноTON
func TONToNano(ton float64) int64 {
	return int64(ton * NanoTON)
}

// конвертирует наноTON в TON
func NanoToTON(nano int64) float64 {
	return float64(nano) / NanoTON
}

// конвертирует наноTON в драгоценные камни по курсу обмена (устаревшее)
func NanoToGems(nano int64) int64 {
	ton := NanoToTON(nano)
	return int64(ton * GemsPerTON)
}

// конвертирует драгоценные камни в наноTON по курсу обмена (устаревшее)
func GemsToNano(gems int64) int64 {
	ton := float64(gems) / GemsPerTON
	return TONToNano(ton)
}

// конвертирует наноTON в монеты (1 TON = 10 монет)
func NanoToCoins(nano int64) int64 {
	ton := NanoToTON(nano)
	return int64(ton * CoinsPerTON)
}

// конвертирует монеты в наноTON (10 монет = 1 TON)
func CoinsToNano(coins int64) int64 {
	ton := float64(coins) / CoinsPerTON
	return TONToNano(ton)
}

// конвертирует монеты в TON
func CoinsToTON(coins int64) float64 {
	return float64(coins) / CoinsPerTON
}

// конвертирует TON в монеты
func TONToCoins(ton float64) int64 {
	return int64(ton * CoinsPerTON)
}

// рассчитывает комиссию на вывод в монетах (фиксированная 0.1 TON = 1 монета)
func CalculateWithdrawFeeCoins(coinsAmount int64) int64 {
	return WithdrawFeeCoinsFixed
}

// рассчитывает чистые монеты после комиссии
func CalculateWithdrawNetCoins(coinsAmount int64) int64 {
	return coinsAmount - WithdrawFeeCoinsFixed
}

// устаревшие функции для обратной совместимости
func CalculateWithdrawFee(gemsAmount int64) int64 {
	return gemsAmount * WithdrawFeePercent / 100
}

func CalculateWithdrawNet(gemsAmount int64) int64 {
	return gemsAmount - CalculateWithdrawFee(gemsAmount)
}