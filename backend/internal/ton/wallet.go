package ton

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

// Wallet представляет TON кошелек для отправки транзакций
type Wallet struct {
	client  *ton.APIClient
	wallet  *wallet.Wallet
	network Network
}

// SendResult результат отправки транзакции
type SendResult struct {
	TxHash  string
	Success bool
}

// NewWallet создает новый кошелек из мнемоники
func NewWallet(mnemonic string, network Network) (*Wallet, error) {
	// Выбираем конфиг сети
	configURL := "https://ton.org/global.config.json"
	if network == NetworkTestnet {
		configURL = "https://ton.org/testnet-global.config.json"
	}

	// Подключаемся к лайтсерверам
	client := liteclient.NewConnectionPool()
	err := client.AddConnectionsFromConfigUrl(context.Background(), configURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to lite servers: %w", err)
	}

	api := ton.NewAPIClient(client)

	// Парсим мнемонику
	words := strings.Fields(strings.TrimSpace(mnemonic))
	if len(words) != 24 {
		return nil, fmt.Errorf("invalid mnemonic: expected 24 words, got %d", len(words))
	}

	// Создаем кошелек (V5R1 Final - версия W5 из Tonkeeper)
	// NetworkGlobalID: -239 для mainnet, -3 для testnet
	networkID := int32(-239)
	if network == NetworkTestnet {
		networkID = -3
	}
	w, err := wallet.FromSeed(api, words, wallet.ConfigV5R1Final{
		NetworkGlobalID: networkID,
		Workchain:       0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet from seed: %w", err)
	}

	return &Wallet{
		client:  api,
		wallet:  w,
		network: network,
	}, nil
}

// GetAddress возвращает адрес кошелька
func (w *Wallet) GetAddress() string {
	return w.wallet.WalletAddress().String()
}

// GetBalance возвращает баланс кошелька в нанотонах
func (w *Wallet) GetBalance(ctx context.Context) (uint64, error) {
	walletAddr := w.wallet.WalletAddress()
	fmt.Printf("[GetBalance] Checking balance for wallet: %s (network: %s)\n", walletAddr.String(), w.network)

	block, err := w.client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get masterchain info: %w", err)
	}

	acc, err := w.client.GetAccount(ctx, block, walletAddr)
	if err != nil {
		return 0, fmt.Errorf("failed to get account: %w", err)
	}

	if acc.State == nil {
		fmt.Printf("[GetBalance] Account state is nil (wallet not deployed or empty)\n")
		return 0, nil
	}

	balance := acc.State.Balance.Nano().Uint64()
	fmt.Printf("[GetBalance] Balance: %d nano (%f TON)\n", balance, float64(balance)/1e9)
	return balance, nil
}

// SendTON отправляет TON на указанный адрес
// amount в нанотонах (1 TON = 1_000_000_000 наноТОН)
// comment - опциональный комментарий к транзакции
func (w *Wallet) SendTON(ctx context.Context, toAddress string, amountNano uint64, comment string) (*SendResult, error) {
	fmt.Printf("[SendTON] Attempting to parse address: %q (len=%d)\n", toAddress, len(toAddress))

	var addr *address.Address
	var err error

	// Проверяем формат адреса
	if strings.HasPrefix(toAddress, "0:") || strings.HasPrefix(toAddress, "-1:") {
		// Raw формат (0:hex или -1:hex) - создаём адрес напрямую
		addr, err = parseRawAddress(toAddress)
		if err != nil {
			fmt.Printf("[SendTON] parseRawAddress failed: %v\n", err)
			return nil, fmt.Errorf("invalid raw address: %w (original: %s)", err, toAddress)
		}
		fmt.Printf("[SendTON] Parsed raw address successfully: %s\n", addr.String())
	} else {
		// User-friendly формат (EQ.../UQ...)
		addr, err = address.ParseAddr(toAddress)
		if err != nil {
			fmt.Printf("[SendTON] ParseAddr failed: %v\n", err)
			return nil, fmt.Errorf("invalid recipient address: %w (original: %s)", err, toAddress)
		}
		fmt.Printf("[SendTON] Parsed as user-friendly address\n")
	}

	// Проверяем баланс
	balance, err := w.GetBalance(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check balance: %w", err)
	}

	// Учитываем комиссию сети (~0.01 TON)
	networkFee := uint64(10_000_000) // 0.01 TON
	if balance < amountNano+networkFee {
		return nil, fmt.Errorf("insufficient balance: have %d, need %d + fee", balance, amountNano)
	}

	// Создаем сообщение с или без комментария
	amount := tlb.MustFromTON(fmt.Sprintf("%.9f", float64(amountNano)/1e9))

	var msg *wallet.Message
	if comment != "" {
		// Создаем payload с комментарием (0x00000000 + UTF-8 текст)
		commentCell := buildCommentCell(comment)
		msg = &wallet.Message{
			Mode: wallet.PayGasSeparately + wallet.IgnoreErrors,
			InternalMessage: &tlb.InternalMessage{
				IHRDisabled: true,
				Bounce:      false,
				DstAddr:     addr,
				Amount:      amount,
				Body:        commentCell,
			},
		}
	} else {
		msg = wallet.SimpleMessage(addr, amount, nil)
	}

	// Отправляем транзакцию
	tx, _, err := w.wallet.SendWaitTransaction(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	// Получаем хэш транзакции
	txHash := fmt.Sprintf("%x", tx.Hash)

	return &SendResult{
		TxHash:  txHash,
		Success: true,
	}, nil
}

// buildCommentCell создает cell с текстовым комментарием
func buildCommentCell(comment string) *cell.Cell {
	// Комментарий в TON: 32 бита нулей + UTF-8 текст
	c := cell.BeginCell().
		MustStoreUInt(0, 32). // op = 0 (text comment)
		MustStoreStringSnake(comment).
		EndCell()
	return c
}

// parseRawAddress парсит raw адрес формата "0:hex" или "-1:hex" и создаёт Address
func parseRawAddress(rawAddr string) (*address.Address, error) {
	var workchain int32
	var hashHex string

	if strings.HasPrefix(rawAddr, "0:") {
		workchain = 0
		hashHex = rawAddr[2:]
	} else if strings.HasPrefix(rawAddr, "-1:") {
		workchain = -1
		hashHex = rawAddr[3:]
	} else {
		return nil, fmt.Errorf("unknown raw address format: %s", rawAddr)
	}

	hashBytes, err := hex.DecodeString(hashHex)
	if err != nil {
		return nil, fmt.Errorf("invalid hex in address: %w", err)
	}

	if len(hashBytes) != 32 {
		return nil, fmt.Errorf("invalid hash length: expected 32 bytes, got %d", len(hashBytes))
	}

	// Создаём адрес напрямую используя tonutils-go
	addr := address.NewAddress(0, byte(workchain), hashBytes)
	return addr, nil
}
