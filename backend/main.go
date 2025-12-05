package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/google/uuid"
)

// init загружает переменные окружения из .env файла при запуске приложения
func init() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Println("Файл .env не найден")
	}
}

// Константы для настройки WebSocket соединений
const (
	writeWait      = 10 * time.Second  // Максимальное время на запись сообщения
	pongWait       = 60 * time.Second  // Максимальное время бездействия соединения
	pingPeriod     = (pongWait * 9) / 10 // Период отправки ping сообщений
	maxMessageSize = 8192              // Максимальный размер сообщения в байтах
)

// Типы сообщений для WebSocket протокола
const (
	MessageTypeUserJoined = "USER_JOINED"   // Сообщение о входе пользователя в комнату
	MessageTypeThrowStart = "THROW_START"   // Сообщение о начале броска кубиков
	MessageTypeThrowResult = "THROW_RESULT" // Сообщение с результатом броска
)

// Структуры данных для работы с базой данных

// User представляет пользователя в системе
type User struct {
	ID       uuid.UUID `json:"id"`       // Уникальный идентификатор пользователя
	Username string    `json:"username"` // Имя пользователя
}

// Roll представляет запись о броске кубиков
type Roll struct {
	ID        int             `json:"id"`                  // ID записи
	RoomID    uuid.UUID       `json:"room_id"`             // ID комнаты
	UserID    uuid.UUID       `json:"user_id"`             // ID пользователя
	Config    string          `json:"config"`              // Конфигурация броска (например, "2d6")
	Result    json.RawMessage `json:"result"`              // Результат броска в формате JSON
	CreatedAt time.Time       `json:"created_at"`          // Время создания записи
}

// Структуры данных для WebSocket сообщений

// Vector3 представляет трехмерный вектор для физики броска
type Vector3 struct {
	X float64 `json:"x"` // Координата X
	Y float64 `json:"y"` // Координата Y
	Z float64 `json:"z"` // Координата Z
}

// ThrowStartPayload содержит данные о начале броска
type ThrowStartPayload struct {
	Impulse   Vector3 `json:"impulse"`   // Вектор силы броска
	Torque    Vector3 `json:"torque"`    // Вектор вращения
	DiceCount int     `json:"diceCount"` // Количество кубиков
	DiceType  string  `json:"diceType"`  // Тип кубиков (например, "d6")
}

// ThrowResultPayload содержит результат броска
type ThrowResultPayload struct {
	Values    []int   `json:"values"`    // Выпавшие значения на кубиках
	Positions []float64 `json:"positions,omitempty"` // Позиции кубиков (опционально)
}

// WebSocketMessage - общий формат сообщения для WebSocket
type WebSocketMessage struct {
	Type    string          `json:"type"`    // Тип сообщения
	Payload json.RawMessage `json:"payload"` // Тело сообщения в формате JSON
}

// UserJoinedPayload содержит данные о вошедшем пользователе
type UserJoinedPayload struct {
	UserID   uuid.UUID `json:"userId"`   // ID пользователя
	Username string    `json:"username"` // Имя пользователя
}

// Client представляет одно WebSocket соединение
type Client struct {
	hub      *Hub           // Ссылка на хаб
	conn     *websocket.Conn // WebSocket соединение
	send     chan []byte    // Канал для отправки сообщений клиенту
	roomID   uuid.UUID      // ID комнаты, в которой находится клиент
	userID   uuid.UUID      // ID пользователя
	username string         // Имя пользователя
}

