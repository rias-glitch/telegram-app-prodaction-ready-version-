package repository

import (
	"context"
	"encoding/json"

	"telegram_webapp/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// отвечает за операции с базой данных для логов аудита
type AuditRepository struct {
	db *pgxpool.Pool
}

// создает новый репозиторий для логов аудита
func NewAuditRepository(db *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{db: db}
}

// создает новую запись в логе аудита
func (r *AuditRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	detailsJSON, err := json.Marshal(log.Details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO audit_logs (user_id, action, category, details, ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, log.UserID, log.Action, log.Category, detailsJSON, log.IP, log.UserAgent)
	return err
}

// создает новую запись в логе аудита внутри транзакции
func (r *AuditRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, log *domain.AuditLog) error {
	detailsJSON, err := json.Marshal(log.Details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (user_id, action, category, details, ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, log.UserID, log.Action, log.Category, detailsJSON, log.IP, log.UserAgent)
	return err
}

// возвращает логи аудита для пользователя
func (r *AuditRepository) GetByUserID(ctx context.Context, userID int64, limit int) ([]*domain.AuditLog, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, action, category, details, ip, user_agent, created_at
		FROM audit_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAuditLogs(rows)
}

// возвращает логи аудита по категории
func (r *AuditRepository) GetByCategory(ctx context.Context, category string, limit int) ([]*domain.AuditLog, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, action, category, details, ip, user_agent, created_at
		FROM audit_logs
		WHERE category = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, category, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAuditLogs(rows)
}

// возвращает логи аудита по действию
func (r *AuditRepository) GetByAction(ctx context.Context, action string, limit int) ([]*domain.AuditLog, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, action, category, details, ip, user_agent, created_at
		FROM audit_logs
		WHERE action = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, action, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAuditLogs(rows)
}

//  возвращает самые последние логи аудита
func (r *AuditRepository) GetRecent(ctx context.Context, limit int) ([]*domain.AuditLog, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, action, category, details, ip, user_agent, created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAuditLogs(rows)
}

// преобразует строки из БД в структуры AuditLog
func scanAuditLogs(rows pgx.Rows) ([]*domain.AuditLog, error) {
	var logs []*domain.AuditLog
	for rows.Next() {
		var log domain.AuditLog
		var detailsJSON []byte
		if err := rows.Scan(&log.ID, &log.UserID, &log.Action, &log.Category, &detailsJSON, &log.IP, &log.UserAgent, &log.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(detailsJSON, &log.Details); err != nil {
			log.Details = make(map[string]interface{})
		}
		logs = append(logs, &log)
	}
	return logs, rows.Err()
}