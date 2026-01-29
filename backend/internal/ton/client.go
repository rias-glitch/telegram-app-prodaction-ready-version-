package ton

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// клиент TON API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	network    Network
}

// создает новый клиент TON API
func NewClient(network Network, apiKey string) *Client {
	baseURL := TonAPIMainnet
	if network == NetworkTestnet {
		baseURL = TonAPITestnet
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		network: network,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AccountAddress представляет адрес аккаунта в ответе API
type AccountAddress struct {
	Address  string `json:"address"`
	IsScam   bool   `json:"is_scam"`
	IsWallet bool   `json:"is_wallet"`
}

// Transaction представляет транзакцию в сети TON (tonapi.io v2 format)
type Transaction struct {
	Hash       string          `json:"hash"`
	Lt         int64           `json:"lt"`
	Account    *AccountAddress `json:"account"`
	Utime      int64           `json:"utime"`
	OrigStatus string          `json:"orig_status"`
	EndStatus  string          `json:"end_status"`
	TotalFees  int64           `json:"total_fees"`
	InMsg      *Message        `json:"in_msg"`
	OutMsgs    []Message       `json:"out_msgs"`
	Success    bool            `json:"success"`
}

// Message представляет сообщение в сети TON
type Message struct {
	MsgType       string          `json:"msg_type"`
	CreatedLt     int64           `json:"created_lt"`
	IhrDisabled   bool            `json:"ihr_disabled"`
	Bounce        bool            `json:"bounce"`
	Bounced       bool            `json:"bounced"`
	Value         int64           `json:"value"`
	FwdFee        int64           `json:"fwd_fee"`
	IhrFee        int64           `json:"ihr_fee"`
	Destination   *AccountAddress `json:"destination"`
	Source        *AccountAddress `json:"source"`
	ImportFee     int64           `json:"import_fee"`
	CreatedAt     int64           `json:"created_at"`
	OpCode        string          `json:"op_code"`
	Hash          string          `json:"hash"`
	RawBody       string          `json:"raw_body"`
	DecodedOpName string          `json:"decoded_op_name"`
	DecodedBody   *DecodedBody    `json:"decoded_body"`
}

// DecodedBody представляет декодированное тело сообщения
type DecodedBody struct {
	Text    string `json:"text"`
	Payload string `json:"payload"`
}

// AccountInfo представляет информацию об аккаунте
type AccountInfo struct {
	Address    string `json:"address"`
	Balance    int64  `json:"balance"`
	Status     string `json:"status"`
	LastTxLt   int64  `json:"last_transaction_lt"`
	LastTxHash string `json:"last_transaction_hash"`
}

// получает информацию об аккаунте
func (c *Client) GetAccountInfo(ctx context.Context, address string) (*AccountInfo, error) {
	reqURL := fmt.Sprintf("%s/accounts/%s", c.baseURL, address)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ошибка API: %s - %s", resp.Status, string(body))
	}

	var account AccountInfo
	if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
		return nil, err
	}

	return &account, nil
}

// получает последние транзакции для адреса
func (c *Client) GetTransactions(ctx context.Context, address string, limit int, beforeLt int64) ([]Transaction, error) {
	// Используем blockchain endpoint для получения транзакций
	reqURL := fmt.Sprintf("%s/blockchain/accounts/%s/transactions?limit=%d", c.baseURL, address, limit)
	if beforeLt > 0 {
		reqURL = fmt.Sprintf("%s&before_lt=%d", reqURL, beforeLt)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ошибка API: %s - %s", resp.Status, string(body))
	}

	var result struct {
		Transactions []Transaction `json:"transactions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Transactions, nil
}

// получает конкретную транзакцию по хэшу
func (c *Client) GetTransaction(ctx context.Context, hash string) (*Transaction, error) {
	reqURL := fmt.Sprintf("%s/blockchain/transactions/%s", c.baseURL, hash)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ошибка API: %s - %s", resp.Status, string(body))
	}

	var tx Transaction
	if err := json.NewDecoder(resp.Body).Decode(&tx); err != nil {
		return nil, err
	}

	return &tx, nil
}

// ожидает появления транзакции в блокчейне
func (c *Client) WaitForTransaction(ctx context.Context, hash string, timeout time.Duration) (*Transaction, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		tx, err := c.GetTransaction(ctx, hash)
		if err != nil {
			return nil, err
		}
		if tx != nil {
			return tx, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	return nil, fmt.Errorf("транзакция не найдена в течение таймаута")
}

// setAuthHeader устанавливает заголовок авторизации если ключ валидный
func (c *Client) setAuthHeader(req *http.Request) {

	// Пропускаем невалидные ключи (валидные ключи tonapi начинаются с AF/AG или это длинные JWT)
	if c.apiKey == "" {
		return
	}
	if strings.HasPrefix(c.apiKey, "AF") || strings.HasPrefix(c.apiKey, "AG") || len(c.apiKey) > 100 {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
}

// фильтрует транзакции для входящих платежей на указанный адрес
func ParseIncomingTransactions(txs []Transaction, recipientAddress string) []Transaction {
	// нормализуем адрес получателя для сравнения
	normalizedRecipient, _ := NormalizeAddress(recipientAddress)
	if normalizedRecipient == "" {
		normalizedRecipient = recipientAddress
	}

	var incoming []Transaction
	for _, tx := range txs {
		if tx.InMsg == nil || tx.InMsg.Destination == nil || tx.InMsg.Value <= 0 {
			continue
		}

		destAddr := tx.InMsg.Destination.Address

		// нормализуем адрес из транзакции
		normalizedDest, _ := NormalizeAddress(destAddr)
		if normalizedDest == "" {
			normalizedDest = destAddr
		}

		// сравниваем нормализованные адреса
		if normalizedDest == normalizedRecipient {
			incoming = append(incoming, tx)
		}
	}
	return incoming
}

// извлекает текстовую заметку из транзакции
func ExtractMemo(tx *Transaction) string {
	if tx.InMsg != nil && tx.InMsg.DecodedBody != nil {
		return tx.InMsg.DecodedBody.Text
	}
	return ""
}
