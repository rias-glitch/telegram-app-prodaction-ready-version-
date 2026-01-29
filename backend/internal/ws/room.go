package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"telegram_webapp/internal/domain"
	"telegram_webapp/internal/game"
	"telegram_webapp/internal/repository"
)

const (
	// ставка обоих игроков отдается победителю
	winnerPayoutMultiplier = 2
)
const (
	StateWaiting  = "waiting"
	StatePlaying  = "playing"
	StateFinished = "finished"
)


type Room struct {
	ID      string
	Clients map[int64]*Client

	Register   chan *Client
	Disconnect chan *Client

	mu             sync.RWMutex
	timer          *time.Timer
	timerRound     int  // номер раунда для которого создан таймер
	createdAt      time.Time
	roundStarted   bool // prevents multiple startRound calls for same round
	setupCompleted bool // prevents multiple completeSetup calls
	roundChecking  bool // prevents concurrent checkRound calls
	setupDoneChan  chan struct{} // канал для сигнализации о завершении setup

	game            game.Game // ← игра через интерфейс
	GameRepo        *repository.GameRepository
	GameHistoryRepo *repository.GameHistoryRepository
	hub             *Hub // ← ссылка на Hub для cleanup

	// получение info
	BetAmount int64
	Currency  string // "gems" or "coins"
	UserRepo  *repository.UserRepository
	betPaid   bool // отслеживание выплаты ставки
}
func NewRoom(id string, g game.Game, hub *Hub) *Room {
	return &Room{
		ID:        id,
		Clients:   make(map[int64]*Client),
		Register:  make(chan *Client, 2),
		Disconnect: make(chan *Client, 2),
		createdAt: time.Now(),
		game:      g,
		hub:       hub,
	}
}

func NewRoomWithRepo(id string, g game.Game, gameRepo *repository.GameRepository, gameHistoryRepo *repository.GameHistoryRepository, hub *Hub) *Room {
	r := NewRoom(id, g, hub)
	r.GameRepo = gameRepo
	r.GameHistoryRepo = gameHistoryRepo
	return r
}



func (r *Room) Run() {
	log.Printf("Room.Run: starting room=%s", r.ID)

	setupDone := make(chan struct{})

	// Фаза настройки (если нужна для игры)
	if r.game.SetupTimeout() > 0 {
		log.Printf("Room.Run: room=%s has setup phase", r.ID)

		go func() {
			timer := time.NewTimer(r.game.SetupTimeout())
			defer timer.Stop()

			select {
			case <-timer.C:
				log.Printf("Room.Run: room=%s setup timeout", r.ID)

				// Check if second player joined - if not, cancel the game
				players := r.game.Players()
				if players[1] == 0 {
					log.Printf("Room.Run: room=%s no opponent found, cancelling game", r.ID)
					r.cancelGameNoOpponent()
					close(setupDone)
					return
				}

				// Check if setup was already completed (game already started)
				r.mu.RLock()
				alreadyCompleted := r.setupCompleted
				r.mu.RUnlock()
				if alreadyCompleted {
					log.Printf("Room.Run: room=%s setup already completed, skipping", r.ID)
					close(setupDone)
					return
				}

				r.completeSetup()
				// Запускаем первый раунд после таймаута setup
				time.Sleep(100 * time.Millisecond)
				r.mu.Lock()
				r.roundStarted = false
				r.mu.Unlock()
				log.Printf("Room.Run: starting round after setup timeout in room=%s", r.ID)
				r.startRound()
				close(setupDone)
			case <-setupDone:
				log.Printf("Room.Run: room=%s setup completed manually", r.ID)
			}
		}()
	} else {
		close(setupDone)
	}

	// Сохраняем канал setupDone для закрытия при ручном завершении
	r.mu.Lock()
	r.setupDoneChan = setupDone
	r.mu.Unlock()

	// Обработка событий
	for {
		// ПРОВЕРКА ЗАВЕРШИЛАСЬ ЛИ ИГРА ПРЕЖДЕ ЧЕМ БЛОКИРОВАТЬ ВЫБОР
		if r.game.IsFinished() {
			log.Printf("Room.Run: room=%s game finished, exiting", r.ID)
			r.saveResult()
			r.cleanup()
			return
		}

		select {
		case c := <-r.Register:
			log.Printf("Room.Run: room=%s received Register for user=%d", r.ID, c.UserID)
			r.handleRegister(c)

			// Если setup завершён и оба игрока подключены
			r.mu.RLock()
			clientsCount := len(r.Clients)
			r.mu.RUnlock()

			if r.game.IsSetupComplete() && clientsCount == 2 {
				// Mark setup as completed to prevent setup timer from interfering
				r.mu.Lock()
				r.setupCompleted = true
				r.mu.Unlock()

				// ДОЖДАТЬСЯ готовности обоих(с таймаутом небольшим)
				r.mu.RLock()
				clientsCopy := make([]*Client, 0, len(r.Clients))
				for _, cl := range r.Clients {
					clientsCopy = append(clientsCopy, cl)
				}
				r.mu.RUnlock()

				for _, cl := range clientsCopy {
					select {
					case <-cl.Ready:
						log.Printf("Room.Run: client %d Ready in room=%s", cl.UserID, r.ID)
					case <-time.After(1 * time.Second):
						log.Printf("Room.Run: timeout waiting for client %d Ready in room=%s", cl.UserID, r.ID)
					}
				}

				log.Printf("Room.Run: starting round in room=%s", r.ID)
				r.startRound()
			}

		case c := <-r.Disconnect:
			log.Printf("Room.Run: room=%s received Disconnect for user=%d", r.ID, c.UserID)
			shouldTerminate := r.handleDisconnect(c)

			// Если handleDisconnect вернул true - комната уже очищена
			if shouldTerminate {
				log.Printf("Room.Run: room=%s terminated after disconnect", r.ID)
				return
			}

		case <-time.After(100 * time.Millisecond):
			// Периодическая проверка состояния IsFinished (в начале цикла)
			continue
		}
	}
}

