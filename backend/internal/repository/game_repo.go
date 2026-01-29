package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"telegram_webapp/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type GameRepository struct {
    db *pgxpool.Pool
}

func NewGameRepository(db *pgxpool.Pool) *GameRepository {
    return &GameRepository{db: db}
}

func (r *GameRepository) Create(ctx context.Context, g *domain.Game) error {
    movesJSON, err := json.Marshal(g.Moves)
    if err != nil {
        return err
    }

    var createdAt time.Time
    var id int64
    err = r.db.QueryRow(ctx,
        `INSERT INTO games (room_id, player_a_id, player_b_id, moves, winner_id)
         VALUES ($1, $2, $3, $4, $5)
         RETURNING id, created_at`,
        g.RoomID,
        g.PlayerAID,
        g.PlayerBID,
        movesJSON,
        g.WinnerID,
    ).Scan(&id, &createdAt)
    if err != nil {
        return err
    }

    g.ID = id
    g.CreatedAt = createdAt
    return nil
}

func (r *GameRepository) GetByUser(ctx context.Context, userID int64) ([]*domain.Game, error) {
    rows, err := r.db.Query(ctx,
        `SELECT id, room_id, player_a_id, player_b_id, moves, winner_id, created_at
         FROM games
         WHERE player_a_id = $1 OR player_b_id = $1
         ORDER BY created_at DESC
         LIMIT 100`,
        userID,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var res []*domain.Game
    for rows.Next() {
        var (
            id int64
            roomID string
            a int64
            b int64
            movesBytes []byte
            winnerPtr *int64
            createdAt time.Time
        )

        if err := rows.Scan(&id, &roomID, &a, &b, &movesBytes, &winnerPtr, &createdAt); err != nil {
            return nil, err
        }

        //  преобразует ходы (JSON-объект с string keys)
        var movesMap map[string]string
        _ = json.Unmarshal(movesBytes, &movesMap)

        moves := make(map[int64]string)
        for ks, v := range movesMap {
            // попытка распарсить ключ
            var k int64
            fmt.Sscan(ks, &k)
            moves[k] = v
        }



        g := &domain.Game{
            ID: id,
            RoomID: roomID,
            PlayerAID: a,
            PlayerBID: b,
            Moves: moves,
            WinnerID: winnerPtr,
            CreatedAt: createdAt,
        }
        res = append(res, g)
    }

    return res, nil
}


