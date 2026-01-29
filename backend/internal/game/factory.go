package game

import "fmt"

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) CreateGame(gameType GameType, roomID string, players [2]int64) (Game, error) {
	switch gameType {
	case TypeRPS:
		return NewRPSGame(roomID, players), nil
	case TypeMines:
		return NewMinesGame(roomID, players), nil
	default:
		return nil, fmt.Errorf("unknown game type: %s", gameType)
	}
}