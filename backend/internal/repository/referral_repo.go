package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Referral struct {
	ID           int64     `json:"id"`
	ReferrerID   int64     `json:"referrer_id"`
	ReferredID   int64     `json:"referred_id"`
	BonusClaimed bool      `json:"bonus_claimed"`
	CreatedAt    time.Time `json:"created_at"`
}

type ReferralStats struct {
	TotalReferrals int   `json:"total_referrals"`
	TotalEarned    int64 `json:"total_earned"`
}

type ReferralRepository struct {
	db *pgxpool.Pool
}

func NewReferralRepository(db *pgxpool.Pool) *ReferralRepository {
	return &ReferralRepository{db: db}
}

// Генерирует уникальный реферальный код
func GenerateReferralCode() string {
	bytes := make([]byte, 6)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Получает существующий или создает новый реферальный код для пользователя
func (r *ReferralRepository) GetOrCreateReferralCode(ctx context.Context, userID int64) (string, error) {
	// Сначала пытаемся получить существующий код
	var code string
	err := r.db.QueryRow(ctx,
		`SELECT referral_code FROM users WHERE id = $1`,
		userID,
	).Scan(&code)

	if err == nil && code != "" {
		return code, nil
	}

	// Генерируем новый код
	for i := 0; i < 5; i++ { // Пытаемся до 5 раз в случае коллизии
		code = GenerateReferralCode()
		_, err = r.db.Exec(ctx,
			`UPDATE users SET referral_code = $1 WHERE id = $2`,
			code, userID,
		)
		if err == nil {
			return code, nil
		}
	}

	return "", err
}

// Находит пользователя по его реферальному коду
func (r *ReferralRepository) GetUserByReferralCode(ctx context.Context, code string) (int64, error) {
	var userID int64
	err := r.db.QueryRow(ctx,
		`SELECT id FROM users WHERE referral_code = $1`,
		code,
	).Scan(&userID)
	return userID, err
}

// Создает новую реферальную связь
func (r *ReferralRepository) CreateReferral(ctx context.Context, referrerID, referredID int64) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO referrals (referrer_id, referred_id)
		 VALUES ($1, $2)
		 ON CONFLICT (referred_id) DO NOTHING`,
		referrerID, referredID,
	)
	if err != nil {
		return err
	}

	// Также обновляем поле referred_by в таблице users
	_, err = r.db.Exec(ctx,
		`UPDATE users SET referred_by = $1 WHERE id = $2 AND referred_by IS NULL`,
		referrerID, referredID,
	)
	return err
}

// Возвращает все рефералы, сделанные пользователем
func (r *ReferralRepository) GetReferralsByUser(ctx context.Context, userID int64) ([]Referral, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, referrer_id, referred_id, bonus_claimed, created_at
		 FROM referrals
		 WHERE referrer_id = $1
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var referrals []Referral
	for rows.Next() {
		var ref Referral
		if err := rows.Scan(&ref.ID, &ref.ReferrerID, &ref.ReferredID, &ref.BonusClaimed, &ref.CreatedAt); err != nil {
			continue
		}
		referrals = append(referrals, ref)
	}

	return referrals, nil
}

// Возвращает реферальную статистику для пользователя
func (r *ReferralRepository) GetReferralStats(ctx context.Context, userID int64) (*ReferralStats, error) {
	stats := &ReferralStats{}

	// Считаем общее количество рефералов
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM referrals WHERE referrer_id = $1`,
		userID,
	).Scan(&stats.TotalReferrals)
	if err != nil {
		return nil, err
	}

	// Вычисляем общий заработок (500 gems за каждого реферала)
	stats.TotalEarned = int64(stats.TotalReferrals) * 500

	return stats, nil
}

// Отмечает бонус как полученный и начисляет награды
func (r *ReferralRepository) ClaimReferralBonus(ctx context.Context, referralID int64, referrerID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Отмечаем как полученный
	result, err := tx.Exec(ctx,
		`UPDATE referrals SET bonus_claimed = true
		 WHERE id = $1 AND referrer_id = $2 AND bonus_claimed = false`,
		referralID, referrerID,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return nil // Уже получено или не найдено
	}

	// Добавляем бонусные gems пригласившему
	_, err = tx.Exec(ctx,
		`UPDATE users SET gems = gems + 500 WHERE id = $1`,
		referrerID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Проверяет, был ли пользователь приглашен кем-то
func (r *ReferralRepository) IsReferred(ctx context.Context, userID int64) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM referrals WHERE referred_id = $1`,
		userID,
	).Scan(&count)
	return count > 0, err
}

// Возвращает ID пригласившего для данного пользователя
func (r *ReferralRepository) GetReferrerID(ctx context.Context, userID int64) (int64, error) {
	var referrerID int64
	err := r.db.QueryRow(ctx,
		`SELECT referred_by FROM users WHERE id = $1 AND referred_by IS NOT NULL`,
		userID,
	).Scan(&referrerID)
	return referrerID, err
}

// Возвращает список полученных пороговых наград GK для пользователя
func (r *ReferralRepository) GetClaimedGKRewards(ctx context.Context, userID int64) ([]int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT threshold FROM gk_rewards_claimed WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return []int{}, nil // Возвращаем пустой список, если таблица еще не существует
	}
	defer rows.Close()

	var thresholds []int
	for rows.Next() {
		var t int
		if err := rows.Scan(&t); err != nil {
			continue
		}
		thresholds = append(thresholds, t)
	}
	return thresholds, nil
}

// Проверяет, была ли получена пороговая награда GK
func (r *ReferralRepository) IsGKRewardClaimed(ctx context.Context, userID int64, threshold int) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM gk_rewards_claimed WHERE user_id = $1 AND threshold = $2`,
		userID, threshold,
	).Scan(&count)
	if err != nil {
		return false, nil // Предполагаем не полученным, если таблица не существует
	}
	return count > 0, nil
}

// Получает награду GK и добавляет GK к балансу пользователя
func (r *ReferralRepository) ClaimGKReward(ctx context.Context, userID int64, threshold int, reward int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Записываем получение награды
	_, err = tx.Exec(ctx,
		`INSERT INTO gk_rewards_claimed (user_id, threshold, reward, claimed_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (user_id, threshold) DO NOTHING`,
		userID, threshold, reward,
	)
	if err != nil {
		return err
	}

	// Добавляем GK пользователю
	_, err = tx.Exec(ctx,
		`UPDATE users SET gk = COALESCE(gk, 0) + $1 WHERE id = $2`,
		reward, userID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Возвращает всех пользователей с количеством их рефералов (для администратора)
func (r *ReferralRepository) GetAllReferralStats(ctx context.Context, limit int) ([]struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	Count     int    `json:"count"`
}, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, u.username, u.first_name, COUNT(r.id) as ref_count
		FROM users u
		LEFT JOIN referrals r ON r.referrer_id = u.id
		GROUP BY u.id, u.username, u.first_name
		HAVING COUNT(r.id) > 0
		ORDER BY ref_count DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []struct {
		UserID    int64  `json:"user_id"`
		Username  string `json:"username"`
		FirstName string `json:"first_name"`
		Count     int    `json:"count"`
	}

	for rows.Next() {
		var r struct {
			UserID    int64  `json:"user_id"`
			Username  string `json:"username"`
			FirstName string `json:"first_name"`
			Count     int    `json:"count"`
		}
		if err := rows.Scan(&r.UserID, &r.Username, &r.FirstName, &r.Count); err != nil {
			continue
		}
		results = append(results, r)
	}
	return results, nil
}