// readPump читает сообщения из WebSocket соединения
func (c *Client) readPump() {
	// Отложенный вызов при выходе из функции - закрываем соединение и удаляем клиента из хаба
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	// Настраиваем ограничения на чтение
	c.conn.SetReadLimit(maxMessageSize)                    // Устанавливаем максимальный размер сообщения
	c.conn.SetReadDeadline(time.Now().Add(pongWait))       // Устанавливаем таймаут чтения
	c.conn.SetPongHandler(func(string) error {             // Обработчик pong сообщений
		c.conn.SetReadDeadline(time.Now().Add(pongWait))   // Сбрасываем таймаут при получении pong
		return nil
	})

	// Бесконечный цикл чтения сообщений
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			// Проверяем, является ли ошибка ожидаемым закрытием соединения
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Ошибка чтения сообщения: %v", err)
			}
			break
		}

		// Разбираем JSON сообщение
		var wsMsg WebSocketMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("Ошибка разбора сообщения: %v", err)
			continue
		}

		// Обрабатываем сообщение в зависимости от его типа
		switch wsMsg.Type {
		case MessageTypeThrowStart:
			// Обработка начала броска
			var payload ThrowStartPayload
			if err := json.Unmarshal(wsMsg.Payload, &payload); err != nil {
				log.Printf("Неверный формат THROW_START: %v", err)
				continue
			}

			// Сохраняем конфигурацию броска для последующей валидации
			c.hub.storeThrowConfig(c.roomID, c.userID, payload)
			
			// Публикуем сообщение в Redis для рассылки другим клиентам
			c.hub.publishToRedis(c.roomID, message)

		case MessageTypeThrowResult:
			// Обработка результата броска
			var payload ThrowResultPayload
			if err := json.Unmarshal(wsMsg.Payload, &payload); err != nil {
				log.Printf("Неверный формат THROW_RESULT: %v", err)
				continue
			}

			// Валидация результатов броска (проверка соответствия типу кубика)
			if !c.validateDiceValues(payload.Values, c.hub.getDiceTypeFromLastThrow(c.roomID, c.userID)) {
				log.Printf("Неверные значения кубиков для пользователя %s", c.userID)
				continue
			}

			// Сохранение результата в базу данных
			if err := c.saveRollToDB(c.roomID, c.userID, payload); err != nil {
				log.Printf("Ошибка сохранения броска в БД: %v", err)
				continue
			}

			// Публикация результата в Redis для рассылки другим клиентам
			c.hub.publishToRedis(c.roomID, message)

		default:
			log.Printf("Неизвестный тип сообщения: %s", wsMsg.Type)
		}
	}
}

// writePump отправляет сообщения клиенту через WebSocket
func (c *Client) writePump() {
	// Создаем таймер для отправки ping сообщений
	ticker := time.NewTicker(pingPeriod)
	
	// Отложенный вызов при выходе из функции - останавливаем таймер и закрываем соединение
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	// Бесконечный цикл отправки сообщений
	for {
		select {
		case message, ok := <-c.send:
			// Устанавливаем таймаут на запись
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			
			// Если канал закрыт, отправляем сообщение о закрытии соединения
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Получаем writer для записи сообщения
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message) // Записываем сообщение

			// Закрываем writer
			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C: // Отправка ping сообщения для поддержания соединения
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// validateDiceValues проверяет корректность выпавших значений кубиков
func (c *Client) validateDiceValues(values []int, diceType string) bool {
	if diceType == "" {
		return true // Если тип кубика неизвестен, пропускаем валидацию
	}

	// Извлекаем количество граней из типа кубика (например, "d6" -> 6)
	var sides int
	_, err := fmt.Sscanf(diceType, "d%d", &sides)
	if err != nil || sides <= 0 {
		return false
	}

	// Проверяем каждое значение
	for _, value := range values {
		if value < 1 || value > sides {
			return false // Значение выходит за пределы допустимого диапазона
		}
	}
	return true
}

// saveRollToDB сохраняет результат броска в базу данных
func (c *Client) saveRollToDB(roomID, userID uuid.UUID, payload ThrowResultPayload) error {
	// Получаем конфигурацию броска из хаба
	diceConfig := c.hub.getDiceConfig(roomID, userID)

	// Создаем JSON объект с результатами
	resultData, err := json.Marshal(map[string]interface{}{
		"values":    payload.Values,
		"positions": payload.Positions,
	})
	if err != nil {
		return fmt.Errorf("ошибка создания JSON результата: %w", err)
	}

	// Выполняем SQL запрос для вставки данных
	_, err = c.hub.db.ExecContext(context.Background(),
		`INSERT INTO rolls (room_id, user_id, config, result_data) 
		VALUES ($1, $2, $3, $4)`,
		roomID, userID, diceConfig, resultData)

	return err
}

// Hub управляет всеми активными WebSocket соединениями и комнатами
type Hub struct {
	clients    map[*Client]bool                  // Все активные клиенты
	rooms      map[uuid.UUID]map[*Client]bool    // Клиенты по комнатам (roomID -> клиенты)
	roomsMutex sync.RWMutex                      // Мьютекс для безопасного доступа к rooms

	// Интеграция с Redis
	redisClient *redis.Client                    // Клиент Redis
	redisCtx    context.Context                  // Контекст для Redis операций
	redisCancel context.CancelFunc               // Функция отмены контекста Redis

	// База данных
	db *sql.DB                                  // Подключение к PostgreSQL

	// Каналы для управления клиентами
	register   chan *Client                      // Канал для регистрации новых клиентов
	unregister chan *Client                      // Канал для отмены регистрации клиентов
	broadcast  chan []byte                       // Канал для широковещательных сообщений

	// Хранение конфигурации последних бросков для валидации
	lastThrowConfig map[uuid.UUID]map[uuid.UUID]ThrowStartPayload // roomID -> userID -> конфигурация
	lastThrowMutex  sync.RWMutex                                  // Мьютекс для безопасного доступа к lastThrowConfig
}

// NewHub создает новый экземпляр хаба
func NewHub(db *sql.DB, redisClient *redis.Client) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		clients:         make(map[*Client]bool),
		rooms:           make(map[uuid.UUID]map[*Client]bool),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		broadcast:       make(chan []byte),
		redisClient:     redisClient,
		redisCtx:        ctx,
		redisCancel:     cancel,
		db:              db,
		lastThrowConfig: make(map[uuid.UUID]map[uuid.UUID]ThrowStartPayload),
	}
}

