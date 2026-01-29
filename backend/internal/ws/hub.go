package ws

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"telegram_webapp/internal/game"
	"telegram_webapp/internal/repository"
)

// уникально идентифицирует очередь матчмейкинга
// игроки сопоставляются по типу игры, сумме ставки и валюте
type WaitingKey struct {
	GameType  game.GameType
	BetAmount int64
	Currency  string
}

func (k WaitingKey) String() string {
	return fmt.Sprintf("%s_%d_%s", k.GameType, k.BetAmount, k.Currency)
}

type Hub struct {
	Rooms    map[string]*Room
	UserRoom map[int64]string
	mu       sync.RWMutex
	roomSeq  int64
	// отдельные очереди ожидания для каждого типа игры + ставки + валюты
	WaitingByKey map[WaitingKey]*Client
	// устаревшее: для обратной совместимости
	WaitingByGame   map[game.GameType]*Client
	GameRepo        *repository.GameRepository
	GameHistoryRepo *repository.GameHistoryRepository
	UserRepo        *repository.UserRepository
}

func NewHub(gameRepo *repository.GameRepository, gameHistoryRepo *repository.GameHistoryRepository) *Hub {
	return &Hub{
		Rooms:           make(map[string]*Room),
		UserRoom:        make(map[int64]string),
		WaitingByKey:    make(map[WaitingKey]*Client),
		WaitingByGame:   make(map[game.GameType]*Client),
		GameRepo:        gameRepo,
		GameHistoryRepo: gameHistoryRepo,
	}
}

func NewHubWithUserRepo(gameRepo *repository.GameRepository, gameHistoryRepo *repository.GameHistoryRepository, userRepo *repository.UserRepository) *Hub {
	hub := NewHub(gameRepo, gameHistoryRepo)
	hub.UserRepo = userRepo
	return hub
}

