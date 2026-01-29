package repository

import (
	"context"
	"errors"

	"telegram_webapp/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInsufficientFunds = errors.New("insufficient funds")

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// получает пользователя по telegram id
func (r *UserRepository) GetByTgID(ctx context.Context, tgID int64) (*domain.User, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, tg_id, COALESCE(username, ''), COALESCE(first_name, ''), gems, COALESCE(coins, 0), created_at
		 FROM users
		 WHERE tg_id = $1`,
		tgID,
	)

	var u domain.User
	if err := row.Scan(
		&u.ID,
		&u.TgID,
		&u.Username,
		&u.FirstName,
		&u.Gems,
		&u.Coins,
		&u.CreatedAt,
	); err != nil {
		return nil, err
	}

	// пробуем получить новые колонки, если они существуют
	_ = r.db.QueryRow(ctx,
		`SELECT COALESCE(gk, 0), COALESCE(character_level, 1), COALESCE(referral_earnings, 0)
		 FROM users WHERE id = $1`, u.ID,
	).Scan(&u.GK, &u.CharacterLevel, &u.ReferralEarnings)

	return &u, nil
}

// создает нового пользователя
func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	// начальный баланс для новых пользователей
	const initialGems = 10000

	return r.db.QueryRow(ctx,
		`INSERT INTO users (tg_id, username, first_name, gems)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		u.TgID,
		u.Username,
		u.FirstName,
		initialGems,
	).Scan(&u.ID)
}

// получает пользователя по id
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, tg_id, COALESCE(username, ''), COALESCE(first_name, ''), gems, COALESCE(coins, 0), created_at
		 FROM users
		 WHERE id = $1`,
		id,
	)

	var u domain.User
	if err := row.Scan(
		&u.ID,
		&u.TgID,
		&u.Username,
		&u.FirstName,
		&u.Gems,
		&u.Coins,
		&u.CreatedAt,
	); err != nil {
		return nil, err
	}

	// пробуем получить новые колонки, если они существуют
	_ = r.db.QueryRow(ctx,
		`SELECT COALESCE(gk, 0), COALESCE(character_level, 1), COALESCE(referral_earnings, 0)
		 FROM users WHERE id = $1`, u.ID,
	).Scan(&u.GK, &u.CharacterLevel, &u.ReferralEarnings)

	return &u, nil
}

// возвращает пользователей, отсортированных по количеству побед
func (r *UserRepository) GetTopByWins(ctx context.Context, limit int) ([]struct {
	User domain.User `json:"user"`
	Wins int64       `json:"wins"`
}, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, u.tg_id, u.username, u.first_name, u.gems, u.coins, u.created_at, COALESCE(wc.wins,0) as wins
		FROM users u
		LEFT JOIN (
			SELECT winner_id, COUNT(*) as wins FROM games WHERE winner_id IS NOT NULL GROUP BY winner_id
		) wc ON wc.winner_id = u.id
		ORDER BY wins DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []struct {
		User domain.User `json:"user"`
		Wins int64       `json:"wins"`
	}

	for rows.Next() {
		var u domain.User
		var wins int64
		if err := rows.Scan(&u.ID, &u.TgID, &u.Username, &u.FirstName, &u.Gems, &u.Coins, &u.CreatedAt, &wins); err != nil {
			return nil, err
		}
		res = append(res, struct {
			User domain.User `json:"user"`
			Wins int64       `json:"wins"`
		}{User: u, Wins: wins})
	}
	return res, nil
}

// обновляет баланс монет пользователя
func (r *UserRepository) UpdateCoins(ctx context.Context, userID int64, delta int64) (int64, error) {
	var newBalance int64
	err := r.db.QueryRow(ctx,
		`UPDATE users SET coins = coins + $1 WHERE id = $2 AND coins + $1 >= 0 RETURNING coins`,
		delta, userID,
	).Scan(&newBalance)
	return newBalance, err
}

// возвращает баланс монет пользователя
func (r *UserRepository) GetCoins(ctx context.Context, userID int64) (int64, error) {
	var coins int64
	err := r.db.QueryRow(ctx, `SELECT coins FROM users WHERE id = $1`, userID).Scan(&coins)
	return coins, err
}

// обновляет баланс драгоценных камней пользователя
func (r *UserRepository) UpdateGems(ctx context.Context, userID int64, delta int64) (int64, error) {
	var newBalance int64
	err := r.db.QueryRow(ctx,
		`UPDATE users SET gems = gems + $1 WHERE id = $2 AND gems + $1 >= 0 RETURNING gems`,
		delta, userID,
	).Scan(&newBalance)
	return newBalance, err
}

// возвращает баланс драгоценных камней пользователя
func (r *UserRepository) GetGems(ctx context.Context, userID int64) (int64, error) {
	var gems int64
	err := r.db.QueryRow(ctx, `SELECT gems FROM users WHERE id = $1`, userID).Scan(&gems)
	return gems, err
}

// обновляет баланс GK (игровая валюта) пользователя
func (r *UserRepository) UpdateGK(ctx context.Context, userID int64, delta int64) (int64, error) {
	var newBalance int64
	err := r.db.QueryRow(ctx,
		`UPDATE users SET gk = COALESCE(gk, 0) + $1 WHERE id = $2 AND COALESCE(gk, 0) + $1 >= 0 RETURNING gk`,
		delta, userID,
	).Scan(&newBalance)
	return newBalance, err
}

// возвращает баланс GK пользователя
func (r *UserRepository) GetGK(ctx context.Context, userID int64) (int64, error) {
	var gk int64
	err := r.db.QueryRow(ctx, `SELECT COALESCE(gk, 0) FROM users WHERE id = $1`, userID).Scan(&gk)
	return gk, err
}

// возвращает уровень персонажа пользователя
func (r *UserRepository) GetCharacterLevel(ctx context.Context, userID int64) (int, error) {
	var level int
	err := r.db.QueryRow(ctx, `SELECT COALESCE(character_level, 1) FROM users WHERE id = $1`, userID).Scan(&level)
	return level, err
}

// повышает уровень персонажа и списывает GK
func (r *UserRepository) UpgradeCharacter(ctx context.Context, userID int64, newLevel int, gkCost int64) error {
	result, err := r.db.Exec(ctx,
		`UPDATE users
		 SET character_level = $1, gk = COALESCE(gk, 0) - $2
		 WHERE id = $3 AND COALESCE(character_level, 1) = $1 - 1 AND COALESCE(gk, 0) >= $2`,
		newLevel, gkCost, userID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrInsufficientFunds
	}
	return nil
}

// добавляет доход от реферальной программы пользователю
func (r *UserRepository) AddReferralEarnings(ctx context.Context, userID int64, amount int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET referral_earnings = COALESCE(referral_earnings, 0) + $1 WHERE id = $2`,
		amount, userID,
	)
	return err
}

