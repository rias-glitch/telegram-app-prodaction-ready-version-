package game

import "time"

type GameType string

const (
	TypeRPS   GameType = "rps"
	TypeMines GameType = "mines"
)

type Game interface {
	Type() GameType
	Players() [2]int64

	// обновляет ID второго игрока (используется при сопоставлении)
	SetSecondPlayer(playerID int64)

	// фаза настройки (опционально для некоторых игр)
	SetupTimeout() time.Duration
	HandleSetup(playerID int64, data interface{}) error
	IsSetupComplete() bool

	// фаза игры
	TurnTimeout() time.Duration
	HandleMove(playerID int64, data interface{}) error
	IsRoundComplete() bool

	// проверка результата
	CheckResult() *GameResult
	IsFinished() bool

	// сериализация для клиента
	SerializeState(playerID int64) interface{}
}

type GameResult struct {
	WinnerID  *int64
	Reason    string
	Details   map[string]interface{}
}