func (r *Room) completeSetup() {
	r.mu.Lock()
	// Prevent multiple completeSetup calls
	if r.setupCompleted {
		r.mu.Unlock()
		log.Printf("Room.completeSetup: already completed in room=%s, skipping", r.ID)
		return
	}
	r.setupCompleted = true

	// Для каждого игрока который не завершил setup - вызываем HandleMove с nil (бот сделает)
	for _, playerID := range r.game.Players() {
		if !r.game.IsSetupComplete() {
			r.game.HandleMove(playerID, nil)
		}
	}
	// Собираем клиентов, удерживая блокировку
	clients := r.getClientsUnlocked()
	r.mu.Unlock()

	// Отправка без блокировки
	r.broadcastToClients(clients, Message{Type: "setup_complete"})
}

func (r *Room) startRound() {
	r.mu.Lock()
	// Prevent multiple startRound calls for the same round
	if r.roundStarted {
		r.mu.Unlock()
		log.Printf("Room.startRound: round already started in room=%s, skipping", r.ID)
		return
	}
	r.roundStarted = true

	// Собираем клиентов, удерживая блокировку
	clients := r.getClientsUnlocked()

	// Настройка таймера с номером раунда
	if r.timer != nil {
		r.timer.Stop()
	}
	r.timerRound++
	currentRound := r.timerRound
	r.timer = time.AfterFunc(r.game.TurnTimeout(), func() {
		r.handleRoundTimeout(currentRound)
	})
	r.mu.Unlock()

	// Отправляем start с меткой времени, чтобы фронтенд обнаружил новый раунд
	log.Printf("Room.startRound: sending start message to %d clients, timerRound=%d", len(clients), currentRound)
	r.broadcastToClients(clients, Message{
		Type: "start",
		Payload: map[string]any{
			"type":      "start",
			"timestamp": time.Now().UnixMilli(),
		},
	})
}

func (r *Room) handleRoundTimeout(forRound int) {
	r.mu.Lock()
	// Проверяем что таймер для актуального раунда (защита от race condition с timer.Stop)
	if forRound != r.timerRound {
		r.mu.Unlock()
		log.Printf("Room.handleRoundTimeout: stale timer forRound=%d, currentRound=%d, skipping", forRound, r.timerRound)
		return
	}

	log.Printf("Room.handleRoundTimeout: processing timeout for round=%d in room=%s", forRound, r.ID)

	// Для каждого игрока который не сходил - бот делает ход
	for _, playerID := range r.game.Players() {
		r.game.HandleMove(playerID, nil)
	}
	isComplete := r.game.IsRoundComplete()
	r.mu.Unlock()

	if isComplete {
		r.checkRound()
	}
}

// getClientsUnlocked возвращает копию map клиентов - вызывающий должен удерживать блокировку
func (r *Room) getClientsUnlocked() map[int64]*Client {
	clients := make(map[int64]*Client, len(r.Clients))
	for k, v := range r.Clients {
		clients[k] = v
	}
	return clients
}

// broadcastToClients отправляет сообщение всем клиентам без взятия блокировки
func (r *Room) broadcastToClients(clients map[int64]*Client, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Room.broadcastToClients: marshal error: %v", err)
		return
	}

	for userID, c := range clients {
		if c == nil {
			continue
		}
		select {
		case c.Send <- data:
			log.Printf("Room.broadcastToClients: ✅ sent to user=%d type=%s", userID, msg.Type)
		case <-time.After(2 * time.Second):
			log.Printf("Room.broadcastToClients: ❌ timeout sending to user=%d type=%s", userID, msg.Type)
		}
	}
}

