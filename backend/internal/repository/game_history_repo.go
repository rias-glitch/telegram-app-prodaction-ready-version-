package repository

import (
	"context"
	"encoding/json"
	"time"

	"telegram_webapp/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type GameHistoryRepository struct {
	db *pgxpool.Pool
}

func NewGameHistoryRepository(db *pgxpool.Pool) *GameHistoryRepository {
	return &GameHistoryRepository{db: db}
}

// сохраняет запись игры в историю
func (r *GameHistoryRepository) Create(ctx context.Context, gh *domain.GameHistory) error {
	detailsJSON, err := json.Marshal(gh.Details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	err = r.db.QueryRow(ctx,
		`INSERT INTO game_history
			(user_id, game_type, mode, opponent_id, room_id, result, bet_amount, win_amount, details)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, created_at`,
		gh.UserID,
		gh.GameType,
		gh.Mode,
		gh.OpponentID,
		gh.RoomID,
		gh.Result,
		gh.BetAmount,
		gh.WinAmount,
		detailsJSON,
	).Scan(&gh.ID, &gh.CreatedAt)

	return err
}

// возвращает историю игр пользователя
func (r *GameHistoryRepository) GetByUser(ctx context.Context, userID int64, limit int) ([]*domain.GameHistory, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, game_type, mode, opponent_id, room_id, result,
				bet_amount, win_amount, details, created_at
		 FROM game_history
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

// возвращает историю игр определённого типа
func (r *GameHistoryRepository) GetByUserAndType(ctx context.Context, userID int64, gameType domain.GameType, limit int) ([]*domain.GameHistory, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, game_type, mode, opponent_id, room_id, result,
				bet_amount, win_amount, details, created_at
		 FROM game_history
		 WHERE user_id = $1 AND game_type = $2
		 ORDER BY created_at DESC
		 LIMIT $3`,
		userID, gameType, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// статистика пользователя
type UserStats struct {
	UserID     int64 `json:"user_id"`
	TotalGames int   `json:"total_games"`
	Wins       int   `json:"wins"`
	Losses     int   `json:"losses"`
	Draws      int   `json:"draws"`
	TotalWon   int64 `json:"total_won"`
	TotalLost  int64 `json:"total_lost"`
}

// возвращает статистику пользователя за период
func (r *GameHistoryRepository) GetUserStats(ctx context.Context, userID int64, since time.Time) (*UserStats, error) {
	stats := &UserStats{UserID: userID}

	err := r.db.QueryRow(ctx,
		`SELECT
			COUNT(*) as total_games,
			COUNT(*) FILTER (WHERE result = 'win') as wins,
			COUNT(*) FILTER (WHERE result = 'lose') as losses,
			COUNT(*) FILTER (WHERE result = 'draw') as draws,
			COALESCE(SUM(win_amount) FILTER (WHERE win_amount > 0), 0) as total_won,
			COALESCE(ABS(SUM(win_amount) FILTER (WHERE win_amount < 0)), 0) as total_lost
		 FROM game_history
		 WHERE user_id = $1 AND created_at >= $2`,
		userID, since,
	).Scan(&stats.TotalGames, &stats.Wins, &stats.Losses, &stats.Draws, &stats.TotalWon, &stats.TotalLost)

	if err != nil {
		return nil, err
	}

	return stats, nil
}

// запись в топе
type TopUser struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	Gems      int64  `json:"gems"`
	Wins      int    `json:"wins"`
	Games     int    `json:"games"`
}

// возвращает топ пользователей за последний месяц
func (r *GameHistoryRepository) GetTopUsers(ctx context.Context, limit int) ([]*TopUser, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.db.Query(ctx,
		`SELECT
			u.id,
			COALESCE(u.username, '') as username,
			COALESCE(u.first_name, '') as first_name,
			u.gems,
			COUNT(*) FILTER (WHERE gh.result = 'win') as wins,
			COUNT(*) as games
		 FROM users u
		 LEFT JOIN game_history gh ON u.id = gh.user_id
			AND gh.created_at >= now() - interval '1 month'
		 GROUP BY u.id, u.username, u.first_name, u.gems
		 ORDER BY wins DESC, games DESC, u.gems DESC
		 LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*TopUser
	for rows.Next() {
		var tu TopUser
		if err := rows.Scan(&tu.UserID, &tu.Username, &tu.FirstName, &tu.Gems, &tu.Wins, &tu.Games); err != nil {
			return nil, err
		}
		result = append(result, &tu)
	}

	return result, nil
}

//подсчитывает действия пользователя для квестов
func (r *GameHistoryRepository) CountUserActions(ctx context.Context, userID int64, actionType domain.ActionType, gameType *string, since time.Time) (int, error) {
	var count int
	var query string
	var args []interface{}

	switch actionType {
	case domain.ActionTypePlay:
		if gameType == nil || *gameType == "any" {
			query = `SELECT COUNT(*) FROM game_history WHERE user_id = $1 AND created_at >= $2`
			args = []interface{}{userID, since}
		} else {
			query = `SELECT COUNT(*) FROM game_history WHERE user_id = $1 AND game_type = $2 AND created_at >= $3`
			args = []interface{}{userID, *gameType, since}
		}
	case domain.ActionTypeWin:
		if gameType == nil || *gameType == "any" {
			query = `SELECT COUNT(*) FROM game_history WHERE user_id = $1 AND result = 'win' AND created_at >= $2`
			args = []interface{}{userID, since}
		} else {
			query = `SELECT COUNT(*) FROM game_history WHERE user_id = $1 AND game_type = $2 AND result = 'win' AND created_at >= $3`
			args = []interface{}{userID, *gameType, since}
		}
	case domain.ActionTypeLose:
		if gameType == nil || *gameType == "any" {
			query = `SELECT COUNT(*) FROM game_history WHERE user_id = $1 AND result = 'lose' AND created_at >= $2`
			args = []interface{}{userID, since}
		} else {
			query = `SELECT COUNT(*) FROM game_history WHERE user_id = $1 AND game_type = $2 AND result = 'lose' AND created_at >= $3`
			args = []interface{}{userID, *gameType, since}
		}
	default:
		return 0, nil
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&count)
	return count, err
}

// для сканирования строк
func (r *GameHistoryRepository) scanRows(rows interface{ Next() bool; Scan(dest ...interface{}) error }) ([]*domain.GameHistory, error) {
	var result []*domain.GameHistory

	for rows.Next() {
		var (
			gh          domain.GameHistory
			detailsJSON []byte
		)

		if err := rows.Scan(
			&gh.ID, &gh.UserID, &gh.GameType, &gh.Mode, &gh.OpponentID,
			&gh.RoomID, &gh.Result, &gh.BetAmount, &gh.WinAmount,
			&detailsJSON, &gh.CreatedAt,
		); err != nil {
			return nil, err
		}

		if len(detailsJSON) > 0 {
			_ = json.Unmarshal(detailsJSON, &gh.Details)
		}

		result = append(result, &gh)
	}

	return result, nil
}