func (h *Hub) AssignClient(c *Client) *Room {
	h.mu.Lock()

	// преобразуем строковый тип игры в GameType
	gameType := game.GameType(c.GameType)
	if gameType != game.TypeRPS && gameType != game.TypeMines {
		gameType = game.TypeRPS // по умолчанию
	}

	// создаем ключ ожидания для матчмейкинга по типу игры + ставке + валюте
	waitingKey := WaitingKey{
		GameType:  gameType,
		BetAmount: c.BetAmount,
		Currency:  c.Currency,
	}

	log.Printf("Hub.AssignClient: пользователь=%d игра=%s ставка=%d валюта=%s - назначение через слот ожидания (комнат=%d)",
		c.UserID, gameType, c.BetAmount, c.Currency, len(h.Rooms))

	// очищаем любое устаревшее состояние для этого пользователя (например, из предыдущей игры/реконнекта)
	if oldRoomID, exists := h.UserRoom[c.UserID]; exists {
		log.Printf("Hub.AssignClient: пользователь=%d имеет устаревшее отображение комнаты на %s, очищаем", c.UserID, oldRoomID)
		delete(h.UserRoom, c.UserID)
		// если пользователь был в WaitingByKey, очищаем
		// collect keys first to avoid modifying map during iteration
		var keysToDelete []WaitingKey
		for key, waiting := range h.WaitingByKey {
			if waiting != nil && waiting.UserID == c.UserID {
				keysToDelete = append(keysToDelete, key)
			}
		}
		for _, key := range keysToDelete {
			log.Printf("Hub.AssignClient: очистка устаревшего слота ожидания для пользователя=%d ключ=%s", c.UserID, key)
			delete(h.WaitingByKey, key)
		}
		// устаревшее: также проверяем WaitingByGame
		if waiting := h.WaitingByGame[gameType]; waiting != nil && waiting.UserID == c.UserID {
			log.Printf("Hub.AssignClient: очистка устаревшего слота ожидания (устаревшее) для пользователя=%d", c.UserID)
			delete(h.WaitingByGame, gameType)
		}
	}

	// если есть ожидающий клиент для этого точного ключа (игра + ставка + валюта), пытаемся соединить
	waiting := h.WaitingByKey[waitingKey]
	if waiting != nil {
		// не соединяем с самим собой
		if waiting.UserID != c.UserID {
			// проверяем, живое ли еще соединение ожидающего клиента
			// пытаясь выполнить неблокирующую отправку в его Send канал
			waitingAlive := false
			select {
			case waiting.Send <- []byte(`{"type":"ping"}`):
				waitingAlive = true
			default:
				// канал заполнен или закрыт - клиент может быть мертв
				log.Printf("Hub.AssignClient: Send канал ожидающего клиента=%d заблокирован, возможно мертв", waiting.UserID)
			}

			if !waitingAlive {
				log.Printf("Hub.AssignClient: ожидающий клиент=%d кажется мертвым, очищаем слот ожидания", waiting.UserID)
				delete(h.WaitingByKey, waitingKey)
				// переходим к созданию новой комнаты
			} else {
				// находим id комнаты ожидающего клиента
				roomID, ok := h.UserRoom[waiting.UserID]
				if ok {
					foundRoom, ok2 := h.Rooms[roomID]
					if ok2 {
						// убеждаемся, что ожидающий клиент все еще присутствует в клиентах комнаты
						foundRoom.mu.RLock()
						_, stillThere := foundRoom.Clients[waiting.UserID]
						foundRoom.mu.RUnlock()
						if stillThere {
							log.Printf("Hub.AssignClient: соединение пользователя=%d с ожидающим пользователем=%d в комнате=%s игра=%s ставка=%d валюта=%s",
								c.UserID, waiting.UserID, foundRoom.ID, gameType, c.BetAmount, c.Currency)

							// обновляем существующую игру вторым игроком (сохраняем состояние настройки для Mines)
							foundRoom.mu.Lock()
							foundRoom.game.SetSecondPlayer(c.UserID)
							foundRoom.Clients[c.UserID] = c
							foundRoom.mu.Unlock()

							h.UserRoom[c.UserID] = foundRoom.ID
							// очищаем слот ожидания для этого ключа
							delete(h.WaitingByKey, waitingKey)
							h.mu.Unlock()

							log.Printf("Hub.AssignClient: собираюсь зарегистрировать пользователя=%d в комнату=%s", c.UserID, foundRoom.ID)

							// неблокирующая отправка для избежания deadlock'а
							select {
							case foundRoom.Register <- c:
								log.Printf("Hub.AssignClient: зарегистрирован пользователь=%d в комнату=%s", c.UserID, foundRoom.ID)
							case <-time.After(5 * time.Second):
								log.Printf("Hub.AssignClient: ТАЙМАУТ регистрации пользователя=%d в комнату=%s", c.UserID, foundRoom.ID)
								return nil
							}

							return foundRoom
						}
						// если ожидающий клиент отсутствует в комнате, очищаем устаревший слот ожидания
						log.Printf("Hub.AssignClient: найден устаревший ожидающий клиент=%d (не в комнате), очищаем слот ожидания", waiting.UserID)
						delete(h.WaitingByKey, waitingKey)
					} else {
						// комната отсутствует, очищаем устаревший слот ожидания
						log.Printf("Hub.AssignClient: комната ожидающего клиента отсутствует (id=%s), очищаем слот ожидания", roomID)
						delete(h.WaitingByKey, waitingKey)
					}
				} else {
					// нет отображения для ожидающего пользователя, очищаем
					log.Printf("Hub.AssignClient: ожидающий пользователь не отображен ни на одну комнату, очищаем слот ожидания")
					delete(h.WaitingByKey, waitingKey)
				}
			}
		} else {
			// ожидающий - тот же пользователь - очищаем и переходим к созданию комнаты
			log.Printf("Hub.AssignClient: ожидающий пользователь тот же, что и текущий пользователь=%d, очищаем слот ожидания", c.UserID)
			delete(h.WaitingByKey, waitingKey)
		}
	}

	// создаем новую комнату для этого типа игры с информацией о ставке
	players := [2]int64{c.UserID, 0}
	room := h.newRoomWithBet(gameType, players, c.BetAmount, c.Currency)

	if room == nil {
		log.Printf("Hub.AssignClient: не удалось создать комнату для пользователя=%d", c.UserID)
		h.mu.Unlock()
		return nil
	}

	log.Printf("Hub.AssignClient: пользователь=%d создал новую комнату=%s игра=%s ставка=%d валюта=%s",
		c.UserID, room.ID, gameType, c.BetAmount, c.Currency)
	// резервируем слот для этого клиента сразу, чтобы избежать гонки с другим AssignClient
	room.mu.Lock()
	room.Clients[c.UserID] = c
	room.mu.Unlock()

	log.Printf("Hub.AssignClient: зарезервирован пользователь=%d в комнате=%s (предварительная регистрация)", c.UserID, room.ID)

	h.UserRoom[c.UserID] = room.ID
	// помечаем этого клиента как ожидающего пира с такой же ставкой
	h.WaitingByKey[waitingKey] = c

	h.mu.Unlock()

	log.Printf("Hub.AssignClient: регистрация пользователя=%d в НОВОЙ комнате=%s", c.UserID, room.ID)

	// неблокирующая отправка для избежания deadlock'а, если room.Run() завершился
	select {
	case room.Register <- c:
		log.Printf("Hub.AssignClient: успешно зарегистрирован пользователь=%d в комнату=%s", c.UserID, room.ID)
	case <-time.After(5 * time.Second):
		log.Printf("Hub.AssignClient: ТАЙМАУТ регистрации пользователя=%d в комнату=%s - комната могла завершиться", c.UserID, room.ID)
		return nil
	}

	return room
}