func (r *Room) checkRound() {
	log.Printf("Room.checkRound: ENTER room=%s", r.ID)

	// Prevent concurrent checkRound calls - must check BEFORE IsRoundComplete to avoid race
	r.mu.Lock()
	if r.roundChecking {
		r.mu.Unlock()
		log.Printf("Room.checkRound: already checking round in room=%s, skipping", r.ID)
		return
	}
	r.roundChecking = true
	r.mu.Unlock()

	// Reset flag when done
	defer func() {
		r.mu.Lock()
		r.roundChecking = false
		r.mu.Unlock()
		log.Printf("Room.checkRound: EXIT room=%s", r.ID)
	}()

	// Don't check result if round is not complete
	if !r.game.IsRoundComplete() {
		log.Printf("Room.checkRound: round not complete yet in room=%s", r.ID)
		return
	}

	// debug: log players and connected clients
	r.mu.RLock()
	clientIDs := make([]int64, 0, len(r.Clients))
	for uid := range r.Clients {
		clientIDs = append(clientIDs, uid)
	}
	r.mu.RUnlock()
	log.Printf("Room.checkRound: room=%s players=%v clients=%v", r.ID, r.game.Players(), clientIDs)

	result := r.game.CheckResult()
	log.Printf("Room.checkRound: room=%s check result=%v finished=%v", r.ID, result, r.game.IsFinished())

	if result == nil {
		// Round was a draw or both players had same outcome - continue to next round
		if !r.game.IsFinished() {
			log.Printf("Room.checkRound: round draw, starting next round in room %s", r.ID)

			// For Mines game, send round results to each player
			// This already triggers state update on frontend
			r.sendMinesRoundResult()

			// Send draw notification only for non-Mines games
			// For Mines, round_result + start is enough
			if r.game.Type() != game.TypeMines {
				r.mu.RLock()
				clients := make(map[int64]*Client, len(r.Clients))
				for k, v := range r.Clients {
					clients[k] = v
				}
				r.mu.RUnlock()

				// For RPS, send moves with draw notification
				if r.game.Type() == game.TypeRPS {
					rpsGame, ok := r.game.(*game.RPSGame)
					if ok {
						lastMoves := rpsGame.GetLastMoves()
						players := r.game.Players()
						p1, p2 := players[0], players[1]

						// Send personalized draw to each player
						for uid, c := range clients {
							var yourMove, opponentMove string
							if uid == p1 {
								yourMove = lastMoves[p1]
								opponentMove = lastMoves[p2]
							} else {
								yourMove = lastMoves[p2]
								opponentMove = lastMoves[p1]
							}

							data, _ := json.Marshal(Message{
								Type: "round_draw",
								Payload: map[string]any{
									"message":       "Round ended in draw, starting next round",
									"round":         rpsGame.GetRound(),
									"your_move":     yourMove,
									"opponent_move": opponentMove,
								},
							})
							select {
							case c.Send <- data:
							case <-time.After(1 * time.Second):
							}
						}
					}
				} else {
					r.broadcastToClients(clients, Message{
						Type: "round_draw",
						Payload: map[string]any{
							"message": "Round ended in draw, starting next round",
						},
					})
				}
			}

			// Reset roundStarted to allow next round
			r.mu.Lock()
			r.roundStarted = false
			r.mu.Unlock()
			r.startRound()
		}
		return
	}

	// For Mines game, send final round result before game result
	r.sendMinesRoundResult()

	// Отправляем результат (this function handles its own locking)
	r.broadcastResult(result)

	if r.game.IsFinished() {
		// Игра полностью закончена
		log.Printf("Room.checkRound: game finished in room %s", r.ID)
		return
	}

	// Игра продолжается - следующий раунд
	log.Printf("Room.checkRound: starting next round in room %s", r.ID)
	// Reset roundStarted to allow next round
	r.mu.Lock()
	r.roundStarted = false
	r.mu.Unlock()
	r.startRound()
}

func (r *Room) cleanup() {
	// Collect data while holding room lock
	r.mu.Lock()
	if r.timer != nil {
		r.timer.Stop()
		r.timer = nil
	}
	players := r.game.Players()
	gameType := r.game.Type()
	clientIDs := make([]int64, 0, len(r.Clients))
	for uid := range r.Clients {
		clientIDs = append(clientIDs, uid)
	}
	hub := r.hub
	roomID := r.ID
	r.mu.Unlock()

	// Now update hub WITHOUT holding room lock (prevents deadlock)
	if hub != nil {
		hub.mu.Lock()
		delete(hub.Rooms, roomID)

		// Clear UserRoom for all players (from game, not just connected clients)
		for _, uid := range players {
			if uid != 0 {
				delete(hub.UserRoom, uid)
			}
		}

		// Also clear UserRoom for any connected clients (safety)
		for _, uid := range clientIDs {
			delete(hub.UserRoom, uid)
		}

		// Clear WaitingByGame if any player from this room was waiting
		if waiting := hub.WaitingByGame[gameType]; waiting != nil {
			for _, uid := range players {
				if waiting.UserID == uid {
					log.Printf("Room.cleanup: clearing stale waiting slot for user=%d game=%s", uid, gameType)
					delete(hub.WaitingByGame, gameType)
					break
				}
			}
		}

		hub.mu.Unlock()
	}

	log.Printf("Room.cleanup: room=%s cleaned up", roomID)
}

