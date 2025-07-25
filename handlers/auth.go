package handlers

import (
        "net/http"
        "premium-chat-backend/models"
        "premium-chat-backend/services"
        "premium-chat-backend/utils"

        "github.com/gin-gonic/gin"
)

type AuthHandler struct {
        authService *services.AuthService
}

type RegisterRequest struct {
        Username string `json:"username" binding:"required,min=3,max=50"`
        Email    string `json:"email" binding:"required,email"`
        Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
        Email    string `json:"email" binding:"required,email"`
        Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
        User         *models.User `json:"user"`
        AccessToken  string       `json:"access_token"`
        RefreshToken string       `json:"refresh_token"`
        ExpiresIn    int          `json:"expires_in"`
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
        return &AuthHandler{
                authService: authService,
        }
}

func (h *AuthHandler) CreateGuestUser(c *gin.Context) {
        var req struct {
                Username string `json:"username" binding:"required,min=3,max=30"`
        }
        
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }
        
        user, accessToken, refreshToken, err := h.authService.CreateGuestUser(req.Username)
        if err != nil {
                if err.Error() == "username already taken" {
                        c.JSON(http.StatusConflict, gin.H{"error": "Username already taken"})
                        return
                }
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create guest user"})
                return
        }
        
        c.JSON(http.StatusCreated, gin.H{
                "user":          user,
                "access_token":  accessToken,
                "refresh_token": refreshToken,
                "expires_in":    3600,
                "user_type":     "guest",
        })
}

func (h *AuthHandler) Register(c *gin.Context) {
        var req RegisterRequest
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": utils.FormatValidationError(err)})
                return
        }

        user, err := h.authService.Register(req.Username, req.Email, req.Password)
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }

        accessToken, refreshToken, err := h.authService.GenerateTokens(user)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
                return
        }

        response := &AuthResponse{
                User:         user,
                AccessToken:  accessToken,
                RefreshToken: refreshToken,
                ExpiresIn:    3600, // 1 hour
        }

        c.JSON(http.StatusCreated, response)
}

func (h *AuthHandler) Login(c *gin.Context) {
        var req LoginRequest
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": utils.FormatValidationError(err)})
                return
        }

        user, err := h.authService.Login(req.Email, req.Password)
        if err != nil {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
                return
        }

        accessToken, refreshToken, err := h.authService.GenerateTokens(user)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
                return
        }

        // Update last seen
        h.authService.UpdateLastSeen(user.ID)

        response := &AuthResponse{
                User:         user,
                AccessToken:  accessToken,
                RefreshToken: refreshToken,
                ExpiresIn:    3600,
        }

        c.JSON(http.StatusOK, response)
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
        type RefreshRequest struct {
                RefreshToken string `json:"refresh_token" binding:"required"`
        }

        var req RefreshRequest
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": utils.FormatValidationError(err)})
                return
        }

        user, accessToken, refreshToken, err := h.authService.RefreshToken(req.RefreshToken)
        if err != nil {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
                return
        }

        response := &AuthResponse{
                User:         user,
                AccessToken:  accessToken,
                RefreshToken: refreshToken,
                ExpiresIn:    3600,
        }

        c.JSON(http.StatusOK, response)
}

func (h *AuthHandler) Logout(c *gin.Context) {
        user, _ := c.Get("user")
        userObj := user.(*models.User)

        if err := h.authService.Logout(userObj.ID); err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
                return
        }

        c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
        user, _ := c.Get("user")
        userObj := user.(*models.User)

        // Get subscription details
        subscription, _ := h.authService.GetUserSubscription(userObj.ID)
        userObj.Subscription = subscription

        c.JSON(http.StatusOK, gin.H{"user": userObj})
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
        type UpdateProfileRequest struct {
                Username string `json:"username,omitempty" binding:"omitempty,min=3,max=50"`
        }

        var req UpdateProfileRequest
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": utils.FormatValidationError(err)})
                return
        }

        user, _ := c.Get("user")
        userObj := user.(*models.User)

        updatedUser, err := h.authService.UpdateProfile(userObj.ID, req.Username)
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }

        c.JSON(http.StatusOK, gin.H{"user": updatedUser})
}
