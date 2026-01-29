package service

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// создает валидную строку init_data для тестов, используя тот же алгоритм,
// что и ValidateTelegramInitData
func buildInitData(t *testing.T, botToken string, fields map[string]string) string {
	t.Helper()
	var parts []string
	for k, v := range fields {
		parts = append(parts, k+"="+v)
	}
	sort.Strings(parts)
	dataString := strings.Join(parts, "\n")

	secret := sha256.Sum256([]byte(botToken))
	h := hmacNew(secret[:], []byte(dataString))
	hash := hex.EncodeToString(h)

	// собираем query: включаем оригинальные поля и hash
	vals := url.Values{}
	for k, v := range fields {
		vals.Add(k, v)
	}
	vals.Add("hash", hash)
	return vals.Encode()
}

// hmacNew - небольшая вспомогательная функция, дублирующая HMAC-SHA256- старая хуета от копилота auth через WebAppData
// используемый в продакшен-коде
func hmacNew(key, data []byte) []byte {
	h := sha256.New()
	// простая реализация HMAC для тестов (не заменяет прод код)
	blockSize := 64
	if len(key) > blockSize {
		tmp := sha256.Sum256(key)
		key = tmp[:]
	}
	if len(key) < blockSize {
		pad := make([]byte, blockSize-len(key))
		key = append(key, pad...)
	}
	ipad := make([]byte, blockSize)
	opad := make([]byte, blockSize)
	for i := 0; i < blockSize; i++ {
		ipad[i] = key[i] ^ 0x36
		opad[i] = key[i] ^ 0x5c
	}
	h.Reset()
	h.Write(ipad)
	h.Write(data)
	inner := h.Sum(nil)

	h2 := sha256.New()
	h2.Write(opad)
	h2.Write(inner)
	return h2.Sum(nil)
}

func TestValidateTelegramInitData_Valid(t *testing.T) {
	botToken := "test-bot-token"
	fields := map[string]string{
		"auth_date": strconv.FormatInt(time.Now().Unix(), 10),
		"user":      `{"id":1,"username":"u","first_name":"F"}`,
	}

	initData := buildInitData(t, botToken, fields)

	vals, ok := ValidateTelegramInitData(initData, botToken)
	if !ok {
		t.Fatalf("ожидалась валидная init data")
	}
	if vals.Get("user") == "" {
		t.Fatalf("ожидалось поле user в значениях")
	}
}

func TestValidateTelegramInitData_Tampered(t *testing.T) {
	botToken := "test-bot-token"
	fields := map[string]string{
		"auth_date": strconv.FormatInt(time.Now().Unix(), 10),
		"user":      `{"id":1,"username":"u","first_name":"F"}`,
	}
	initData := buildInitData(t, botToken, fields)

	// изменяем данные, добавляя дополнительное поле (нарушит хэш)
	tampered := initData + "&x=1"

	_, ok := ValidateTelegramInitData(tampered, botToken)
	if ok {
		t.Fatalf("ожидалось, что измененная init data будет невалидной")
	}
}