// getDiceTypeFromLastThrow возвращает тип кубика из последнего броска пользователя
func (h *Hub) getDiceTypeFromLastThrow(roomID, userID uuid.UUID) string {
	h.lastThrowMutex.RLock()
	defer h.lastThrowMutex.RUnlock()

	if roomMap, exists := h.lastThrowConfig[roomID]; exists {
		if config, exists := roomMap[userID]; exists {
			return config.DiceType
		}
	}
	return ""
}

// getDiceConfig возвращает строку конфигурации броска (например, "2d6")
func (h *Hub) getDiceConfig(roomID, userID uuid.UUID) string {
	h.lastThrowMutex.RLock()
	defer h.lastThrowMutex.RUnlock()

	if roomMap, exists := h.lastThrowConfig[roomID]; exists {
		if config, exists := roomMap[userID]; exists {
			return fmt.Sprintf("%dd%s", config.DiceCount, config.DiceType)
		}
	}
	return "1d6" // Значение по умолчанию
}

// storeThrowConfig сохраняет конфигурацию броска для последующей валидации
func (h *Hub) storeThrowConfig(roomID, userID uuid.UUID, config ThrowStartPayload) {
	h.lastThrowMutex.Lock()
	defer h.lastThrowMutex.Unlock()

	if _, exists := h.lastThrowConfig[roomID]; !exists {
		h.lastThrowConfig[roomID] = make(map[uuid.UUID]ThrowStartPayload)
	}
	h.lastThrowConfig[roomID][userID] = config
}

// publishToRedis публикует сообщение в Redis канал для комнаты
func (h *Hub) publishToRedis(roomID uuid.UUID, message []byte) {
	channel := fmt.Sprintf("room:%s:events", roomID)
	if err := h.redisClient.Publish(h.redisCtx, channel, message).Err(); err != nil {
		log.Printf("Ошибка публикации в Redis: %v", err)
	}
}

// subscribeToRoom подписывается на Redis канал для комнаты
func (h *Hub) subscribeToRoom(roomID uuid.UUID) {
	channel := fmt.Sprintf("room:%s:events", roomID)
	pubsub := h.redisClient.Subscribe(h.redisCtx, channel)

	// Запускаем горутину для получения сообщений из Redis
	go func() {
		for {
			select {
			case <-h.redisCtx.Done(): // Если контекст отменен, выходим из цикла
				return
			default:
				// Получаем сообщение из Redis
				msg, err := pubsub.ReceiveMessage(h.redisCtx)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						return
					}
					log.Printf("Ошибка получения сообщения из Redis: %v", err)
					time.Sleep(time.Second) // Ждем перед повторной попыткой
					continue
				}

				// Рассылаем сообщение всем клиентам в комнате
				h.roomsMutex.RLock()
				if clients, exists := h.rooms[roomID]; exists {
					for client := range clients {
						select {
						case client.send <- []byte(msg.Payload): // Отправляем сообщение
						default:
							close(client.send) // Если буфер заполнен, закрываем канал
						}
					}
				}
				h.roomsMutex.RUnlock()
			}
		}
	}()
}

