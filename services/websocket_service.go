package services

import (
	"context"
	"encoding/json"
	"log"
	"premium-chat-backend/models"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

type WebSocketService struct {
	db          *gorm.DB
	redisClient *redis.Client
	clients     map[*Client]bool
	rooms       map[uuid.UUID]map[*Client]bool
	register    chan *Client
	unregister  chan *Client
	broadcast   chan *BroadcastMessage
	mu          sync.RWMutex
}

type Client struct {
	ID     uuid.UUID
	User   *models.User
	Conn   *websocket.Conn
	Send   chan *WebSocketMessage
	Rooms  map[uuid.UUID]bool
	Done   chan bool
	LastActivity time.Time
}

type WebSocketMessage struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	RoomID  *uuid.UUID  `json:"room_id,omitempty"`
	UserID  *uuid.UUID  `json:"user_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type BroadcastMessage struct {
	RoomID  uuid.UUID
	Message *WebSocketMessage
	Sender  *Client
}

type MessageData struct {
	ID         uuid.UUID    `json:"id"`
	Content    string       `json:"content"`
	Type       models.MessageType `json:"type"`
	UserID     uuid.UUID    `json:"user_id"`
	Username   string       `json:"username"`
	ChatRoomID uuid.UUID    `json:"chat_room_id"`
	ParentID   *uuid.UUID   `json:"parent_id,omitempty"`
	Mentions   []uuid.UUID  `json:"mentions,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
}

type UserStatusData struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Status   string    `json:"status"` // online, offline, away
}

type TypingData struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	RoomID   uuid.UUID `json:"room_id"`
	IsTyping bool      `json:"is_typing"`
}

func NewWebSocketService(db *gorm.DB, redisClient *redis.Client) *WebSocketService {
	return &WebSocketService{
		db:          db,
		redisClient: redisClient,
		clients:     make(map[*Client]bool),
		rooms:       make(map[uuid.UUID]map[*Client]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   make(chan *BroadcastMessage),
	}
}

func (ws *WebSocketService) Start() {
	// Start connection cleanup routine
	go ws.cleanupConnections()

	for {
		select {
		case client := <-ws.register:
			ws.registerClient(client)

		case client := <-ws.unregister:
			ws.unregisterClient(client)

		case message := <-ws.broadcast:
			ws.broadcastToRoom(message)
		}
	}
}

func (ws *WebSocketService) RegisterClient(conn *websocket.Conn, user *models.User) *Client {
	client := &Client{
		ID:           uuid.New(),
		User:         user,
		Conn:         conn,
		Send:         make(chan *WebSocketMessage, 256),
		Rooms:        make(map[uuid.UUID]bool),
		Done:         make(chan bool),
		LastActivity: time.Now(),
	}

	ws.register <- client

	// Join user to their rooms
	go ws.joinUserRooms(client)

	return client
}

func (ws *WebSocketService) UnregisterClient(client *Client) {
	ws.unregister <- client
}

func (ws *WebSocketService) registerClient(client *Client) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.clients[client] = true

	// Update user status in Redis
	ctx := context.Background()
	userStatus := UserStatusData{
		UserID:   client.User.ID,
		Username: client.User.Username,
		Status:   "online",
	}

	statusJSON, _ := json.Marshal(userStatus)
	ws.redisClient.Set(ctx, "user_status:"+client.User.ID.String(), statusJSON, time.Hour)

	// Notify other clients about user coming online
	ws.broadcastUserStatus(client.User, "online")

	log.Printf("Client registered: %s (%s)", client.User.Username, client.ID)
}

func (ws *WebSocketService) unregisterClient(client *Client) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if _, ok := ws.clients[client]; ok {
		delete(ws.clients, client)
		close(client.Send)
		close(client.Done)

		// Remove from all rooms
		for roomID := range client.Rooms {
			if room, exists := ws.rooms[roomID]; exists {
				delete(room, client)
				if len(room) == 0 {
					delete(ws.rooms, roomID)
				}
			}
		}

		// Update user status in Redis
		ctx := context.Background()
		userStatus := UserStatusData{
			UserID:   client.User.ID,
			Username: client.User.Username,
			Status:   "offline",
		}

		statusJSON, _ := json.Marshal(userStatus)
		ws.redisClient.Set(ctx, "user_status:"+client.User.ID.String(), statusJSON, time.Minute*5)

		// Notify other clients about user going offline
		ws.broadcastUserStatus(client.User, "offline")

		log.Printf("Client unregistered: %s (%s)", client.User.Username, client.ID)
	}
}

func (ws *WebSocketService) joinUserRooms(client *Client) {
	// Get user's chat rooms
	var members []models.ChatRoomMember
	if err := ws.db.Where("user_id = ?", client.User.ID).Find(&members).Error; err != nil {
		log.Printf("Error fetching user rooms: %v", err)
		return
	}

	for _, member := range members {
		ws.joinRoom(client, member.ChatRoomID)
	}
}

func (ws *WebSocketService) joinRoom(client *Client, roomID uuid.UUID) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.rooms[roomID] == nil {
		ws.rooms[roomID] = make(map[*Client]bool)
	}

	ws.rooms[roomID][client] = true
	client.Rooms[roomID] = true

	// Send room joined confirmation
	message := &WebSocketMessage{
		Type:    "room_joined",
		RoomID:  &roomID,
		Timestamp: time.Now(),
	}

	select {
	case client.Send <- message:
	default:
		close(client.Send)
		delete(ws.clients, client)
	}
}

