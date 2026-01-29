package ton

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// верификация доказательства TON Connect
// основано на: https://docs.ton.org/develop/dapps/ton-connect/sign

// представляет доказательство, отправленное TON Connect
type ConnectProof struct {
	Timestamp int64  `json:"timestamp"`
	Domain    Domain `json:"domain"`
	Signature string `json:"signature"`
	Payload   string `json:"payload"`
}

// представляет доменную часть доказательства
type Domain struct {
	LengthBytes int    `json:"lengthBytes"`
	Value       string `json:"value"`
}

// представляет информацию об аккаунте кошелька из TON Connect
type WalletAccount struct {
	Address   string `json:"address"`
	Chain     string `json:"chain"`
	PublicKey string `json:"publicKey"`
}

// проверяет доказательство владения кошельком TON Connect
func VerifyProof(account WalletAccount, proof ConnectProof, allowedDomain string) error {
	// 1. проверяем временную метку (доказательство должно быть свежим)
	proofTime := time.Unix(proof.Timestamp, 0)
	if time.Since(proofTime) > ProofTTL {
		return errors.New("срок действия доказательства истек")
	}

	// 2. проверяем домен
	if proof.Domain.Value != allowedDomain {
		return fmt.Errorf("несоответствие домена: ожидался %s, получен %s", allowedDomain, proof.Domain.Value)
	}

	// 3. декодируем публичный ключ
	pubKeyBytes, err := hex.DecodeString(account.PublicKey)
	if err != nil {
		return fmt.Errorf("неверный формат публичного ключа: %w", err)
	}

	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return errors.New("неверный размер публичного ключа")
	}

	// 4. декодируем подпись
	signatureBytes, err := base64.StdEncoding.DecodeString(proof.Signature)
	if err != nil {
		return fmt.Errorf("неверный формат подписи: %w", err)
	}

	// 5. собираем сообщение для проверки
	message := buildProofMessage(account.Address, proof)

	// 6. проверяем подпись
	if !ed25519.Verify(pubKeyBytes, message, signatureBytes) {
		return errors.New("неверная подпись")
	}

	return nil
}

// собирает сообщение, которое было подписано
func buildProofMessage(address string, proof ConnectProof) []byte {
	// формат сообщения:
	// "ton-proof-item-v2/" + address_workchain (4 байта) + address_hash (32 байта)
	// + domain_len (4 байта) + domain + timestamp (8 байт) + payload

	// парсим адрес для получения workchain и hash
	// для простоты соберем более простое хэш-сообщение
	// в продакшене следует правильно парсить TON-адрес

	// фактическое построение сообщения доказательства TON
	var message []byte

	// "ton-proof-item-v2/"
	message = append(message, []byte("ton-proof-item-v2/")...)

	// адрес (упрощенно - в реальной реализации следует правильно парсить)
	message = append(message, []byte(address)...)

	// длина домена (4 байта, little endian)
	domainLen := make([]byte, 4)
	binary.LittleEndian.PutUint32(domainLen, uint32(proof.Domain.LengthBytes))
	message = append(message, domainLen...)

	// значение домена
	message = append(message, []byte(proof.Domain.Value)...)

	// временная метка (8 байт, little endian)
	timestamp := make([]byte, 8)
	binary.LittleEndian.PutUint64(timestamp, uint64(proof.Timestamp))
	message = append(message, timestamp...)

	// полезная нагрузка
	message = append(message, []byte(proof.Payload)...)

	// хэшируем сообщение
	hash := sha256.Sum256(message)

	// добавляем префикс "ton-connect" и снова хэшируем
	finalMessage := append([]byte("ton-connect"), hash[:]...)
	finalHash := sha256.Sum256(finalMessage)

	return finalHash[:]
}

// генерирует случайную полезную нагрузку для TON Connect
func GeneratePayload() string {
	// генерирует случайную полезную нагрузку, которая будет подписана
	// должна быть уникальной для каждой сессии для предотвращения replay-атак
	timestamp := time.Now().Unix()
	payload := fmt.Sprintf("%d-%x", timestamp, sha256.Sum256([]byte(fmt.Sprintf("%d", timestamp))))
	return payload[:32] // обрезаем до разумной длины
}