func (r *Room) handleRegister(c *Client) {
	r.mu.Lock()

	r.Clients[c.UserID] = c

	log.Printf("Room.handleRegister: room=%s user=%d players=%d game_type=%s", r.ID, c.UserID, len(r.Clients), r.game.Type())

	// Check if the client's writePump has started; do not block here
	if c != nil {
		select {
		case <-c.Ready:
			log.Printf("Room.handleRegister: client %d already ready in room=%s", c.UserID, r.ID)
		default:
			log.Printf("Room.handleRegister: client %d not ready yet in room=%s", c.UserID, r.ID)
		}
	}

	// acknowledge registration to the handler by closing the channel (idempotent)
	if c != nil && c.Registered != nil {
		// close to signal registration (safe because handleRegister called once per client)
		close(c.Registered)
		log.Printf("Room.handleRegister: closed Registered for user=%d room=%s", c.UserID, r.ID)
	}



	if len(r.Clients) == 2 {
		log.Printf("Room.handleRegister: room=%s BOTH PLAYERS REGISTERED; will send matched messages", r.ID)

		// Collect data while holding lock
		players := r.game.Players()
		p1, p2 := players[0], players[1]
		c1 := r.Clients[p1]
		c2 := r.Clients[p2]
		userRepo := r.UserRepo

		// Release lock before sending to avoid deadlock
		r.mu.Unlock()

		// Load user info for opponents
		var user1Info, user2Info map[string]any
		if userRepo != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			if u1, err := userRepo.GetByID(ctx, p1); err == nil {
				user1Info = map[string]any{
					"id":         p1,
					"first_name": u1.FirstName,
					"username":   u1.Username,
				}
			}
			if u2, err := userRepo.GetByID(ctx, p2); err == nil {
				user2Info = map[string]any{
					"id":         p2,
					"first_name": u2.FirstName,
					"username":   u2.Username,
				}
			}
			cancel()
		}
		// Fallback if user info not loaded
		if user1Info == nil {
			user1Info = map[string]any{"id": p1}
		}
		if user2Info == nil {
			user2Info = map[string]any{"id": p2}
		}

		// Send matched to both players
		if c1 != nil {
			data1, _ := json.Marshal(Message{
				Type: "matched",
				Payload: map[string]any{
					"room_id":  r.ID,
					"opponent": user2Info,
				},
			})
			select {
			case c1.Send <- data1:
				log.Printf("Room.handleRegister: sent matched to p1=%d", p1)
			case <-time.After(1 * time.Second):
				log.Printf("Room.handleRegister: timeout sending matched to p1=%d", p1)
			}
		}

		if c2 != nil {
			data2, _ := json.Marshal(Message{
				Type: "matched",
				Payload: map[string]any{
					"room_id":  r.ID,
					"opponent": user1Info,
				},
			})
			select {
			case c2.Send <- data2:
				log.Printf("Room.handleRegister: sent matched to p2=%d", p2)
			case <-time.After(1 * time.Second):
				log.Printf("Room.handleRegister: timeout sending matched to p2=%d", p2)
			}
		}

		// Re-acquire lock
		r.mu.Lock()
	} else {
		log.Printf("Room.handleRegister: room=%s waiting for second player (have %d)", r.ID, len(r.Clients))
	}

	// drain any pending messages that the client sent before registration
	c.pendingMu.Lock()
	pending := c.pending
	c.pending = nil
	c.pendingMu.Unlock()

	log.Printf("Room.handleRegister: room=%s user=%d pending_count=%d", r.ID, c.UserID, len(pending))

	// release room lock before processing pending messages to avoid deadlocks
	r.mu.Unlock()

	// send state now that lock is released
	r.send(c.UserID, Message{
		Type: "state",
		Payload: map[string]any{
			"room_id":   r.ID,
			"players":   len(r.Clients),
			"game_type": string(r.game.Type()),
		},
	})

	for i, m := range pending {
		log.Printf("Room.handleRegister: replaying pending[%d] for user=%d: %s", i, c.UserID, string(m))
		r.HandleMessage(c, m)
	}

	// after replaying pending messages, try to check the round a few times synchronously
	for i := 0; i < 5; i++ {
		time.Sleep(20 * time.Millisecond)
		r.checkRound()
		if r.game.IsFinished() {
			break
		}
	}
}

