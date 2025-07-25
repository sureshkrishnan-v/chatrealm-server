package database

import (
        "premium-chat-backend/models"

        "gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) error {
        // Enable UUID extension for PostgreSQL
        if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
                return err
        }

        // Enable gen_random_uuid function
        if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto").Error; err != nil {
                return err
        }

        // Auto-migrate all models
        err := db.AutoMigrate(
                &models.User{},
                &models.UserSession{},
                &models.ChatRoom{},
                &models.ChatRoomMember{},
                &models.Message{},
                &models.MessageReaction{},
                &models.Subscription{},
                &models.BillingHistory{},
        )

        if err != nil {
                return err
        }

        // Add custom indexes for better performance
        if err := addCustomIndexes(db); err != nil {
                return err
        }

        // Create default admin user if it doesn't exist
        if err := createDefaultAdmin(db); err != nil {
                return err
        }

        // Create default chat rooms
        if err := createDefaultRooms(db); err != nil {
                return err
        }

        return nil
}

func addCustomIndexes(db *gorm.DB) error {
        // Index on user email for faster login
        if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)").Error; err != nil {
                return err
        }

        // Index on user username for faster searches
        if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)").Error; err != nil {
                return err
        }

        // Index on user tier for analytics
        if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_tier ON users(tier)").Error; err != nil {
                return err
        }

        // Index on user last_seen_at for active users queries
        if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_last_seen ON users(last_seen_at)").Error; err != nil {
                return err
        }

        // Index on messages for chat room and timestamp
        if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_messages_room_created ON messages(chat_room_id, created_at DESC)").Error; err != nil {
                return err
        }

        // Index on messages user_id for user statistics
        if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_messages_user_id ON messages(user_id)").Error; err != nil {
                return err
        }

        // Index on chat room type for filtering
        if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_chat_rooms_type ON chat_rooms(type)").Error; err != nil {
                return err
        }

        // Index on chat room members for membership queries
        if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_chat_room_members_room_user ON chat_room_members(chat_room_id, user_id)").Error; err != nil {
                return err
        }

        // Index on subscriptions for user lookup
        if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id)").Error; err != nil {
                return err
        }

        // Index on subscriptions status for analytics
        if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status)").Error; err != nil {
                return err
        }

        // Index on billing history for user and date queries
        if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_billing_history_user_paid ON billing_histories(user_id, paid_at)").Error; err != nil {
                return err
        }

        // Index on user sessions for cleanup
        if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_user_sessions_expires ON user_sessions(expires_at)").Error; err != nil {
                return err
        }

        // Index on message reactions for fast lookups
        if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_message_reactions_message ON message_reactions(message_id)").Error; err != nil {
                return err
        }

        // Composite index for message parent/thread queries
        if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_messages_parent_created ON messages(parent_id, created_at DESC) WHERE parent_id IS NOT NULL").Error; err != nil {
                return err
        }

        return nil
}

func createDefaultAdmin(db *gorm.DB) error {
        // Check if admin user already exists
        var count int64
        db.Model(&models.User{}).Where("tier = ?", models.TierAdmin).Count(&count)
        
        if count > 0 {
                return nil // Admin already exists
        }

        // Create default admin user
        admin := &models.User{
                Username:     "admin",
                Email:        "admin@premiumchat.com",
                PasswordHash: "$2a$10$8K8nV5Z5Z5Z5Z5Z5Z5Z5Zu", // Default password: "admin123" (should be changed)
                Tier:         models.TierAdmin,
                IsActive:     true,
        }

        return db.Create(admin).Error
}