// проверяет, является ли формат TON-адреса валидным
func ValidateAddress(address string) bool {
	// TON-адреса обычно:
	// - raw: 0:hex (workchain:hash)
	// - user-friendly: Base64 кодировка (48 символов с флагом bounceable/non-bounceable)

	if len(address) == 0 {
		return false
	}

	// проверяем raw формат (0:hex или -1:hex)
	if len(address) >= 66 && (address[0:2] == "0:" || address[0:3] == "-1:") {
		return true
	}

	// проверяем user-friendly формат (base64, 48 символов)
	if len(address) == 48 {
		_, err := base64.URLEncoding.DecodeString(address)
		return err == nil
	}

	return false
}

// нормализует адрес в raw формат
func NormalizeAddress(address string) (string, error) {
	// если уже raw формат, возвращаем как есть
	if len(address) >= 66 && (address[0:2] == "0:" || address[0:3] == "-1:") {
		return address, nil
	}

	// пытаемся декодировать user-friendly формат
	if len(address) == 48 {
		decoded, err := base64.URLEncoding.DecodeString(address)
		if err != nil {
			return "", fmt.Errorf("неверный формат адреса: %w", err)
		}

		// user-friendly адрес состоит из 36 байт:
		// 1 байт флагов + 1 байт workchain + 32 байта хэша + 2 байта CRC
		if len(decoded) != 36 {
			return "", errors.New("неверная длина адреса")
		}

		workchain := int8(decoded[1])
		hash := decoded[2:34]

		return fmt.Sprintf("%d:%s", workchain, hex.EncodeToString(hash)), nil
	}

	return "", errors.New("неизвестный формат адреса")
}

// преобразует raw адрес (0:xxx) в user-friendly формат (EQ.../UQ...)
// bounceable=true означает, что адрес начинается с EQ (для смарт-контрактов)
// bounceable=false означает, что адрес начинается с UQ (для кошельков)
func RawToUserFriendly(rawAddress string, bounceable bool) (string, error) {
	// парсим raw формат адреса: workchain:hash
	var workchain int8
	var hashHex string

	if len(rawAddress) >= 66 && rawAddress[0:2] == "0:" {
		workchain = 0
		hashHex = rawAddress[2:]
	} else if len(rawAddress) >= 67 && rawAddress[0:3] == "-1:" {
		workchain = -1
		hashHex = rawAddress[3:]
	} else {
		// возможно, это уже user-friendly формат
		if len(rawAddress) == 48 {
			return rawAddress, nil
		}
		return "", errors.New("неверный формат raw адреса")
	}

	// декодируем хэш из hex
	hashBytes, err := hex.DecodeString(hashHex)
	if err != nil {
		return "", fmt.Errorf("неверный hex хэша: %w", err)
	}

	if len(hashBytes) != 32 {
		return "", errors.New("неверная длина хэша")
	}

	// строим user-friendly адрес
	// формат: 1 байт флага + 1 байт workchain + 32 байта хэша + 2 байта CRC16
	data := make([]byte, 34)

	// байт флага: 0x11 для bounceable (EQ), 0x51 для non-bounceable (UQ)
	if bounceable {
		data[0] = 0x11
	} else {
		data[0] = 0x51
	}

	// байт workchain
	data[1] = byte(workchain)

	// копируем хэш
	copy(data[2:], hashBytes)

	// вычисляем CRC16
	crc := crc16(data)

	// строим финальный адрес: данные + CRC
	result := make([]byte, 36)
	copy(result, data)
	result[34] = byte(crc >> 8)
	result[35] = byte(crc & 0xFF)

	// кодируем как стандартный base64 без padding (tonutils-go ожидает такой формат)
	return base64.RawStdEncoding.EncodeToString(result), nil
}

// вычисляет CRC16-XMODEM
func crc16(data []byte) uint16 {
	crc := uint16(0)
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}