func (h *Hub) newRoom(gameType game.GameType, players [2]int64) *Room {
	return h.newRoomWithBet(gameType, players, 0, "gems")
}

func (h *Hub) newRoomWithBet(gameType game.GameType, players [2]int64, betAmount int64, currency string) *Room {
	h.roomSeq++
	id := strconv.FormatInt(h.roomSeq, 10)

	factory := game.NewFactory()
	g, err := factory.CreateGame(gameType, id, players)
	if err != nil {
		log.Printf("Hub.newRoom: не удалось создать игру: %v", err)
		return nil
	}

	room := NewRoomWithRepo(id, g, h.GameRepo, h.GameHistoryRepo, h)
	room.BetAmount = betAmount
	room.Currency = currency
	room.UserRepo = h.UserRepo
	h.Rooms[id] = room

	log.Printf("Hub.newRoom: создана комната=%s игра=%s ставка=%d валюта=%s, запуск Run()", id, gameType, betAmount, currency)
	go room.Run()

	return room
}

func (h *Hub) OnDisconnect(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	log.Printf("Hub.OnDisconnect: пользователь=%d тип игры=%s ставка=%d валюта=%s", c.UserID, c.GameType, c.BetAmount, c.Currency)

	// очищаем слот ожидания, если это был ожидающий клиент для любого ключа
	// collect keys first to avoid modifying map during iteration
	var keysToDelete []WaitingKey
	for key, waiting := range h.WaitingByKey {
		if waiting != nil && waiting.UserID == c.UserID {
			keysToDelete = append(keysToDelete, key)
		}
	}
	for _, key := range keysToDelete {
		log.Printf("Hub.OnDisconnect: очистка слота ожидания для пользователя=%d ключ=%s", c.UserID, key)
		delete(h.WaitingByKey, key)
	}

	// устаревшее: также проверяем WaitingByGame
	var gameTypesToDelete []game.GameType
	for gt, waiting := range h.WaitingByGame {
		if waiting != nil && waiting.UserID == c.UserID {
			gameTypesToDelete = append(gameTypesToDelete, gt)
		}
	}
	for _, gt := range gameTypesToDelete {
		log.Printf("Hub.OnDisconnect: очистка слота ожидания (устаревшее) для пользователя=%d игра=%s", c.UserID, gt)
		delete(h.WaitingByGame, gt)
	}

	if roomID, ok := h.UserRoom[c.UserID]; ok {
		log.Printf("Hub.OnDisconnect: пользователь=%d комната=%s", c.UserID, roomID)
		if room, ok := h.Rooms[roomID]; ok {
			// неблокирующая отправка для избежания deadlock'а, если room.Run() завершился
			select {
			case room.Disconnect <- c:
			default:
				log.Printf("Hub.OnDisconnect: комната=%s канал Disconnect заполнен/закрыт", roomID)
			}
		}
	}
}