// handleDisconnect handles client disconnection.
// Returns true if room should be terminated (either empty or winner declared).
func (r *Room) handleDisconnect(c *Client) bool {
	r.mu.Lock()

	delete(r.Clients, c.UserID)

	log.Printf("Room.handleDisconnect: room=%s user=%d bet=%d currency=%s", r.ID, c.UserID, r.BetAmount, r.Currency)

	// Collect remaining client info while holding lock
	var remainingUID int64
	var remainingClient *Client
	shouldNotifyWinner := len(r.Clients) == 1

	// Check if game had 2 players (second player joined)
	players := r.game.Players()
	hadTwoPlayers := players[1] != 0

	if shouldNotifyWinner {
		for uid, cl := range r.Clients {
			remainingUID = uid
			remainingClient = cl
			break
		}
	}

	clientsLeft := len(r.Clients)

	// Handle bet payouts
	shouldPayWinner := r.BetAmount > 0 && !r.betPaid && hadTwoPlayers && shouldNotifyWinner
	shouldRefundDisconnecting := r.BetAmount > 0 && !r.betPaid && !hadTwoPlayers // Game never started (waiting for opponent)
	if shouldPayWinner || shouldRefundDisconnecting {
		r.betPaid = true
	}
	r.mu.Unlock()

	// Handle bet payouts outside of lock
	if shouldPayWinner && remainingClient != nil {
		// Winner gets both bets (opponent forfeited)
		log.Printf("Room.handleDisconnect: opponent left, paying winner=%d pot=%d %s",
			remainingUID, r.BetAmount*2, r.Currency)
		r.payoutWinner(&remainingUID, remainingUID, c.UserID)
	} else if shouldRefundDisconnecting {
		// Game never started, refund disconnecting player
		log.Printf("Room.handleDisconnect: game never started, refunding user=%d", c.UserID)
		r.refundBet(c.UserID)
	}

	// Send win notification without holding lock (avoids deadlock with r.send)
	if shouldNotifyWinner && remainingClient != nil {
		winAmount := r.BetAmount * 2
		data, _ := json.Marshal(Message{
			Type: "result",
			Payload: map[string]any{
				"you":        "win",
				"reason":     "opponent_left",
				"win_amount": winAmount,
				"currency":   r.Currency,
			},
		})
		select {
		case remainingClient.Send <- data:
			log.Printf("Room.handleDisconnect: sent win to user=%d", remainingUID)
		case <-time.After(2 * time.Second):
			log.Printf("Room.handleDisconnect: timeout sending win to user=%d", remainingUID)
		}

		// Cleanup without holding room lock (cleanup takes its own lock)
		r.cleanup()
		return true // Room should terminate
	}

	// If room is empty, also cleanup
	if clientsLeft == 0 {
		r.cleanup()
		return true // Room should terminate
	}

	return false // Room continues (still has 2 players somehow, or waiting for second)
}

func (r *Room) HandleMessage(c *Client, raw []byte) {
	var msg struct {
		Type  string      `json:"type"`
		Value interface{} `json:"value"`
	}

	if err := json.Unmarshal(raw, &msg); err != nil {
		log.Printf("Room.HandleMessage: failed to unmarshal: %v", err)
		return
	}

	log.Printf("Room.HandleMessage: room=%s user=%d type=%s value=%v valueType=%T raw=%s", r.ID, c.UserID, msg.Type, msg.Value, msg.Value, string(raw))

	// Convert value to appropriate type for the game
	var moveValue interface{} = msg.Value

	// For RPS, value should be a string
	// For Mines setup, value should be []int
	// For Mines move, value should be int
	if r.game.Type() == "mines" {
		// Handle setup (array of mine positions)
		if arr, ok := msg.Value.([]interface{}); ok {
			intArr := make([]int, len(arr))
			for i, v := range arr {
				if num, ok := v.(float64); ok {
					intArr[i] = int(num)
				}
			}
			moveValue = intArr
			log.Printf("Room.HandleMessage: converted mines setup value to []int: %v", intArr)
		}
		// Handle move (single cell number)
		if num, ok := msg.Value.(float64); ok {
			moveValue = int(num)
			log.Printf("Room.HandleMessage: converted mines move value to int: %d", int(num))
		} else {
			log.Printf("Room.HandleMessage: mines move value NOT float64, type=%T value=%v", msg.Value, msg.Value)
		}
	}

	// Обрабатываем ход через игру
	if err := r.game.HandleMove(c.UserID, moveValue); err != nil {
		log.Printf("Room.HandleMessage: invalid move from user=%d: %v", c.UserID, err)
		r.send(c.UserID, Message{
			Type: "error",
			Payload: map[string]string{"message": err.Error()},
		})
		return
	}

	// Проверяем завершение setup фазы
	r.mu.Lock()
	setupWasCompleted := r.setupCompleted
	r.mu.Unlock()

	if !setupWasCompleted && r.game.IsSetupComplete() {
		log.Printf("Room.HandleMessage: setup complete after move in room=%s, transitioning to gameplay", r.ID)
		r.completeSetup()
		// Закрываем канал setupDone чтобы остановить timeout goroutine
		r.mu.Lock()
		if r.setupDoneChan != nil {
			select {
			case <-r.setupDoneChan:
				// Канал уже закрыт
			default:
				close(r.setupDoneChan)
			}
		}
		r.mu.Unlock()
		// Небольшая пауза для завершения отправки setup_complete
		time.Sleep(100 * time.Millisecond)
		// Запускаем первый раунд
		r.mu.Lock()
		r.roundStarted = false // сбрасываем флаг чтобы startRound сработал
		r.mu.Unlock()
		log.Printf("Room.HandleMessage: calling startRound in room=%s", r.ID)
		r.startRound()
		return
	}

	// Если раунд завершён - проверяем результат
	isComplete := r.game.IsRoundComplete()
	log.Printf("Room.HandleMessage: after move, isComplete=%v room=%s user=%d", isComplete, r.ID, c.UserID)

	if isComplete {
		log.Printf("Room.HandleMessage: round complete in room=%s, stopping timer and calling checkRound", r.ID)

		r.mu.Lock()
		if r.timer != nil {
			r.timer.Stop()
			r.timer = nil
		}
		r.mu.Unlock()

		// Один вызов checkRound - он сам обработает результат
		r.checkRound()
	} else {
		log.Printf("Room.HandleMessage: waiting for other player in room=%s", r.ID)
	}
}

