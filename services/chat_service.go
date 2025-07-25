package services

import (
        "errors"
        "fmt"
        "io"
        "mime/multipart"
        "os"
        "path/filepath"
        "premium-chat-backend/models"
        "strings"
        "time"

        "github.com/go-redis/redis/v8"
        "github.com/google/uuid"
        "golang.org/x/crypto/bcrypt"
        "gorm.io/gorm"
)

type ChatService struct {
        db          *gorm.DB
        redisClient *redis.Client
}

// Request types to avoid circular imports
type CreateRoomRequest struct {
        Name        string `json:"name" binding:"required"`
        Description string `json:"description"`
        Type        string `json:"type" binding:"required,oneof=public private vip"`
        MaxMembers  int    `json:"max_members"`
        IsPrivate   bool   `json:"is_private"`
        Password    string `json:"password"`
}

type SendMessageRequest struct {
        Content     string                `form:"content" binding:"required"`
        MessageType string                `form:"message_type"`
        Type        string                `form:"type"`
        ParentID    *uuid.UUID            `form:"parent_id"`
        File        *multipart.FileHeader `form:"file"`
        Mentions    []uuid.UUID           `form:"mentions"`
}

func NewChatService(db *gorm.DB, redisClient *redis.Client) *ChatService {
        return &ChatService{
                db:          db,
                redisClient: redisClient,
        }
}

func (s *ChatService) SearchUsers(query string, limit int) ([]models.User, error) {
        var users []models.User
        
        if limit <= 0 || limit > 50 {
                limit = 20
        }
        
        err := s.db.Where("username ILIKE ? AND is_active = true", "%"+query+"%").
                Select("id, username, tier, created_at").
                Limit(limit).
                Find(&users).Error
                
        return users, err
}

func (s *ChatService) GetChatRooms(user *models.User, page, limit int, roomType string) ([]models.ChatRoom, int64, error) {
        offset := (page - 1) * limit

        query := s.db.Model(&models.ChatRoom{}).
                Preload("Creator").
                Preload("Members", "user_id = ?", user.ID)

        // Filter by room type if specified
        if roomType != "" {
                query = query.Where("type = ?", roomType)
        }

        // Filter by user access level
        if !user.IsAdmin() {
                accessibleTypes := []models.ChatRoomType{models.RoomTypePublic}
                if user.IsPremium() {
                        accessibleTypes = append(accessibleTypes, models.RoomTypePrivate, models.RoomTypeVIP)
                }
                query = query.Where("type IN ?", accessibleTypes)
        }

        var total int64
        query.Count(&total)

        var rooms []models.ChatRoom
        err := query.Offset(offset).Limit(limit).Find(&rooms).Error

        return rooms, total, err
}

func (s *ChatService) CreateChatRoom(user *models.User, req *CreateRoomRequest) (*models.ChatRoom, error) {
        // Validate max members
        if req.MaxMembers <= 0 {
                req.MaxMembers = 100
        }

        room := &models.ChatRoom{
                Name:        req.Name,
                Description: &req.Description,
                Type:        models.ChatRoomType(req.Type),
                CreatedBy:   user.ID,
                MaxMembers:  req.MaxMembers,
                IsPrivate:   req.IsPrivate,
        }

        // Hash password if provided
        if req.Password != "" {
                hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
                if err != nil {
                        return nil, errors.New("failed to hash password")
                }
                passwordStr := string(hashedPassword)
                room.Password = &passwordStr
        }

        if err := s.db.Create(room).Error; err != nil {
                return nil, errors.New("failed to create chat room")
        }

        // Add creator as admin member
        member := &models.ChatRoomMember{
                ChatRoomID: room.ID,
                UserID:     user.ID,
                Role:       models.RoleAdmin,
                JoinedAt:   time.Now(),
        }

        if err := s.db.Create(member).Error; err != nil {
                return nil, errors.New("failed to add creator as member")
        }

        return room, nil
}

func (s *ChatService) GetChatRoom(roomID, userID uuid.UUID) (*models.ChatRoom, error) {
        var room models.ChatRoom
        if err := s.db.Preload("Creator").Preload("Members.User").First(&room, "id = ?", roomID).Error; err != nil {
                return nil, errors.New("room not found")
        }

        // Check if user is a member (unless admin)
        var user models.User
        if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
                return nil, errors.New("user not found")
        }

        if !user.IsAdmin() {
                var member models.ChatRoomMember
                if err := s.db.Where("chat_room_id = ? AND user_id = ?", roomID, userID).First(&member).Error; err != nil {
                        return nil, errors.New("access denied")
                }
        }

        return &room, nil
}

