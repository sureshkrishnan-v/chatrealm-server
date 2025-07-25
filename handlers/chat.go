package handlers

import (
        "net/http"
        "premium-chat-backend/models"
        "premium-chat-backend/services"
        "premium-chat-backend/utils"
        "strconv"

        "github.com/gin-gonic/gin"
        "github.com/google/uuid"
)

type ChatHandler struct {
        chatService *services.ChatService
}

type CreateRoomRequest struct {
        Name        string                  `json:"name" binding:"required,min=1,max=100"`
        Description string                  `json:"description,omitempty"`
        Type        models.ChatRoomType     `json:"type" binding:"required"`
        MaxMembers  int                     `json:"max_members,omitempty"`
        IsPrivate   bool                    `json:"is_private,omitempty"`
        Password    string                  `json:"password,omitempty"`
}

type SendMessageRequest struct {
        Content  string           `json:"content" binding:"required"`
        Type     models.MessageType `json:"type,omitempty"`
        ParentID *uuid.UUID       `json:"parent_id,omitempty"`
        Mentions []uuid.UUID      `json:"mentions,omitempty"`
}

func NewChatHandler(chatService *services.ChatService) *ChatHandler {
        return &ChatHandler{
                chatService: chatService,
        }
}

// Convert handler request to service request
func (req *CreateRoomRequest) toServiceRequest() *services.CreateRoomRequest {
        return &services.CreateRoomRequest{
                Name:        req.Name,
                Description: req.Description,
                Type:        string(req.Type),
                MaxMembers:  req.MaxMembers,
                IsPrivate:   req.IsPrivate,
                Password:    req.Password,
        }
}

func (req *SendMessageRequest) toServiceRequest() *services.SendMessageRequest {
        return &services.SendMessageRequest{
                Content:     req.Content,
                MessageType: "",
                Type:        string(req.Type),
                ParentID:    req.ParentID,
                File:        nil,
                Mentions:    req.Mentions,
        }
}

func (h *ChatHandler) SearchUsers(c *gin.Context) {
        query := c.Query("q")
        if query == "" {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
                return
        }
        
        limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
        
        users, err := h.chatService.SearchUsers(query, limit)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search users"})
                return
        }
        
        c.JSON(http.StatusOK, gin.H{
                "users": users,
                "count": len(users),
        })
}

func (h *ChatHandler) GetChatRooms(c *gin.Context) {
        user, _ := c.Get("user")
        userObj := user.(*models.User)

        page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
        limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
        roomType := c.Query("type")

        rooms, total, err := h.chatService.GetChatRooms(userObj, page, limit, roomType)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chat rooms"})
                return
        }

        c.JSON(http.StatusOK, gin.H{
                "rooms": rooms,
                "total": total,
                "page":  page,
                "limit": limit,
        })
}

func (h *ChatHandler) CreateChatRoom(c *gin.Context) {
        var req CreateRoomRequest
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": utils.FormatValidationError(err)})
                return
        }

        user, _ := c.Get("user")
        userObj := user.(*models.User)

        // Check if user can create this type of room
        if !userObj.CanAccessRoom(req.Type) {
                c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient privileges to create this room type"})
                return
        }

        room, err := h.chatService.CreateChatRoom(userObj, req.toServiceRequest())
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }

        c.JSON(http.StatusCreated, gin.H{"room": room})
}

func (h *ChatHandler) GetChatRoom(c *gin.Context) {
        roomID, err := uuid.Parse(c.Param("id"))
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
                return
        }

        user, _ := c.Get("user")
        userObj := user.(*models.User)

        room, err := h.chatService.GetChatRoom(roomID, userObj.ID)
        if err != nil {
                c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
                return
        }

        c.JSON(http.StatusOK, gin.H{"room": room})
}

func (h *ChatHandler) JoinChatRoom(c *gin.Context) {
        roomID, err := uuid.Parse(c.Param("id"))
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
                return
        }

        user, _ := c.Get("user")
        userObj := user.(*models.User)

        type JoinRoomRequest struct {
                Password string `json:"password,omitempty"`
        }

        var req JoinRoomRequest
        c.ShouldBindJSON(&req)

        if err := h.chatService.JoinChatRoom(roomID, userObj.ID, req.Password); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }

        c.JSON(http.StatusOK, gin.H{"message": "Successfully joined room"})
}