func (r *Room) broadcastResult(result *game.GameResult) {
	r.mu.RLock()
	players := r.game.Players()
	p1, p2 := players[0], players[1]
	c1 := r.Clients[p1]
	c2 := r.Clients[p2]
	r.mu.RUnlock()

	log.Printf("Room.broadcastResult: room=%s winner=%v", r.ID, result.WinnerID)

	// Определяем результат для каждого игрока
	var result1, result2 string

	if result.WinnerID == nil {
		result1 = "draw"
		result2 = "draw"
	} else if *result.WinnerID == p1 {
		result1 = "win"
		result2 = "lose"
	} else {
		result1 = "lose"
		result2 = "win"
	}

	log.Printf("Room.broadcastResult: sending results - p1=%d=%s p2=%d=%s", p1, result1, p2, result2)

	// Персонализируем details для каждого игрока (для RPS)
	details1 := result.Details
	details2 := result.Details

	if r.game.Type() == game.TypeRPS && result.Details != nil {
		// Получаем ходы из moves map
		if moves, ok := result.Details["moves"].(map[int64]string); ok {
			move1 := moves[p1]
			move2 := moves[p2]

			// Details для игрока 1: его ход = yourMove, ход противника = opponentMove
			details1 = map[string]interface{}{
				"moves":        moves,
				"yourMove":     move1,
				"opponentMove": move2,
			}

			// Details для игрока 2: его ход = yourMove, ход противника = opponentMove
			details2 = map[string]interface{}{
				"moves":        moves,
				"yourMove":     move2,
				"opponentMove": move1,
			}

			log.Printf("Room.broadcastResult: RPS personalized - p1 sees your=%s opp=%s, p2 sees your=%s opp=%s",
				move1, move2, move2, move1)
		} else {
			log.Printf("Room.broadcastResult: RPS moves type assertion failed, details=%+v", result.Details)
		}
	}

	// Send to player 1
	if c1 != nil {
		data1, _ := json.Marshal(Message{
			Type: "result",
			Payload: map[string]any{
				"you":     result1,
				"reason":  result.Reason,
				"details": details1,
			},
		})
		select {
		case c1.Send <- data1:
			log.Printf("Room.broadcastResult: ✅ sent result to p1=%d", p1)
		case <-time.After(2 * time.Second):
			log.Printf("Room.broadcastResult: ❌ timeout sending result to p1=%d", p1)
		}
	} else {
		log.Printf("Room.broadcastResult: ❌ p1=%d client is nil", p1)
	}

	// Send to player 2
	if c2 != nil {
		data2, _ := json.Marshal(Message{
			Type: "result",
			Payload: map[string]any{
				"you":     result2,
				"reason":  result.Reason,
				"details": details2,
			},
		})
		select {
		case c2.Send <- data2:
			log.Printf("Room.broadcastResult: ✅ sent result to p2=%d", p2)
		case <-time.After(2 * time.Second):
			log.Printf("Room.broadcastResult: ❌ timeout sending result to p2=%d", p2)
		}
	} else {
		log.Printf("Room.broadcastResult: ❌ p2=%d client is nil", p2)
	}
}


func (r *Room) send(userID int64, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Room.send: marshal error: %v", err)
		return
	}

	r.mu.RLock()
	c, ok := r.Clients[userID]
	r.mu.RUnlock()

	if ok {
		// blocking send with generous timeout to improve reliability in tests
		select {
		case c.Send <- data:
			log.Printf("Room.send: ✅ sent to user=%d type=%s", userID, msg.Type)
		case <-time.After(2 * time.Second):
			log.Printf("Room.send: ❌ timeout sending to user=%d type=%s", userID, msg.Type)
		}
	} else {
		log.Printf("Room.send: ❌ user=%d not in room", userID)
	}

	// if this was a result message, wait for client writePump ack
	if ok && msg.Type == "result" && c != nil && c.ResultAck != nil {
		select {
		case <-c.ResultAck:
			log.Printf("Room.send: delivery ack received for user=%d type=%s", userID, msg.Type)
		case <-time.After(2 * time.Second):
			log.Printf("Room.send: delivery ack TIMEOUT for user=%d type=%s", userID, msg.Type)
		}
	}
}