func (h *Hub) StartCleanup() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			h.cleanupStaleRooms()
		}
	}()

	// более частая очистка для слотов ожидания (каждые 30 секунд)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			h.cleanupStaleWaiting()
		}
	}()
}

func (h *Hub) cleanupStaleWaiting() {
	h.mu.Lock()
	defer h.mu.Unlock()

	// очистка WaitingByKey
	// First pass: identify stale entries
	type staleEntry struct {
		key     WaitingKey
		userID  int64
	}
	var staleKeys []staleEntry

	for key, waiting := range h.WaitingByKey {
		if waiting == nil {
			continue
		}

		// проверяем, жив ли еще ожидающий клиент
		alive := false
		select {
		case waiting.Send <- []byte(`{"type":"ping"}`):
			alive = true
		default:
			// канал заблокирован - клиент может быть мертв
		}

		if !alive {
			staleKeys = append(staleKeys, staleEntry{key: key, userID: waiting.UserID})
		}
	}

	// Second pass: delete stale entries
	for _, entry := range staleKeys {
		log.Printf("Hub.cleanupStaleWaiting: удаление устаревшего ожидающего клиента=%d ключ=%s", entry.userID, entry.key)
		delete(h.WaitingByKey, entry.key)

		// также очищаем отображение UserRoom
		if roomID, ok := h.UserRoom[entry.userID]; ok {
			if room, ok := h.Rooms[roomID]; ok {
				room.mu.Lock()
				delete(room.Clients, entry.userID)
				clientsLeft := len(room.Clients)
				room.mu.Unlock()

				if clientsLeft == 0 {
					delete(h.Rooms, roomID)
					log.Printf("Hub.cleanupStaleWaiting: удалена пустая комната=%s", roomID)
				}
			}
			delete(h.UserRoom, entry.userID)
		}
	}

	// устаревшее: также очищаем WaitingByGame
	type staleGameEntry struct {
		gameType game.GameType
		userID   int64
	}
	var staleGameKeys []staleGameEntry

	for gameType, waiting := range h.WaitingByGame {
		if waiting == nil {
			continue
		}

		alive := false
		select {
		case waiting.Send <- []byte(`{"type":"ping"}`):
			alive = true
		default:
		}

		if !alive {
			staleGameKeys = append(staleGameKeys, staleGameEntry{gameType: gameType, userID: waiting.UserID})
		}
	}

	for _, entry := range staleGameKeys {
		log.Printf("Hub.cleanupStaleWaiting: удаление устаревшего ожидающего клиента=%d игра=%s (устаревшее)", entry.userID, entry.gameType)
		delete(h.WaitingByGame, entry.gameType)

		if roomID, ok := h.UserRoom[entry.userID]; ok {
			if room, ok := h.Rooms[roomID]; ok {
				room.mu.Lock()
				delete(room.Clients, entry.userID)
				clientsLeft := len(room.Clients)
				room.mu.Unlock()

				if clientsLeft == 0 {
					delete(h.Rooms, roomID)
					log.Printf("Hub.cleanupStaleWaiting: удалена пустая комната=%s", roomID)
				}
			}
			delete(h.UserRoom, entry.userID)
		}
	}
}

func (h *Hub) cleanupStaleRooms() {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()

	for roomID, room := range h.Rooms {
		room.mu.RLock()
		clientsCount := len(room.Clients)
		createdAt := room.createdAt
		room.mu.RUnlock()

		if clientsCount == 0 && now.Sub(createdAt) > time.Hour {
			delete(h.Rooms, roomID)

			for uid, rid := range h.UserRoom {
				if rid == roomID {
					delete(h.UserRoom, uid)
				}
			}

			log.Printf("очищена устаревшая комната: %s", roomID)
		}
	}
}