func (s *ChatService) JoinChatRoom(roomID, userID uuid.UUID, password string) error {
        var room models.ChatRoom
        if err := s.db.First(&room, "id = ?", roomID).Error; err != nil {
                return errors.New("room not found")
        }

        var user models.User
        if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
                return errors.New("user not found")
        }

        // Check if user can access this room type
        if !user.CanAccessRoom(room.Type) {
                return errors.New("insufficient privileges to join this room")
        }

        // Check if already a member
        var existingMember models.ChatRoomMember
        if err := s.db.Where("chat_room_id = ? AND user_id = ?", roomID, userID).First(&existingMember).Error; err == nil {
                return errors.New("already a member of this room")
        }

        // Check if room is full
        var memberCount int64
        s.db.Model(&models.ChatRoomMember{}).Where("chat_room_id = ?", roomID).Count(&memberCount)
        if int(memberCount) >= room.MaxMembers {
                return errors.New("room is full")
        }

        // Check password if required
        if room.Password != nil {
                if password == "" {
                        return errors.New("password required")
                }
                if err := bcrypt.CompareHashAndPassword([]byte(*room.Password), []byte(password)); err != nil {
                        return errors.New("incorrect password")
                }
        }

        // Create membership
        member := &models.ChatRoomMember{
                ChatRoomID: roomID,
                UserID:     userID,
                Role:       models.RoleMember,
                JoinedAt:   time.Now(),
        }

        return s.db.Create(member).Error
}

func (s *ChatService) LeaveChatRoom(roomID, userID uuid.UUID) error {
        // Check if user is the creator
        var room models.ChatRoom
        if err := s.db.First(&room, "id = ?", roomID).Error; err != nil {
                return errors.New("room not found")
        }

        if room.CreatedBy == userID {
                return errors.New("room creator cannot leave the room")
        }

        // Remove membership
        result := s.db.Where("chat_room_id = ? AND user_id = ?", roomID, userID).Delete(&models.ChatRoomMember{})
        if result.RowsAffected == 0 {
                return errors.New("not a member of this room")
        }

        return nil
}

func (s *ChatService) GetMessages(roomID, userID uuid.UUID, page, limit int) ([]models.Message, int64, error) {
        // Check if user is a member
        var member models.ChatRoomMember
        if err := s.db.Where("chat_room_id = ? AND user_id = ?", roomID, userID).First(&member).Error; err != nil {
                return nil, 0, errors.New("access denied")
        }

        offset := (page - 1) * limit

        query := s.db.Model(&models.Message{}).
                Preload("User").
                Preload("Reactions.User").
                Where("chat_room_id = ?", roomID).
                Order("created_at DESC")

        var total int64
        query.Count(&total)

        var messages []models.Message
        err := query.Offset(offset).Limit(limit).Find(&messages).Error

        return messages, total, err
}

func (s *ChatService) SendMessage(roomID, userID uuid.UUID, req *SendMessageRequest) (*models.Message, error) {
        // Check if user is a member
        var member models.ChatRoomMember
        if err := s.db.Where("chat_room_id = ? AND user_id = ?", roomID, userID).First(&member).Error; err != nil {
                return nil, errors.New("access denied")
        }

        if member.IsMuted {
                return nil, errors.New("you are muted in this room")
        }

        // Get room settings
        var room models.ChatRoom
        if err := s.db.First(&room, "id = ?", roomID).Error; err != nil {
                return nil, errors.New("room not found")
        }

        // Check if threads are allowed for thread replies
        if req.ParentID != nil && !room.Settings.AllowThreads {
                return nil, errors.New("thread replies are not allowed in this room")
        }

        // Validate mentions
        if len(req.Mentions) > 0 && !room.Settings.AllowMentions {
                return nil, errors.New("mentions are not allowed in this room")
        }

        message := &models.Message{
                ChatRoomID: roomID,
                UserID:     userID,
                Content:    req.Content,
                Type:       models.MessageType(req.Type),
                ParentID:   req.ParentID,
                Mentions:   req.Mentions,
        }

        if err := s.db.Create(message).Error; err != nil {
                return nil, errors.New("failed to send message")
        }

        // Load user data
        s.db.Preload("User").First(message, message.ID)

        return message, nil
}

