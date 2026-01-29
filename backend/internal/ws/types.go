package ws

const (
	// клиент к серверу
	MsgMove = "move"
	MsgPing = "ping"

	// сервер к клиенту
	MsgMatchFound = "match_found"
	MsgResult     = "result"
	MsgError      = "error"
)
