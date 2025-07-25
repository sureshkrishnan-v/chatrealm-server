package handlers

import (
	"net/http"
	"premium-chat-backend/models"
	"premium-chat-backend/services"
	"premium-chat-backend/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db               *gorm.DB
	analyticsService *services.AnalyticsService
}

func NewAdminHandler(db *gorm.DB, analyticsService *services.AnalyticsService) *AdminHandler {
	return &AdminHandler{
		db:               db,
		analyticsService: analyticsService,
	}
}

func (h *AdminHandler) GetUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	tier := c.Query("tier")
	search := c.Query("search")

	offset := (page - 1) * limit

	query := h.db.Model(&models.User{}).Preload("Subscription")

	if tier != "" {
		query = query.Where("tier = ?", tier)
	}

	if search != "" {
		query = query.Where("username ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	var users []models.User
	if err := query.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *AdminHandler) UpdateUserTier(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	type UpdateTierRequest struct {
		Tier models.UserTier `json:"tier" binding:"required"`
	}

	var req UpdateTierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.FormatValidationError(err)})
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.Tier = req.Tier
	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user tier"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *AdminHandler) GetAnalytics(c *gin.Context) {
	analytics, err := h.analyticsService.GetSystemAnalytics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch analytics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"analytics": analytics})
}

func (h *AdminHandler) GetAllRooms(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	roomType := c.Query("type")

	offset := (page - 1) * limit

	query := h.db.Model(&models.ChatRoom{}).Preload("Creator").Preload("Members")

	if roomType != "" {
		query = query.Where("type = ?", roomType)
	}

	var total int64
	query.Count(&total)

	var rooms []models.ChatRoom
	if err := query.Offset(offset).Limit(limit).Find(&rooms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rooms"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rooms": rooms,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *AdminHandler) DeleteRoom(c *gin.Context) {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	// Delete all messages in the room first
	if err := h.db.Where("chat_room_id = ?", roomID).Delete(&models.Message{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete room messages"})
		return
	}

	// Delete all room members
	if err := h.db.Where("chat_room_id = ?", roomID).Delete(&models.ChatRoomMember{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete room members"})
		return
	}

	// Delete the room
	if err := h.db.Delete(&models.ChatRoom{}, roomID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete room"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Room deleted successfully"})
}

func (h *AdminHandler) GetMessage(c *gin.Context) {
	messageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	var message models.Message
	if err := h.db.Preload("User").Preload("ChatRoom").Preload("Reactions.User").First(&message, "id = ?", messageID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": message})
}

func (h *AdminHandler) DeleteMessage(c *gin.Context) {
	messageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	// Delete message reactions first
	if err := h.db.Where("message_id = ?", messageID).Delete(&models.MessageReaction{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete message reactions"})
		return
	}

	// Delete the message
	if err := h.db.Delete(&models.Message{}, messageID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message deleted successfully"})
}

func (h *AdminHandler) GetUserStats(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	stats, err := h.analyticsService.GetUserStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

func (h *AdminHandler) BanUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	type BanRequest struct {
		Reason string `json:"reason"`
	}

	var req BanRequest
	c.ShouldBindJSON(&req)

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.IsActive = false
	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to ban user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User banned successfully"})
}

func (h *AdminHandler) UnbanUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.IsActive = true
	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unban user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User unbanned successfully"})
}
