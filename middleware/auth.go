package middleware

import (
        "net/http"
        "premium-chat-backend/models"
        "premium-chat-backend/services"
        "premium-chat-backend/utils"
        "strings"

        "github.com/gin-gonic/gin"
)

func Auth(authService *services.AuthService) gin.HandlerFunc {
        return func(c *gin.Context) {
                authHeader := c.GetHeader("Authorization")
                if authHeader == "" {
                        c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
                        c.Abort()
                        return
                }

                tokenParts := strings.Split(authHeader, " ")
                if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
                        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
                        c.Abort()
                        return
                }

                token := tokenParts[1]
                claims, err := utils.ValidateJWT(token)
                if err != nil {
                        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
                        c.Abort()
                        return
                }

                // Get user from database to ensure they still exist and are active
                user, err := authService.GetUserByID(claims.UserID)
                if err != nil {
                        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
                        c.Abort()
                        return
                }

                if !user.IsActive {
                        c.JSON(http.StatusUnauthorized, gin.H{"error": "Account deactivated"})
                        c.Abort()
                        return
                }

                // Store user in context
                c.Set("user", user)
                c.Set("user_id", user.ID)
                c.Set("user_tier", user.Tier)

                c.Next()
        }
}

func RequireAdmin() gin.HandlerFunc {
        return func(c *gin.Context) {
                user, exists := c.Get("user")
                if !exists {
                        c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
                        c.Abort()
                        return
                }

                if !user.(*models.User).IsAdmin() {
                        c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
                        c.Abort()
                        return
                }

                c.Next()
        }
}

func RequirePremium() gin.HandlerFunc {
        return func(c *gin.Context) {
                user, exists := c.Get("user")
                if !exists {
                        c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
                        c.Abort()
                        return
                }

                if !user.(*models.User).IsPremium() {
                        c.JSON(http.StatusForbidden, gin.H{"error": "Premium subscription required"})
                        c.Abort()
                        return
                }

                c.Next()
        }
}