func (h *ChatHandler) LeaveChatRoom(c *gin.Context) {
        roomID, err := uuid.Parse(c.Param("id"))
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
                return
        }

        user, _ := c.Get("user")
        userObj := user.(*models.User)

        if err := h.chatService.LeaveChatRoom(roomID, userObj.ID); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }

        c.JSON(http.StatusOK, gin.H{"message": "Successfully left room"})
}

func (h *ChatHandler) GetMessages(c *gin.Context) {
        roomID, err := uuid.Parse(c.Param("id"))
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
                return
        }

        user, _ := c.Get("user")
        userObj := user.(*models.User)

        page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
        limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

        messages, total, err := h.chatService.GetMessages(roomID, userObj.ID, page, limit)
        if err != nil {
                c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
                return
        }

        c.JSON(http.StatusOK, gin.H{
                "messages": messages,
                "total":    total,
                "page":     page,
                "limit":    limit,
        })
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
        roomID, err := uuid.Parse(c.Param("id"))
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
                return
        }

        var req SendMessageRequest
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": utils.FormatValidationError(err)})
                return
        }

        user, _ := c.Get("user")
        userObj := user.(*models.User)

        message, err := h.chatService.SendMessage(roomID, userObj.ID, req.toServiceRequest())
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }

        c.JSON(http.StatusCreated, gin.H{"message": message})
}

func (h *ChatHandler) ReactToMessage(c *gin.Context) {
        messageID, err := uuid.Parse(c.Param("id"))
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
                return
        }

        user, _ := c.Get("user")
        userObj := user.(*models.User)

        // Premium feature check
        if !userObj.IsPremium() {
                c.JSON(http.StatusForbidden, gin.H{"error": "Message reactions are a premium feature"})
                return
        }

        type ReactRequest struct {
                Emoji string `json:"emoji" binding:"required"`
        }

        var req ReactRequest
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": utils.FormatValidationError(err)})
                return
        }

        reaction, err := h.chatService.ReactToMessage(messageID, userObj.ID, req.Emoji)
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }

        c.JSON(http.StatusCreated, gin.H{"reaction": reaction})
}

func (h *ChatHandler) UploadFile(c *gin.Context) {
        user, _ := c.Get("user")
        userObj := user.(*models.User)

        // Premium feature check
        if !userObj.IsPremium() {
                c.JSON(http.StatusForbidden, gin.H{"error": "File uploads are a premium feature"})
                return
        }

        file, header, err := c.Request.FormFile("file")
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
                return
        }
        defer file.Close()

        roomID, err := uuid.Parse(c.PostForm("room_id"))
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
                return
        }

        fileURL, err := h.chatService.UploadFile(file, header, roomID, userObj.ID)
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }

        c.JSON(http.StatusOK, gin.H{
                "file_url": fileURL,
                "file_name": header.Filename,
                "file_size": header.Size,
        })
}

func (h *ChatHandler) DeleteMessage(c *gin.Context) {
        messageID, err := uuid.Parse(c.Param("id"))
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
                return
        }

        user, _ := c.Get("user")
        userObj := user.(*models.User)

        if err := h.chatService.DeleteMessage(messageID, userObj.ID); err != nil {
                c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
                return
        }

        c.JSON(http.StatusOK, gin.H{"message": "Message deleted successfully"})
}

func (h *ChatHandler) EditMessage(c *gin.Context) {
        messageID, err := uuid.Parse(c.Param("id"))
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
                return
        }

        user, _ := c.Get("user")
        userObj := user.(*models.User)

        // Premium feature check
        if !userObj.IsPremium() {
                c.JSON(http.StatusForbidden, gin.H{"error": "Message editing is a premium feature"})
                return
        }

        type EditMessageRequest struct {
                Content string `json:"content" binding:"required"`
        }

        var req EditMessageRequest
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": utils.FormatValidationError(err)})
                return
        }

        message, err := h.chatService.EditMessage(messageID, userObj.ID, req.Content)
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }

        c.JSON(http.StatusOK, gin.H{"message": message})
}
