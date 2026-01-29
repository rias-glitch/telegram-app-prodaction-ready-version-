package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// проверяет HMAC Telegram WebApp init_data и убеждается,
// что auth_date недавний (в течение 1 часа) для предотвращения replay-атак
func ValidateTelegramInitData(initData, botToken string) (url.Values, bool) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return nil, false
	}

	hash := values.Get("hash")
	if hash == "" {
		return nil, false
	}
	values.Del("hash")

	var dataCheck []string
	for k, v := range values {
		dataCheck = append(dataCheck, k+"="+strings.Join(v, ""))
	}

	sort.Strings(dataCheck)
	dataString := strings.Join(dataCheck, "\n")

	// Telegram WebApp использует HMAC с ключом "WebAppData"
	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(botToken))
	secret := secretKey.Sum(nil)
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(dataString))

	calculated := h.Sum(nil)
	provided, err := hex.DecodeString(hash)
	if err != nil {
		return nil, false
	}

	if !hmac.Equal(calculated, provided) {
		return nil, false
	}

	// проверка актуальности: требуем auth_date в течение последнего часа
	authDateStr := values.Get("auth_date")
	if authDateStr == "" {
		return nil, false
	}
	authDate, err := strconv.ParseInt(authDateStr, 10, 64)
	if err != nil {
		return nil, false
	}

	now := time.Now().Unix()
	// разрешаем небольшую рассинхронизацию часов, но отклоняем всё старше 1 часа
	if now-authDate > 3600 || authDate-now > 300 {
		return nil, false
	}

	return values, true
}