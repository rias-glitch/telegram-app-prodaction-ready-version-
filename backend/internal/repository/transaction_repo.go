package repository

import (
	"context"
	"encoding/json"
	"time"

	"telegram_webapp/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionRepository struct {
	db *pgxpool.Pool
}

func NewTransactionRepository(db *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// возвращает последние транзакции пользователя
func (r *TransactionRepository) GetByUserID(ctx context.Context, userID int64, limit int) ([]*domain.Transaction, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, type, amount, meta, created_at
		 FROM transactions
		 WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// создает новую транзакцию
func (r *TransactionRepository) Create(ctx context.Context, tx *domain.Transaction) error {
	metaJSON, err := json.Marshal(tx.Meta)
	if err != nil {
		metaJSON = []byte("{}")
	}

	return r.db.QueryRow(ctx,
		`INSERT INTO transactions (user_id, type, amount, meta)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`,
		tx.UserID, tx.Type, tx.Amount, metaJSON,
	).Scan(&tx.ID, &tx.CreatedAt)
}

// создает транзакцию с использованием существующей транзакции базы данных
func (r *TransactionRepository) CreateWithTx(ctx context.Context, dbTx pgx.Tx, tx *domain.Transaction) error {
	metaJSON, err := json.Marshal(tx.Meta)
	if err != nil {
		metaJSON = []byte("{}")
	}

	return dbTx.QueryRow(ctx,
		`INSERT INTO transactions (user_id, type, amount, meta)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`,
		tx.UserID, tx.Type, tx.Amount, metaJSON,
	).Scan(&tx.ID, &tx.CreatedAt)
}

// возвращает транзакции пользователя, отфильтрованные по типу
func (r *TransactionRepository) GetByUserIDAndType(ctx context.Context, userID int64, txType string, limit int) ([]*domain.Transaction, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, type, amount, meta, created_at
		 FROM transactions
		 WHERE user_id = $1 AND type = $2
		 ORDER BY created_at DESC
		 LIMIT $3`,
		userID, txType, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// вспомогательный метод для сканирования строк в срез транзакций
func (r *TransactionRepository) scanRows(rows pgx.Rows) ([]*domain.Transaction, error) {
	var result []*domain.Transaction

	for rows.Next() {
		var (
			tx       domain.Transaction
			metaJSON []byte
			createdAt time.Time
		)

		if err := rows.Scan(&tx.ID, &tx.UserID, &tx.Type, &tx.Amount, &metaJSON, &createdAt); err != nil {
			return nil, err
		}

		tx.CreatedAt = createdAt
		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &tx.Meta)
		}

		result = append(result, &tx)
	}

	return result, nil
}