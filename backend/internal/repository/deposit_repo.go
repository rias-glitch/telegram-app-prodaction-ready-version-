package repository

import (
	"context"
	"time"

	"telegram_webapp/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DepositRepository struct {
	db *pgxpool.Pool
}

func NewDepositRepository(db *pgxpool.Pool) *DepositRepository {
	return &DepositRepository{db: db}
}

// получает депозит по ID
func (r *DepositRepository) GetByID(ctx context.Context, id int64) (*domain.Deposit, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, wallet_address, amount_nano, gems_credited, exchange_rate,
		       tx_hash, tx_lt, status, memo, created_at, confirmed_at, processed
		FROM deposits
		WHERE id = $1
	`, id)

	return scanDeposit(row)
}

//получает депозит по хэшу транзакции
func (r *DepositRepository) GetByTxHash(ctx context.Context, txHash string) (*domain.Deposit, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, wallet_address, amount_nano, gems_credited, exchange_rate,
		       tx_hash, tx_lt, status, memo, created_at, confirmed_at, processed
		FROM deposits
		WHERE tx_hash = $1
	`, txHash)

	return scanDeposit(row)
}

// получает все депозиты пользователя
func (r *DepositRepository) GetByUserID(ctx context.Context, userID int64, limit int) ([]domain.Deposit, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, wallet_address, amount_nano, gems_credited, exchange_rate,
		       tx_hash, tx_lt, status, memo, created_at, confirmed_at, processed
		FROM deposits
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDeposits(rows)
}

// получает все ожидающие обработки депозиты
func (r *DepositRepository) GetPending(ctx context.Context) ([]domain.Deposit, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, wallet_address, amount_nano, gems_credited, exchange_rate,
		       tx_hash, tx_lt, status, memo, created_at, confirmed_at, processed
		FROM deposits
		WHERE status = 'pending' AND NOT processed
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDeposits(rows)
}

// создает новую запись о депозите
func (r *DepositRepository) Create(ctx context.Context, d *domain.Deposit) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO deposits (user_id, wallet_address, amount_nano, gems_credited, exchange_rate, tx_hash, tx_lt, status, memo)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`, d.UserID, d.WalletAddress, d.AmountNano, d.GemsCredited, d.ExchangeRate, d.TxHash, d.TxLt, d.Status, d.Memo).Scan(&d.ID, &d.CreatedAt)
}

// подтверждает депозит и начисляет драгоценные камни (gems)
func (r *DepositRepository) Confirm(ctx context.Context, id int64) error {
	now := time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE deposits
		SET status = 'confirmed', confirmed_at = $2, processed = true
		WHERE id = $1
	`, id, now)
	return err
}

// помечает депозит как неудачный
func (r *DepositRepository) Fail(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE deposits SET status = 'failed' WHERE id = $1
	`, id)
	return err
}

// проверяет, существует ли уже хэш транзакции
func (r *DepositRepository) TxHashExists(ctx context.Context, txHash string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM deposits WHERE tx_hash = $1)
	`, txHash).Scan(&exists)
	return exists, err
}

// возвращает общее количество драгоценных гемов, внесенных пользователем
func (r *DepositRepository) GetTotalDeposited(ctx context.Context, userID int64) (int64, error) {
	var total int64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(gems_credited), 0)
		FROM deposits
		WHERE user_id = $1 AND status = 'confirmed'
	`, userID).Scan(&total)
	return total, err
}

// преобразует строку из БД в структуру Deposit
func scanDeposit(row pgx.Row) (*domain.Deposit, error) {
	var d domain.Deposit
	var txLt *int64
	var memo *string
	var confirmedAt *time.Time

	if err := row.Scan(
		&d.ID, &d.UserID, &d.WalletAddress, &d.AmountNano, &d.GemsCredited, &d.ExchangeRate,
		&d.TxHash, &txLt, &d.Status, &memo, &d.CreatedAt, &confirmedAt, &d.Processed,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if txLt != nil {
		d.TxLt = *txLt
	}
	if memo != nil {
		d.Memo = *memo
	}
	d.ConfirmedAt = confirmedAt

	// CoinsCredited = GemsCredited для обратной совместимости
	// (в БД хранится в колонке gems_credited, но фактически это coins)
	d.CoinsCredited = d.GemsCredited

	return &d, nil
}

//преобразует набор строк из БД в срез структур Deposit
func scanDeposits(rows pgx.Rows) ([]domain.Deposit, error) {
	var deposits []domain.Deposit

	for rows.Next() {
		var d domain.Deposit
		var txLt *int64
		var memo *string
		var confirmedAt *time.Time

		if err := rows.Scan(
			&d.ID, &d.UserID, &d.WalletAddress, &d.AmountNano, &d.GemsCredited, &d.ExchangeRate,
			&d.TxHash, &txLt, &d.Status, &memo, &d.CreatedAt, &confirmedAt, &d.Processed,
		); err != nil {
			return nil, err
		}

		if txLt != nil {
			d.TxLt = *txLt
		}
		if memo != nil {
			d.Memo = *memo
		}
		d.ConfirmedAt = confirmedAt

		// CoinsCredited = GemsCredited для обратной совместимости
		d.CoinsCredited = d.GemsCredited

		deposits = append(deposits, d)
	}

	return deposits, nil
}