package telegram

import "telegram_webapp/internal/service"

func ValidateInitData(initData, botToken string) bool {
	_, ok := service.ValidateTelegramInitData(initData, botToken)
	return ok
}