// представляет пользователя в таблице лидеров
type TopUserEntry struct {
	Rank      int          `json:"rank"`
	User      domain.User  `json:"user"`
	WinsCount int64        `json:"wins_count"`
}

// возвращает лучших пользователей по количеству побед за текущий месяц
func (r *UserRepository) GetMonthlyTop(ctx context.Context, limit int) ([]TopUserEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, u.tg_id, COALESCE(u.username, ''), COALESCE(u.first_name, ''),
		       u.gems, COALESCE(u.coins, 0), u.created_at, COALESCE(w.wins, 0) as wins_count
		FROM users u
		LEFT JOIN (
			SELECT user_id, COUNT(*) as wins
			FROM game_history
			WHERE created_at >= date_trunc('month', CURRENT_DATE) AND result = 'win'
			GROUP BY user_id
		) w ON w.user_id = u.id
		ORDER BY wins_count DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []TopUserEntry
	rank := 1
	for rows.Next() {
		var u domain.User
		var winsCount int64
		if err := rows.Scan(&u.ID, &u.TgID, &u.Username, &u.FirstName, &u.Gems, &u.Coins,
			&u.CreatedAt, &winsCount); err != nil {
			return nil, err
		}
		res = append(res, TopUserEntry{
			Rank:      rank,
			User:      u,
			WinsCount: winsCount,
		})
		rank++
	}
	return res, nil
}

// возвращает ранг пользователя в ежемесячной таблице лидеров
func (r *UserRepository) GetUserRank(ctx context.Context, userID int64) (int, int64, error) {
	var rank int
	var winsCount int64
	err := r.db.QueryRow(ctx, `
		WITH user_wins AS (
			SELECT user_id, COUNT(*) as wins
			FROM game_history
			WHERE created_at >= date_trunc('month', CURRENT_DATE) AND result = 'win'
			GROUP BY user_id
		),
		ranked AS (
			SELECT u.id, COALESCE(w.wins, 0) as wins_count,
			       RANK() OVER (ORDER BY COALESCE(w.wins, 0) DESC) as rank
			FROM users u
			LEFT JOIN user_wins w ON w.user_id = u.id
		)
		SELECT rank, wins_count FROM ranked WHERE id = $1
	`, userID).Scan(&rank, &winsCount)
	if err != nil {
		return 0, 0, err
	}
	return rank, winsCount, nil
}