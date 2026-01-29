package domain

import "time"

// тип игры
type GameType string

const (
	GameTypeRPS      GameType = "rps"
	GameTypeMines    GameType = "mines"
	GameTypeMinesPro GameType = "mines_pro"
	GameTypeCoinflip GameType = "coinflip"
	GameTypeCase     GameType = "case"
	GameTypeDice     GameType = "dice"
	GameTypeWheel    GameType = "wheel"
)

// режим
type GameMode string

const (
	GameModePVP  GameMode = "pvp"
	GameModePVE  GameMode = "pve"
	GameModeSolo GameMode = "solo"
)

// результат
type GameResult string

const (
	GameResultWin  GameResult = "win"
	GameResultLose GameResult = "lose"
	GameResultDraw GameResult = "draw"
)

// история с записью
type GameHistory struct {
	ID         int64                  `db:"id" json:"id"`
	UserID     int64                  `db:"user_id" json:"user_id"`
	GameType   GameType               `db:"game_type" json:"game_type"`
	Mode       GameMode               `db:"mode" json:"mode"`
	OpponentID *int64                 `db:"opponent_id" json:"opponent_id,omitempty"`
	RoomID     *string                `db:"room_id" json:"room_id,omitempty"`
	Result     GameResult             `db:"result" json:"result"`
	BetAmount  int64                  `db:"bet_amount" json:"bet_amount"`
	WinAmount  int64                  `db:"win_amount" json:"win_amount"`
	Currency   Currency               `db:"currency" json:"currency"` // gems or coins
	Details    map[string]interface{} `db:"details" json:"details,omitempty"`
	CreatedAt  time.Time              `db:"created_at" json:"created_at"`
}

// !!!! ВАЖНО  стараться не изменять т.к веб сокет ругается и очень сильно зависит совместимость от того ,что ниже
type Game struct {
	ID        int64            `db:"id"`
	RoomID    string           `db:"room_id"`
	PlayerAID int64            `db:"player_a_id"`
	PlayerBID int64            `db:"player_b_id"`
	Moves     map[int64]string `db:"moves"`
	WinnerID  *int64           `db:"winner_id"`
	CreatedAt time.Time        `db:"created_at"`
}

// конвертирует старый Game в GameHistory для обоих игроков
func (g *Game) ToGameHistory(gameType GameType) []*GameHistory {
	var result []*GameHistory

	for _, playerID := range []int64{g.PlayerAID, g.PlayerBID} {
		var opponentID int64
		var gameResult GameResult

		if playerID == g.PlayerAID {
			opponentID = g.PlayerBID
		} else {
			opponentID = g.PlayerAID
		}

		if g.WinnerID == nil {
			gameResult = GameResultDraw
		} else if *g.WinnerID == playerID {
			gameResult = GameResultWin
		} else {
			gameResult = GameResultLose
		}

		roomID := g.RoomID
		history := &GameHistory{
			UserID:     playerID,
			GameType:   gameType,
			Mode:       GameModePVP,
			OpponentID: &opponentID,
			RoomID:     &roomID,
			Result:     gameResult,
			Details:    map[string]interface{}{"moves": g.Moves},
			CreatedAt:  g.CreatedAt,
		}
		result = append(result, history)
	}

	return result
}