// run запускает основной цикл обработки событий хаба
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			// Регистрация нового клиента
			h.roomsMutex.Lock()
			h.clients[client] = true

			// Добавление клиента в комнату
			if _, exists := h.rooms[client.roomID]; !exists {
				h.rooms[client.roomID] = make(map[*Client]bool)
				// Подписываемся на Redis канал для новой комнаты
				h.subscribeToRoom(client.roomID)
			}
			h.rooms[client.roomID][client] = true
			h.roomsMutex.Unlock()

			// Отправка сообщения о входе пользователя всем в комнате
			userJoinedMsg, _ := json.Marshal(WebSocketMessage{
				Type: MessageTypeUserJoined,
				Payload: json.RawMessage(fmt.Sprintf(`{"userId": "%s", "username": "%s"}`,
					client.userID, client.username)),
			})
			h.broadcastToRoom(client.roomID, userJoinedMsg)

		case client := <-h.unregister:
			// Отмена регистрации клиента
			h.roomsMutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send) // Закрываем канал отправки

				// Удаляем клиента из комнаты
				if roomClients, exists := h.rooms[client.roomID]; exists {
					delete(roomClients, client)
					if len(roomClients) == 0 {
						delete(h.rooms, client.roomID) // Удаляем комнату, если она пуста
					}
				}
			}
			h.roomsMutex.Unlock()

		case message := <-h.broadcast:
			// Обработка широковещательных сообщений (не используется в текущей реализации)
			h.roomsMutex.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.roomsMutex.RUnlock()
		}
	}
}

// broadcastToRoom рассылает сообщение всем клиентам в указанной комнате
func (h *Hub) broadcastToRoom(roomID uuid.UUID, message []byte) {
	h.roomsMutex.RLock()
	defer h.roomsMutex.RUnlock()

	if clients, exists := h.rooms[roomID]; exists {
		for client := range clients {
			select {
			case client.send <- message:
			default:
				close(client.send) // Если буфер заполнен, закрываем канал
			}
		}
	}
}

// shutdown выполняет graceful shutdown хаба
func (h *Hub) shutdown() {
	h.redisCancel() // Отменяем контекст Redis

	h.roomsMutex.Lock()
	defer h.roomsMutex.Unlock()

	// Закрываем все соединения с клиентами
	for client := range h.clients {
		close(client.send)
		client.conn.Close()
		delete(h.clients, client)
	}

	// Очищаем карту комнат
	h.rooms = make(map[uuid.UUID]map[*Client]bool)
}

// setupDatabase настраивает подключение к базе данных PostgreSQL
func setupDatabase() (*sql.DB, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, errors.New("переменная окружения DATABASE_URL не установлена")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе данных: %w", err)
	}

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка проверки подключения к базе данных: %w", err)
	}

	// Настраиваем пул соединений
	db.SetMaxOpenConns(25)                 // Максимальное количество открытых соединений
	db.SetMaxIdleConns(25)                 // Максимальное количество простаивающих соединений
	db.SetConnMaxLifetime(5 * time.Minute) // Максимальное время жизни соединения

	return db, nil
}

// setupRedis настраивает подключение к Redis
func setupRedis() (*redis.Client, error) {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379" // Адрес по умолчанию
	}

	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	// Проверяем подключение к Redis
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("ошибка подключения к Redis: %w", err)
	}

	return client, nil
}

// authMiddleware - middleware для аутентификации пользователей
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// В реальном приложении здесь должна быть валидация JWT/куки
		// Для примера используем простого гостевого пользователя
		userID := r.Header.Get("X-User-ID")
		username := r.Header.Get("X-Username")

		if userID == "" {
			userID = uuid.New().String() // Генерируем новый UUID если нет ID
			username = "Guest"            // Имя по умолчанию
		}

		// Создаем контекст с информацией о пользователе
		ctx := context.WithValue(r.Context(), "user", User{
			ID:       uuid.MustParse(userID),
			Username: username,
		})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// upgrader - конфигурация для обновления HTTP соединения до WebSocket
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,  // Размер буфера для чтения
	WriteBufferSize: 1024,  // Размер буфера для записи
	CheckOrigin: func(r *http.Request) bool {
		// В production необходимо проверять origin для безопасности
		return true
	},
}

// serveWs обрабатывает WebSocket соединения
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	// Получаем ID комнаты из URL параметров
	roomIDStr := chi.URLParam(r, "roomID")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		http.Error(w, "Неверный ID комнаты", http.StatusBadRequest)
		return
	}

	// Получаем информацию о пользователе из контекста
	userVal := r.Context().Value("user")
	if userVal == nil {
		http.Error(w, "Неавторизован", http.StatusUnauthorized)
		return
	}

	user := userVal.(User)

	// Обновляем соединение до WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Ошибка обновления соединения: %v", err)
		return
	}

	// Создаем нового клиента
	client := &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256), // Буферизированный канал для отправки сообщений
		roomID:   roomID,
		userID:   user.ID,
		username: user.Username,
	}

	// Регистрируем клиента в хабе
	client.hub.register <- client

	// Запускаем горутины для чтения и записи
	go client.writePump()
	go client.readPump()
}

