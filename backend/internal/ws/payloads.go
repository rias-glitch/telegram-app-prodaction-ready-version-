package ws

//Клиент обращается к серверу
type MovePayload struct {
	Move string `json:"move"` // камень / ножницы / бумага
}

//Сервер отдает ответ клиенту
type MatchFoundPayload struct {
	RoomID     string `json:"room_id"`
	OpponentID int64  `json:"opponent_id"`
}

type ResultPayload struct {
	Winner int64            `json:"winner"`
	Moves  map[int64]string `json:"moves"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}