func (r *Room) broadcast(msg Message) {
	for _, c := range r.Clients {
		// use blocking send to ensure clients receive broadcast (with timeout)
		r.send(c.UserID, msg)
	}
}

// sendMinesRoundResult sends round result to each player for Mines game
func (r *Room) sendMinesRoundResult() {
	if r.game.Type() != game.TypeMines {
		return
	}

	minesGame, ok := r.game.(*game.MinesGame)
	if !ok {
		return
	}

	roundResult := minesGame.GetLastRoundResult()
	if roundResult == nil {
		return
	}

	players := r.game.Players()
	p1, p2 := players[0], players[1]

	r.mu.RLock()
	c1 := r.Clients[p1]
	c2 := r.Clients[p2]
	r.mu.RUnlock()

	nextRound := minesGame.GetRound() + 1

	// Send to player 1 - their move result and history
	if c1 != nil {
		myMove := roundResult.PlayerMoves[p1]
		oppMove := roundResult.PlayerMoves[p2]
		r.send(p1, Message{
			Type: "round_result",
			Payload: map[string]any{
				"round":         roundResult.Round,
				"next_round":    nextRound,
				"your_move":     myMove.Cell,
				"your_hit":      myMove.HitMine,
				"opponent_move": oppMove.Cell,
				"opponent_hit":  oppMove.HitMine,
				"history":       minesGame.GetMoveHistory(p1),
				"timestamp":     time.Now().UnixMilli(),
			},
		})
	}

	// Send to player 2 - their move result and history
	if c2 != nil {
		myMove := roundResult.PlayerMoves[p2]
		oppMove := roundResult.PlayerMoves[p1]
		r.send(p2, Message{
			Type: "round_result",
			Payload: map[string]any{
				"round":         roundResult.Round,
				"next_round":    nextRound,
				"your_move":     myMove.Cell,
				"your_hit":      myMove.HitMine,
				"opponent_move": oppMove.Cell,
				"opponent_hit":  oppMove.HitMine,
				"history":       minesGame.GetMoveHistory(p2),
				"timestamp":     time.Now().UnixMilli(),
			},
		})
	}

	log.Printf("Room.sendMinesRoundResult: sent round %d results to players", roundResult.Round)
}

func (r *Room) saveResult() {
	result := r.game.CheckResult()
	if result == nil {
		return
	}

	players := r.game.Players()
	p1, p2 := players[0], players[1]

	log.Printf("Room.saveResult: room=%s storing game bet=%d currency=%s", r.ID, r.BetAmount, r.Currency)

	// Pay out the winner (if there's a bet and it hasn't been paid yet)
	r.mu.Lock()
	shouldPay := r.BetAmount > 0 && !r.betPaid
	if shouldPay {
		r.betPaid = true
	}
	r.mu.Unlock()

	if shouldPay {
		r.payoutWinner(result.WinnerID, p1, p2)
	}

	// Calculate win amounts for history
	var winAmount1, winAmount2 int64
	if result.WinnerID != nil {
		if *result.WinnerID == p1 {
			winAmount1 = r.BetAmount * winnerPayoutMultiplier
		} else {
			winAmount2 = r.BetAmount * winnerPayoutMultiplier
		}
	} else {
		// Draw - both get refunded (handled in payoutWinner)
		winAmount1 = r.BetAmount
		winAmount2 = r.BetAmount
	}

	// Save to old games table (for backwards compatibility)
	if r.GameRepo != nil {
		g := &domain.Game{
			RoomID:    r.ID,
			PlayerAID: p1,
			PlayerBID: p2,
			Moves:     make(map[int64]string),
			WinnerID:  result.WinnerID,
		}
		go func(game *domain.Game) {
			if err := r.GameRepo.Create(context.Background(), game); err != nil {
				log.Printf("Room.saveResult: game store failed: %v", err)
			}
		}(g)
	}

	// Save to new game_history table
	if r.GameHistoryRepo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		gameType := string(r.game.Type())
		details := result.Details

		// Determine results for each player
		var result1, result2 domain.GameResult

		if result.WinnerID == nil {
			result1 = domain.GameResultDraw
			result2 = domain.GameResultDraw
		} else if *result.WinnerID == p1 {
			result1 = domain.GameResultWin
			result2 = domain.GameResultLose
		} else {
			result1 = domain.GameResultLose
			result2 = domain.GameResultWin
		}

		currency := domain.Currency(r.Currency)

		// Save for player 1
		gh1 := &domain.GameHistory{
			UserID:     p1,
			GameType:   domain.GameType(gameType),
			Mode:       domain.GameModePVP,
			OpponentID: &p2,
			RoomID:     &r.ID,
			Result:     result1,
			BetAmount:  r.BetAmount,
			WinAmount:  winAmount1,
			Currency:   currency,
			Details:    details,
		}
		go func() {
			defer cancel()
			if err := r.GameHistoryRepo.Create(ctx, gh1); err != nil {
				log.Printf("Room.saveResult: game_history p1 failed: %v", err)
			}
		}()

		// Save for player 2
		gh2 := &domain.GameHistory{
			UserID:     p2,
			GameType:   domain.GameType(gameType),
			Mode:       domain.GameModePVP,
			OpponentID: &p1,
			RoomID:     &r.ID,
			Result:     result2,
			BetAmount:  r.BetAmount,
			WinAmount:  winAmount2,
			Currency:   currency,
			Details:    details,
		}
		go func() {
			if err := r.GameHistoryRepo.Create(ctx, gh2); err != nil {
				log.Printf("Room.saveResult: game_history p2 failed: %v", err)
			}
		}()
	}
}