// getRollHistory возвращает историю бросков для комнаты
func getRollHistory(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем ID комнаты из URL параметров
		roomIDStr := chi.URLParam(r, "roomID")
		roomID, err := uuid.Parse(roomIDStr)
		if err != nil {
			http.Error(w, "Неверный ID комнаты", http.StatusBadRequest)
			return
		}

		// Запрашиваем историю бросков из базы данных с информацией о пользователях
		rows, err := db.QueryContext(r.Context(),
			`SELECT r.id, r.room_id, r.user_id, r.config, r.result_data, r.created_at, u.username
			FROM rolls r
			JOIN users u ON r.user_id = u.id
			WHERE r.room_id = $1
			ORDER BY r.created_at DESC
			LIMIT 50`,
			roomID)
		if err != nil {
			http.Error(w, "Ошибка получения истории бросков", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Собираем результаты в массив
		var rolls []map[string]interface{}
		for rows.Next() {
			var roll Roll
			var username string
			var resultData []byte

			// Сканируем данные из результата запроса
			if err := rows.Scan(&roll.ID, &roll.RoomID, &roll.UserID, &roll.Config, &resultData, &roll.CreatedAt, &username); err != nil {
				http.Error(w, "Ошибка обработки истории бросков", http.StatusInternalServerError)
				return
			}

			var result map[string]interface{}
			// Разбираем JSON результат
			if err := json.Unmarshal(resultData, &result); err != nil {
				result = make(map[string]interface{}) // Создаем пустой объект при ошибке
			}

			// Добавляем бросок в результат
			rolls = append(rolls, map[string]interface{}{
				"id":        roll.ID,
				"room_id":   roll.RoomID,
				"user_id":   roll.UserID,
				"username":  username,
				"config":    roll.Config,
				"result":    result,
				"created_at": roll.CreatedAt.Format(time.RFC3339), // Форматируем время в ISO 8601
			})
		}

		if err := rows.Err(); err != nil {
			http.Error(w, "Ошибка итерации по истории бросков", http.StatusInternalServerError)
			return
		}

		// Устанавливаем заголовки и отправляем JSON ответ
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rolls": rolls,
		})
	}
}

// setupRoutes настраивает маршруты HTTP сервера
func setupRoutes(hub *Hub, db *sql.DB) *chi.Mux {
	r := chi.NewRouter()

	// CORS middleware - разрешаем запросы с любых origin для разработки
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // 5 минут
	}))

	// Стандартные middleware Chi
	r.Use(middleware.RequestID)    // Добавляем ID запроса
	r.Use(middleware.RealIP)       // Определяем реальный IP клиента
	r.Use(middleware.Logger)       // Логируем запросы
	r.Use(middleware.Recoverer)    // Восстанавливаемся после паники
	r.Use(authMiddleware)          // Middleware аутентификации

	// WebSocket эндпоинт
	r.Get("/ws/{roomID}", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	// REST эндпоинт для получения истории бросков
	r.Get("/api/room/{roomID}/history", getRollHistory(db))

	// Эндпоинт для проверки здоровья сервиса
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	return r
}

// main - основная функция приложения
func main() {
	// Настраиваем подключение к базе данных
	db, err := setupDatabase()
	if err != nil {
		log.Fatalf("Ошибка настройки базы данных: %v", err)
	}
	defer db.Close() // Отложенно закрываем соединение с БД

	// Настраиваем подключение к Redis
	redisClient, err := setupRedis()
	if err != nil {
		log.Fatalf("Ошибка настройки Redis: %v", err)
	}

	// Создаем хаб
	hub := NewHub(db, redisClient)
	go hub.run() // Запускаем хаб в отдельной горутине

	// Настраиваем маршруты
	router := setupRoutes(hub, db)

	// Определяем порт для сервера
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Порт по умолчанию
	}

	// Создаем HTTP сервер
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Настройка graceful shutdown
	go func() {
		// Создаем канал для сигналов ОС
		sig := make(chan os.Signal, 1)
		// Подписываемся на сигналы прерывания и завершения
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig // Ждем получения сигнала

		log.Println("Завершение работы сервера...")

		// Завершаем работу хаба
		hub.shutdown()

		// Завершаем работу HTTP сервера с таймаутом
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Ошибка завершения HTTP сервера: %v", err)
		}

		// Закрываем соединение с базой данных
		if err := db.Close(); err != nil {
			log.Printf("Ошибка закрытия соединения с БД: %v", err)
		}

		// Закрываем соединение с Redis
		if err := redisClient.Close(); err != nil {
			log.Printf("Ошибка закрытия соединения с Redis: %v", err)
		}

		log.Println("Сервер успешно завершил работу")
		os.Exit(0)
	}()

	// Запускаем сервер
	log.Printf("Сервер запускается на порту %s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}