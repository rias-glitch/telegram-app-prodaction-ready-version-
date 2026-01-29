package ws

import (
	"bytes"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 30 * time.Second
	pingPeriod = 25 * time.Second
)

type Client struct {
	UserID   int64
	GameType string
	Conn     *websocket.Conn
	Send     chan []byte

	// Получить информацию
	BetAmount int64
	Currency  string // gems или coins

	Hub        *Hub
	Room       *Room
	Ready      chan struct{}
	Registered chan struct{}
	ResultAck  chan struct{}
	Done       chan struct{}
	pendingMu  sync.Mutex
	pending    [][]byte
}

func NewClient(userID int64, conn *websocket.Conn, hub *Hub, gameType string, betAmount int64, currency string) *Client {
	return &Client{
		UserID:     userID,
		GameType:   gameType,
		Conn:       conn,
		Send:       make(chan []byte, 1024),
		BetAmount:  betAmount,
		Currency:   currency,
		Hub:        hub,
		Ready:      make(chan struct{}),
		Registered: make(chan struct{}, 1),
		ResultAck:  make(chan struct{}, 1),
		Done:       make(chan struct{}),
	}
}

func (c *Client) Run() {
	// запускаем writer первым, чтобы регистрация комнаты могла наблюдать готовность
	go c.writePump()
	// сигнализируем, что writePump запущен
	close(c.Ready)

	// отправляем явный хендшейк готовности, чтобы тесты/клиенты могли его дождаться
	readyMsg := []byte(`{"type":"ready"}`)
	select {
	case c.Send <- readyMsg:
		log.Printf("Client.Run: пользователь=%d сообщение о готовности поставлено в очередь", c.UserID)
	case <-time.After(500 * time.Millisecond):
		log.Printf("Client.Run: таймаут постановки в очередь ready для пользователя=%d", c.UserID)
	}

	// запускаем readPump рано, чтобы не пропустить сообщения во время матчмейкинга
	go func() {
		log.Printf("Client.Run: запуск readPump (горутина) для пользователя=%d", c.UserID)
		c.readPump()
	}()

	// назначаем комнату (матчмейкинг / реконнект)
	c.Room = c.Hub.AssignClient(c)

	if c.Room == nil {
		log.Printf("Client.Run: не удалось назначить комнату для пользователя=%d", c.UserID)
		c.Conn.Close()
		return
	}

	log.Printf("Client.Run: пользователь=%d назначен в комнату=%s", c.UserID, c.Room.ID)



	<-c.Done
}

// read
func (c *Client) readPump() {
	log.Printf("Client.readPump: СТАРТ для пользователя=%d", c.UserID)
	defer func() {
		c.disconnect()
		close(c.Done)
	}()

	c.Conn.SetReadLimit(4096)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			log.Println("ошибка чтения:", err)
			break
		}
		log.Printf("Client.readPump: пользователь=%d получил %d байт: %s", c.UserID, len(msg), string(msg))
		if c.Room != nil {
			c.Room.HandleMessage(c, msg)
		} else {
			// буферизуем сообщение до назначения комнаты
			c.pendingMu.Lock()
			c.pending = append(c.pending, append([]byte(nil), msg...))
			c.pendingMu.Unlock()
			log.Printf("Client.readPump: пользователь=%d буферизировал %d байт (еще нет комнаты)", c.UserID, len(msg))
		}
	}
}

// write
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("Client.writePump: пользователь=%d ошибка записи: %v", c.UserID, err)
				return
			}
			log.Printf("Client.writePump: пользователь=%d записал %d байт: %s", c.UserID, len(msg), string(msg))

			// если это было сообщение о результате, подтверждаем его,
			// чтобы сервер мог дождаться доставки
			if bytes.Contains(msg, []byte(`"type":"result"`)) {
				select {
				case c.ResultAck <- struct{}{}:
				default:
				}
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// disconnect
func (c *Client) disconnect() {
	if c.Room != nil {
		c.Hub.OnDisconnect(c)
	}
	_ = c.Conn.Close()
}