func (s *ChatService) ReactToMessage(messageID, userID uuid.UUID, emoji string) (*models.MessageReaction, error) {
        // Check if message exists and user has access
        var message models.Message
        if err := s.db.Preload("ChatRoom").First(&message, "id = ?", messageID).Error; err != nil {
                return nil, errors.New("message not found")
        }

        // Check if user is a member of the room
        var member models.ChatRoomMember
        if err := s.db.Where("chat_room_id = ? AND user_id = ?", message.ChatRoomID, userID).First(&member).Error; err != nil {
                return nil, errors.New("access denied")
        }

        // Check if reactions are allowed
        if !message.ChatRoom.Settings.AllowReactions {
                return nil, errors.New("reactions are not allowed in this room")
        }

        // Check if user already reacted with this emoji
        var existingReaction models.MessageReaction
        if err := s.db.Where("message_id = ? AND user_id = ? AND emoji = ?", messageID, userID, emoji).First(&existingReaction).Error; err == nil {
                // Remove existing reaction
                s.db.Delete(&existingReaction)
                return nil, nil
        }

        // Create new reaction
        reaction := &models.MessageReaction{
                MessageID: messageID,
                UserID:    userID,
                Emoji:     emoji,
        }

        if err := s.db.Create(reaction).Error; err != nil {
                return nil, errors.New("failed to add reaction")
        }

        s.db.Preload("User").First(reaction, reaction.ID)

        return reaction, nil
}

func (s *ChatService) UploadFile(file multipart.File, header *multipart.FileHeader, roomID, userID uuid.UUID) (string, error) {
        // Check if user is a member
        var member models.ChatRoomMember
        if err := s.db.Where("chat_room_id = ? AND user_id = ?", roomID, userID).First(&member).Error; err != nil {
                return "", errors.New("access denied")
        }

        // Get room settings
        var room models.ChatRoom
        if err := s.db.First(&room, "id = ?", roomID).Error; err != nil {
                return "", errors.New("room not found")
        }

        // Check file size
        if header.Size > room.Settings.MaxFileSize {
                return "", fmt.Errorf("file size exceeds limit of %d bytes", room.Settings.MaxFileSize)
        }

        // Check file type
        fileExt := strings.ToLower(filepath.Ext(header.Filename))
        contentType := header.Header.Get("Content-Type")

        if strings.HasPrefix(contentType, "image/") && !room.Settings.AllowImageUploads {
                return "", errors.New("image uploads are not allowed in this room")
        }
        if strings.HasPrefix(contentType, "audio/") && !room.Settings.AllowAudioUploads {
                return "", errors.New("audio uploads are not allowed in this room")
        }
        if strings.HasPrefix(contentType, "video/") && !room.Settings.AllowVideoUploads {
                return "", errors.New("video uploads are not allowed in this room")
        }
        if !room.Settings.AllowFileUploads && !strings.HasPrefix(contentType, "image/") {
                return "", errors.New("file uploads are not allowed in this room")
        }

        // Create uploads directory if it doesn't exist
        uploadDir := "uploads"
        if err := os.MkdirAll(uploadDir, 0755); err != nil {
                return "", errors.New("failed to create upload directory")
        }

        // Generate unique filename
        filename := fmt.Sprintf("%s_%s%s", uuid.New().String(), time.Now().Format("20060102150405"), fileExt)
        filePath := filepath.Join(uploadDir, filename)

        // Create the file
        dst, err := os.Create(filePath)
        if err != nil {
                return "", errors.New("failed to create file")
        }
        defer dst.Close()

        // Copy the uploaded file to the destination file
        if _, err := io.Copy(dst, file); err != nil {
                return "", errors.New("failed to save file")
        }

        // Return the file URL (in production, this would be a CDN URL)
        fileURL := fmt.Sprintf("/uploads/%s", filename)
        return fileURL, nil
}

func (s *ChatService) DeleteMessage(messageID, userID uuid.UUID) error {
        var message models.Message
        if err := s.db.First(&message, "id = ?", messageID).Error; err != nil {
                return errors.New("message not found")
        }

        // Check if user is the author or admin
        var user models.User
        if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
                return errors.New("user not found")
        }

        if message.UserID != userID && !user.IsAdmin() {
                return errors.New("access denied")
        }

        // Delete reactions first
        s.db.Where("message_id = ?", messageID).Delete(&models.MessageReaction{})

        // Delete the message
        return s.db.Delete(&message).Error
}

func (s *ChatService) EditMessage(messageID, userID uuid.UUID, content string) (*models.Message, error) {
        var message models.Message
        if err := s.db.First(&message, "id = ?", messageID).Error; err != nil {
                return nil, errors.New("message not found")
        }

        // Check if user is the author
        if message.UserID != userID {
                return nil, errors.New("can only edit your own messages")
        }

        // Check if message is less than 15 minutes old
        if time.Since(message.CreatedAt) > time.Minute*15 {
                return nil, errors.New("message is too old to edit")
        }

        // Update message
        now := time.Now()
        message.Content = content
        message.IsEdited = true
        message.EditedAt = &now

        if err := s.db.Save(&message).Error; err != nil {
                return nil, errors.New("failed to edit message")
        }

        s.db.Preload("User").First(&message, message.ID)

        return &message, nil
}