func createDefaultRooms(db *gorm.DB) error {
        // Check if default rooms already exist
        var count int64
        db.Model(&models.ChatRoom{}).Count(&count)
        
        if count > 0 {
                return nil // Rooms already exist
        }

        // Get admin user
        var admin models.User
        if err := db.Where("tier = ?", models.TierAdmin).First(&admin).Error; err != nil {
                return err // No admin user found
        }

        // Create default public room
        publicRoom := &models.ChatRoom{
                Name:        "General Chat",
                Description: stringPtr("Welcome to the general chat room! This is a public space for all users."),
                Type:        models.RoomTypePublic,
                CreatedBy:   admin.ID,
                MaxMembers:  1000,
                IsPrivate:   false,
        }

        if err := db.Create(publicRoom).Error; err != nil {
                return err
        }

        // Create default premium room
        premiumRoom := &models.ChatRoom{
                Name:        "Premium Lounge",
                Description: stringPtr("Exclusive chat room for premium subscribers only."),
                Type:        models.RoomTypePrivate,
                CreatedBy:   admin.ID,
                MaxMembers:  500,
                IsPrivate:   false,
        }

        if err := db.Create(premiumRoom).Error; err != nil {
                return err
        }

        // Create default VIP room
        vipRoom := &models.ChatRoom{
                Name:        "VIP Club",
                Description: stringPtr("Ultra-exclusive VIP room with premium features and benefits."),
                Type:        models.RoomTypeVIP,
                CreatedBy:   admin.ID,
                MaxMembers:  100,
                IsPrivate:   false,
        }

        if err := db.Create(vipRoom).Error; err != nil {
                return err
        }

        // Add admin as member of all rooms
        rooms := []*models.ChatRoom{publicRoom, premiumRoom, vipRoom}
        for _, room := range rooms {
                member := &models.ChatRoomMember{
                        ChatRoomID: room.ID,
                        UserID:     admin.ID,
                        Role:       models.RoleAdmin,
                }
                if err := db.Create(member).Error; err != nil {
                        return err
                }
        }

        return nil
}

func stringPtr(s string) *string {
        return &s
}

// Migration functions for schema updates
func MigrateAddFileUploads(db *gorm.DB) error {
        // Add file upload related columns if they don't exist
        if !db.Migrator().HasColumn(&models.Message{}, "file_url") {
                if err := db.Migrator().AddColumn(&models.Message{}, "file_url"); err != nil {
                        return err
                }
        }

        if !db.Migrator().HasColumn(&models.Message{}, "file_name") {
                if err := db.Migrator().AddColumn(&models.Message{}, "file_name"); err != nil {
                        return err
                }
        }

        if !db.Migrator().HasColumn(&models.Message{}, "file_size") {
                if err := db.Migrator().AddColumn(&models.Message{}, "file_size"); err != nil {
                        return err
                }
        }

        return nil
}

func MigrateAddMessageThreads(db *gorm.DB) error {
        // Add thread support columns if they don't exist
        if !db.Migrator().HasColumn(&models.Message{}, "parent_id") {
                if err := db.Migrator().AddColumn(&models.Message{}, "parent_id"); err != nil {
                        return err
                }
        }

        return nil
}

func MigrateAddRoomSettings(db *gorm.DB) error {
        // Room settings are embedded, so they should be created automatically
        // This is a placeholder for future room setting migrations
        return nil
}

func MigrateAddUserTiers(db *gorm.DB) error {
        // Ensure all users have a valid tier
        return db.Model(&models.User{}).Where("tier = '' OR tier IS NULL").Update("tier", models.TierFree).Error
}

func MigrateCleanupExpiredSessions(db *gorm.DB) error {
        // Clean up expired user sessions
        return db.Where("expires_at < NOW()").Delete(&models.UserSession{}).Error
}

func MigrateOptimizeMessageRetention(db *gorm.DB) error {
        // Delete old messages based on retention policy
        // This could be run as a scheduled job
        
        // Delete free user messages older than 7 days
        freeUserCutoff := "NOW() - INTERVAL '7 days'"
        if err := db.Exec(`
                DELETE FROM messages 
                WHERE created_at < ` + freeUserCutoff + ` 
                AND user_id IN (SELECT id FROM users WHERE tier = 'free')
        `).Error; err != nil {
                return err
        }

        // Delete premium user messages older than 1 year
        premiumUserCutoff := "NOW() - INTERVAL '365 days'"
        if err := db.Exec(`
                DELETE FROM messages 
                WHERE created_at < ` + premiumUserCutoff + ` 
                AND user_id IN (SELECT id FROM users WHERE tier = 'premium')
        `).Error; err != nil {
                return err
        }

        return nil
}