// payoutWinner pays out the bet to the winner, or refunds both on draw
func (r *Room) payoutWinner(winnerID *int64, p1, p2 int64) {
	if r.UserRepo == nil || r.BetAmount == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	totalPot := r.BetAmount * 2 // Both players bet the same amount

	if winnerID == nil {
		// Draw - refund both players
		log.Printf("Room.payoutWinner: draw in room=%s, refunding both players %d %s", r.ID, r.BetAmount, r.Currency)
		if r.Currency == string(domain.CurrencyCoins) {
			if _, err := r.UserRepo.UpdateCoins(ctx, p1, r.BetAmount); err != nil {
				log.Printf("Room.payoutWinner: failed to refund p1: %v", err)
			}
			if _, err := r.UserRepo.UpdateCoins(ctx, p2, r.BetAmount); err != nil {
				log.Printf("Room.payoutWinner: failed to refund p2: %v", err)
			}
		} else {
			if _, err := r.UserRepo.UpdateGems(ctx, p1, r.BetAmount); err != nil {
				log.Printf("Room.payoutWinner: failed to refund p1: %v", err)
			}
			if _, err := r.UserRepo.UpdateGems(ctx, p2, r.BetAmount); err != nil {
				log.Printf("Room.payoutWinner: failed to refund p2: %v", err)
			}
		}
	} else {
		// Winner gets the entire pot (2x bet)
		log.Printf("Room.payoutWinner: winner=%d in room=%s gets %d %s", *winnerID, r.ID, totalPot, r.Currency)
		if r.Currency == string(domain.CurrencyCoins) {
			if _, err := r.UserRepo.UpdateCoins(ctx, *winnerID, totalPot); err != nil {
				log.Printf("Room.payoutWinner: failed to pay winner: %v", err)
			}
		} else {
			if _, err := r.UserRepo.UpdateGems(ctx, *winnerID, totalPot); err != nil {
				log.Printf("Room.payoutWinner: failed to pay winner: %v", err)
			}
		}
	}
}

// cancelGameNoOpponent cancels the game when no opponent is found, refunds bet and notifies player
func (r *Room) cancelGameNoOpponent() {
	r.mu.Lock()
	players := r.game.Players()
	playerID := players[0]

	// Get the client to notify
	client := r.Clients[playerID]

	// Mark bet as refunded
	shouldRefund := r.BetAmount > 0 && !r.betPaid
	if shouldRefund {
		r.betPaid = true
	}
	r.mu.Unlock()

	// Refund the bet
	if shouldRefund {
		log.Printf("Room.cancelGameNoOpponent: refunding %d %s to user=%d", r.BetAmount, r.Currency, playerID)
		r.refundBet(playerID)
	}

	// Notify player that no opponent was found
	if client != nil {
		data, _ := json.Marshal(Message{
			Type: "result",
			Payload: map[string]any{
				"you":      "cancelled",
				"reason":   "no_opponent",
				"refunded": r.BetAmount,
				"currency": r.Currency,
			},
		})
		select {
		case client.Send <- data:
			log.Printf("Room.cancelGameNoOpponent: notified user=%d", playerID)
		case <-time.After(2 * time.Second):
			log.Printf("Room.cancelGameNoOpponent: timeout notifying user=%d", playerID)
		}
	}

	// Cleanup the room
	r.cleanup()
}

// refundBet refunds a player's bet (used when game is cancelled or player disconnects early)
func (r *Room) refundBet(userID int64) {
	if r.UserRepo == nil || r.BetAmount == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("Room.refundBet: refunding %d %s to user=%d", r.BetAmount, r.Currency, userID)

	if r.Currency == string(domain.CurrencyCoins) {
		if _, err := r.UserRepo.UpdateCoins(ctx, userID, r.BetAmount); err != nil {
			log.Printf("Room.refundBet: failed to refund coins: %v", err)
		}
	} else {
		if _, err := r.UserRepo.UpdateGems(ctx, userID, r.BetAmount); err != nil {
			log.Printf("Room.refundBet: failed to refund gems: %v", err)
		}
	}
}
