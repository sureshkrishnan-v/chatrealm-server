package main

import (
        "log"
        "premium-chat-backend/config"
        "premium-chat-backend/database"
        "premium-chat-backend/handlers"
        "premium-chat-backend/middleware"
        "premium-chat-backend/services"
        "premium-chat-backend/utils"

        "github.com/gin-gonic/gin"
)

func main() {
        // Initialize logger
        utils.InitLogger()

        // Load configuration
        cfg := config.Load()

        // Initialize database
        db, err := database.Initialize(cfg.DatabaseURL)
        if err != nil {
                log.Fatal("Failed to initialize database:", err)
        }

        // Run migrations
        if err := database.RunMigrations(db); err != nil {
                log.Fatal("Failed to run migrations:", err)
        }

        // Initialize Redis
        redisClient := database.InitRedis(cfg.RedisURL)

        // Initialize services
        authService := services.NewAuthService(db, redisClient)
        chatService := services.NewChatService(db, redisClient)
        subscriptionService := services.NewSubscriptionService(db, cfg.StripeSecretKey)
        websocketService := services.NewWebSocketService(db, redisClient)
        analyticsService := services.NewAnalyticsService(db, redisClient)

        // Initialize handlers
        authHandler := handlers.NewAuthHandler(authService)
        chatHandler := handlers.NewChatHandler(chatService)
        websocketHandler := handlers.NewWebSocketHandler(websocketService)
        subscriptionHandler := handlers.NewSubscriptionHandler(subscriptionService)
        adminHandler := handlers.NewAdminHandler(db, analyticsService)

        // Initialize Gin router
        router := gin.Default()

        // Apply middleware
        router.Use(middleware.CORS())
        router.Use(middleware.Logger())

        // Root endpoint - API documentation
        router.GET("/", func(c *gin.Context) {
                c.JSON(200, gin.H{
                        "service": "Premium Chat Backend API",
                        "version": "1.0.0",
                        "status": "running",
                        "endpoints": gin.H{
                                "health": "/health",
                                "api_base": "/api/v1",
                                "auth": "/api/v1/auth/*",
                                "guest_auth": "/api/v1/auth/guest",
                                "chat": "/api/v1/chat/*",
                                "user_search": "/api/v1/chat/search/users",
                                "websocket": "/api/v1/ws",
                                "subscription": "/api/v1/subscription/*",
                                "admin": "/api/v1/admin/*",
                        },
                        "features": []string{
                                "User Authentication (JWT)",
                                "Anonymous Guest Users",
                                "Username Search",
                                "Tier-based Access Control",
                                "Real-time Chat via WebSocket",
                                "Stripe Payment Integration",
                                "File Upload Support",
                                "Admin Panel",
                                "Rate Limiting",
                        },
                })
        })

        // Health check endpoint
        router.GET("/health", func(c *gin.Context) {
                c.JSON(200, gin.H{"status": "healthy"})
        })

        // API routes
        v1 := router.Group("/api/v1")
        {
                // Authentication routes
                auth := v1.Group("/auth")
                {
                        auth.POST("/register", authHandler.Register)
                        auth.POST("/login", authHandler.Login)
                        auth.POST("/guest", authHandler.CreateGuestUser)
                        auth.POST("/refresh", authHandler.RefreshToken)
                        auth.POST("/logout", middleware.Auth(authService), authHandler.Logout)
                }

                // Chat routes
                chat := v1.Group("/chat")
                chat.Use(middleware.Auth(authService))
                {
                        chat.GET("/rooms", chatHandler.GetChatRooms)
                        chat.POST("/rooms", chatHandler.CreateChatRoom)
                        chat.GET("/rooms/:id", chatHandler.GetChatRoom)
                        chat.POST("/rooms/:id/join", chatHandler.JoinChatRoom)
                        chat.POST("/rooms/:id/leave", chatHandler.LeaveChatRoom)
                        chat.GET("/rooms/:id/messages", chatHandler.GetMessages)
                        chat.POST("/rooms/:id/messages", middleware.RateLimit(), chatHandler.SendMessage)
                        chat.POST("/messages/:id/react", chatHandler.ReactToMessage)
                        chat.POST("/upload", chatHandler.UploadFile)
                        chat.GET("/search/users", chatHandler.SearchUsers)
                }

                // WebSocket endpoint
                v1.GET("/ws", middleware.Auth(authService), websocketHandler.HandleWebSocket)

                // Subscription routes
                subscription := v1.Group("/subscription")
                subscription.Use(middleware.Auth(authService))
                {
                        subscription.GET("/status", subscriptionHandler.GetSubscriptionStatus)
                        subscription.POST("/create", subscriptionHandler.CreateSubscription)
                        subscription.POST("/cancel", subscriptionHandler.CancelSubscription)
                        subscription.POST("/webhook", subscriptionHandler.HandleWebhook)
                }

                // Admin routes
                admin := v1.Group("/admin")
                admin.Use(middleware.Auth(authService))
                admin.Use(middleware.RequireAdmin())
                {
                        admin.GET("/users", adminHandler.GetUsers)
                        admin.PUT("/users/:id/tier", adminHandler.UpdateUserTier)
                        admin.GET("/analytics", adminHandler.GetAnalytics)
                        admin.GET("/rooms", adminHandler.GetAllRooms)
                        admin.DELETE("/rooms/:id", adminHandler.DeleteRoom)
                        admin.GET("/messages/:id", adminHandler.GetMessage)
                        admin.DELETE("/messages/:id", adminHandler.DeleteMessage)
                }
        }

        // Start WebSocket service
        go websocketService.Start()

        // Start server
        log.Printf("Server starting on port %s", cfg.Port)
        if err := router.Run("0.0.0.0:" + cfg.Port); err != nil {
                log.Fatal("Failed to start server:", err)
        }
}
