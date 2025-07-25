package handlers

import (
	"log"
	"net/http"
	"premium-chat-backend/models"
	"premium-chat-backend/services"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WebSocketHandler struct {
	websocketService *services.WebSocketService
	upgrader         websocket.Upgrader
}

func NewWebSocketHandler(websocketService *services.WebSocketService) *WebSocketHandler {
	return &WebSocketHandler{
		websocketService: websocketService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userObj := user.(*models.User)

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Register the connection
	client := h.websocketService.RegisterClient(conn, userObj)
	defer h.websocketService.UnregisterClient(client)

	// Handle messages in separate goroutines
	go h.handleClientWrite(client)
	h.handleClientRead(client)
}

func (h *WebSocketHandler) handleClientRead(client *services.Client) {
	defer func() {
		h.websocketService.UnregisterClient(client)
		client.Conn.Close()
	}()

	for {
		var msg services.WebSocketMessage
		if err := client.Conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Process the message
		h.websocketService.HandleMessage(client, &msg)
	}
}

func (h *WebSocketHandler) handleClientWrite(client *services.Client) {
	defer client.Conn.Close()

	for {
		select {
		case message, ok := <-client.Send:
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.Conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-client.Done:
			return
		}
	}
}
