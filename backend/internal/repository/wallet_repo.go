package repository

import (
	"context"
	"fmt"
	"strings"

	"telegram_webapp/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WalletRepository struct {
	db *pgxpool.Pool
}

func NewWalletRepository(db *pgxpool.Pool) *WalletRepository {
	return &WalletRepository{db: db}
}

// получает кошелек по id пользователя
func (r *WalletRepository) GetByUserID(ctx context.Context, userID int64) (*domain.Wallet, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, address, raw_address, linked_at, is_verified, last_proof_timestamp
		FROM wallets
		WHERE user_id = $1
	`, userID)

	var w domain.Wallet
	var rawAddr *string
	var lastProofTs *int64

	if err := row.Scan(
		&w.ID, &w.UserID, &w.Address, &rawAddr, &w.LinkedAt, &w.IsVerified, &lastProofTs,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if rawAddr != nil {
		w.RawAddress = *rawAddr
	}
	if lastProofTs != nil {
		w.LastProofTimestamp = *lastProofTs
	}

	return &w, nil
}

// получает кошелек по адресу в сети ton
func (r *WalletRepository) GetByAddress(ctx context.Context, address string) (*domain.Wallet, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, address, raw_address, linked_at, is_verified, last_proof_timestamp
		FROM wallets
		WHERE address = $1 OR raw_address = $1
	`, address)

	var w domain.Wallet
	var rawAddr *string
	var lastProofTs *int64

	if err := row.Scan(
		&w.ID, &w.UserID, &w.Address, &rawAddr, &w.LinkedAt, &w.IsVerified, &lastProofTs,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if rawAddr != nil {
		w.RawAddress = *rawAddr
	}
	if lastProofTs != nil {
		w.LastProofTimestamp = *lastProofTs
	}

	return &w, nil
}

// создает новую привязку кошелька
func (r *WalletRepository) Create(ctx context.Context, w *domain.Wallet) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO wallets (user_id, address, raw_address, is_verified, last_proof_timestamp)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, linked_at
	`, w.UserID, w.Address, w.RawAddress, w.IsVerified, w.LastProofTimestamp).Scan(&w.ID, &w.LinkedAt)
}

// обновляет информацию о кошельке
func (r *WalletRepository) Update(ctx context.Context, w *domain.Wallet) error {
	_, err := r.db.Exec(ctx, `
		UPDATE wallets
		SET address = $2, raw_address = $3, is_verified = $4, last_proof_timestamp = $5
		WHERE id = $1
	`, w.ID, w.Address, w.RawAddress, w.IsVerified, w.LastProofTimestamp)
	return err
}

// удаляет привязку кошелька
func (r *WalletRepository) Delete(ctx context.Context, userID int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM wallets WHERE user_id = $1`, userID)
	return err
}

// помечает кошелек как верифицированный
func (r *WalletRepository) SetVerified(ctx context.Context, userID int64, verified bool) error {
	_, err := r.db.Exec(ctx, `
		UPDATE wallets SET is_verified = $2 WHERE user_id = $1
	`, userID, verified)
	return err
}

// проверяет, есть ли у пользователя привязанный кошелек
func (r *WalletRepository) Exists(ctx context.Context, userID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM wallets WHERE user_id = $1)
	`, userID).Scan(&exists)
	return exists, err
}

// проверяет, привязан ли адрес к какому-либо пользователю
func (r *WalletRepository) AddressExists(ctx context.Context, address string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM wallets WHERE address = $1 OR raw_address = $1)
	`, address).Scan(&exists)
	return exists, err
}

// GetByAnyAddress ищет кошелек по любому из переданных адресов (разные форматы одного адреса)
func (r *WalletRepository) GetByAnyAddress(ctx context.Context, addresses []string) (*domain.Wallet, error) {
	if len(addresses) == 0 {
		return nil, nil
	}

	// Строим запрос с OR для всех адресов
	query := `
		SELECT id, user_id, address, raw_address, linked_at, is_verified, last_proof_timestamp
		FROM wallets
		WHERE `

	args := make([]interface{}, 0, len(addresses)*2)
	conditions := make([]string, 0, len(addresses))

	for _, addr := range addresses {
		if addr == "" {
			continue
		}
		argIdx := len(args) + 1
		conditions = append(conditions, fmt.Sprintf("(address = $%d OR raw_address = $%d)", argIdx, argIdx))
		args = append(args, addr)
	}

	if len(conditions) == 0 {
		return nil, nil
	}

	query += strings.Join(conditions, " OR ") + " LIMIT 1"

	row := r.db.QueryRow(ctx, query, args...)

	var w domain.Wallet
	var rawAddr *string
	var lastProofTs *int64

	if err := row.Scan(
		&w.ID, &w.UserID, &w.Address, &rawAddr, &w.LinkedAt, &w.IsVerified, &lastProofTs,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if rawAddr != nil {
		w.RawAddress = *rawAddr
	}
	if lastProofTs != nil {
		w.LastProofTimestamp = *lastProofTs
	}

	return &w, nil
}