func (ws *WebSocketService) leaveRoom(client *Client, roomID uuid.UUID) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if room, exists := ws.rooms[roomID]; exists {
		delete(room, client)
		if len(room) == 0 {
			delete(ws.rooms, roomID)
		}
	}

	delete(client.Rooms, roomID)

	// Send room left confirmation
	message := &WebSocketMessage{
		Type:    "room_left",
		RoomID:  &roomID,
		Timestamp: time.Now(),
	}

	select {
	case client.Send <- message:
	default:
		close(client.Send)
		delete(ws.clients, client)
	}
}

func (ws *WebSocketService) HandleMessage(client *Client, message *WebSocketMessage) {
	client.LastActivity = time.Now()

	switch message.Type {
	case "join_room":
		if message.RoomID != nil {
			ws.joinRoom(client, *message.RoomID)
		}

	case "leave_room":
		if message.RoomID != nil {
			ws.leaveRoom(client, *message.RoomID)
		}

	case "typing":
		if message.RoomID != nil {
			ws.handleTyping(client, *message.RoomID, message.Data)
		}

	case "message":
		if message.RoomID != nil {
			ws.handleChatMessage(client, *message.RoomID, message.Data)
		}

	case "ping":
		ws.sendToClient(client, &WebSocketMessage{
			Type:      "pong",
			Timestamp: time.Now(),
		})

	default:
		log.Printf("Unknown message type: %s", message.Type)
	}
}

func (ws *WebSocketService) handleTyping(client *Client, roomID uuid.UUID, data interface{}) {
	typingData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	isTyping, ok := typingData["is_typing"].(bool)
	if !ok {
		return
	}

	// Check if user is member of the room
	if !client.Rooms[roomID] {
		return
	}

	// Broadcast typing status to room members
	message := &WebSocketMessage{
		Type:   "typing",
		RoomID: &roomID,
		Data: TypingData{
			UserID:   client.User.ID,
			Username: client.User.Username,
			RoomID:   roomID,
			IsTyping: isTyping,
		},
		Timestamp: time.Now(),
	}

	ws.broadcast <- &BroadcastMessage{
		RoomID:  roomID,
		Message: message,
		Sender:  client,
	}
}

func (ws *WebSocketService) handleChatMessage(client *Client, roomID uuid.UUID, data interface{}) {
	messageData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	content, ok := messageData["content"].(string)
	if !ok || content == "" {
		return
	}

	// Check if user is member of the room
	if !client.Rooms[roomID] {
		return
	}

	// Create message in database
	dbMessage := &models.Message{
		ChatRoomID: roomID,
		UserID:     client.User.ID,
		Content:    content,
		Type:       models.MessageTypeText,
	}

	if err := ws.db.Create(dbMessage).Error; err != nil {
		log.Printf("Error creating message: %v", err)
		return
	}

	// Load user data
	ws.db.Preload("User").First(dbMessage, dbMessage.ID)

	// Broadcast message to room members
	wsMessage := &WebSocketMessage{
		Type:   "message",
		RoomID: &roomID,
		Data: MessageData{
			ID:         dbMessage.ID,
			Content:    dbMessage.Content,
			Type:       dbMessage.Type,
			UserID:     dbMessage.UserID,
			Username:   dbMessage.User.Username,
			ChatRoomID: dbMessage.ChatRoomID,
			ParentID:   dbMessage.ParentID,
			Mentions:   dbMessage.Mentions,
			CreatedAt:  dbMessage.CreatedAt,
		},
		Timestamp: time.Now(),
	}

	ws.broadcast <- &BroadcastMessage{
		RoomID:  roomID,
		Message: wsMessage,
		Sender:  client,
	}

	// Handle premium user message priority
	if client.User.IsPremium() {
		// Premium users get message delivery priority
		// This could be implemented with different queues or priorities
	}
}

func (ws *WebSocketService) broadcastToRoom(broadcast *BroadcastMessage) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	room, exists := ws.rooms[broadcast.RoomID]
	if !exists {
		return
	}

	// Send to all clients in the room except sender
	for client := range room {
		if client != broadcast.Sender {
			ws.sendToClient(client, broadcast.Message)
		}
	}
}

func (ws *WebSocketService) sendToClient(client *Client, message *WebSocketMessage) {
	select {
	case client.Send <- message:
	default:
		close(client.Send)
		delete(ws.clients, client)
	}
}

func (ws *WebSocketService) broadcastUserStatus(user *models.User, status string) {
	message := &WebSocketMessage{
		Type: "user_status",
		Data: UserStatusData{
			UserID:   user.ID,
			Username: user.Username,
			Status:   status,
		},
		Timestamp: time.Now(),
	}

	ws.mu.RLock()
	defer ws.mu.RUnlock()

	for client := range ws.clients {
		ws.sendToClient(client, message)
	}
}

func (ws *WebSocketService) cleanupConnections() {
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()

	for range ticker.C {
		ws.mu.RLock()
		var inactive []*Client
		for client := range ws.clients {
			if time.Since(client.LastActivity) > time.Minute*10 {
				inactive = append(inactive, client)
			}
		}
		ws.mu.RUnlock()

		for _, client := range inactive {
			client.Conn.Close()
			ws.unregister <- client
		}
	}
}

func (ws *WebSocketService) GetOnlineUsers() []UserStatusData {
	ctx := context.Background()
	keys, err := ws.redisClient.Keys(ctx, "user_status:*").Result()
	if err != nil {
		return []UserStatusData{}
	}

	var users []UserStatusData
	for _, key := range keys {
		statusJSON, err := ws.redisClient.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var userStatus UserStatusData
		if err := json.Unmarshal([]byte(statusJSON), &userStatus); err != nil {
			continue
		}

		if userStatus.Status == "online" {
			users = append(users, userStatus)
		}
	}

	return users
}

func (ws *WebSocketService) GetRoomMemberCount(roomID uuid.UUID) int {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	if room, exists := ws.rooms[roomID]; exists {
		return len(room)
	}
	return 0
}
