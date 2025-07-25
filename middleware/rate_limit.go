package middleware

import (
	"fmt"
	"net/http"
	"premium-chat-backend/models"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiters map[string]*rate.Limiter
}

var rateLimiter = &RateLimiter{
	limiters: make(map[string]*rate.Limiter),
}

func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		userObj := user.(*models.User)
		userID := userObj.ID.String()

		// Get or create rate limiter for this user
		limiter := rateLimiter.getLimiter(userID, userObj.GetRateLimit())

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"retry_after": int(time.Minute.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (rl *RateLimiter) getLimiter(userID string, limit int) *rate.Limiter {
	if limiter, exists := rl.limiters[userID]; exists {
		return limiter
	}

	// Create new limiter
	// For admin users (limit = 0), set a very high limit
	if limit == 0 {
		limit = 1000
	}

	limiter := rate.NewLimiter(rate.Every(time.Minute/time.Duration(limit)), limit)
	rl.limiters[userID] = limiter

	// Clean up old limiters periodically
	go func() {
		time.Sleep(time.Hour)
		delete(rl.limiters, userID)
	}()

	return limiter
}

func GlobalRateLimit(requestsPerMinute int) gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Every(time.Minute/time.Duration(requestsPerMinute)), requestsPerMinute)

	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Global rate limit exceeded",
				"retry_after": int(time.Minute.Seconds()),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func IPRateLimit(requestsPerMinute int) gin.HandlerFunc {
	limiters := make(map[string]*rate.Limiter)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		
		limiter, exists := limiters[ip]
		if !exists {
			limiter = rate.NewLimiter(rate.Every(time.Minute/time.Duration(requestsPerMinute)), requestsPerMinute)
			limiters[ip] = limiter

			// Clean up old limiters
			go func() {
				time.Sleep(time.Hour)
				delete(limiters, ip)
			}()
		}

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": fmt.Sprintf("IP rate limit exceeded for %s", ip),
				"retry_after": int(time.Minute.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
