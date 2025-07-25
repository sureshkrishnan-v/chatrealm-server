package services

import (
        "errors"
        "premium-chat-backend/models"
        "premium-chat-backend/utils"
        "time"

        "github.com/go-redis/redis/v8"
        "github.com/google/uuid"
        "golang.org/x/crypto/bcrypt"
        "gorm.io/gorm"
)

type AuthService struct {
        db          *gorm.DB
        redisClient *redis.Client
}

func NewAuthService(db *gorm.DB, redisClient *redis.Client) *AuthService {
        return &AuthService{
                db:          db,
                redisClient: redisClient,
        }
}

func (s *AuthService) CreateGuestUser(username string) (*models.User, string, string, error) {
        // Check if username is taken
        var existingUser models.User
        err := s.db.Where("username = ?", username).First(&existingUser).Error
        if err == nil {
                return nil, "", "", errors.New("username already taken")
        } else if !errors.Is(err, gorm.ErrRecordNotFound) {
                return nil, "", "", err
        }

        // Create guest user with no email or password
        user := &models.User{
                Username: username,
                Email:    "", // No email for guest users
                Tier:     models.TierFree,
                IsActive: true,
        }

        if err := s.db.Create(user).Error; err != nil {
                return nil, "", "", err
        }

        // Generate tokens
        accessToken, refreshToken, err := s.GenerateTokens(user)
        if err != nil {
                return nil, "", "", err
        }

        return user, accessToken, refreshToken, nil
}

func (s *AuthService) Register(username, email, password string) (*models.User, error) {
        // Check if user already exists
        var existingUser models.User
        if err := s.db.Where("email = ? OR username = ?", email, username).First(&existingUser).Error; err == nil {
                return nil, errors.New("user already exists")
        }

        // Hash password
        hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
        if err != nil {
                return nil, errors.New("failed to hash password")
        }

        // Create user
        user := &models.User{
                Username:     username,
                Email:        email,
                PasswordHash: string(hashedPassword),
                Tier:         models.TierFree,
                IsActive:     true,
        }

        if err := s.db.Create(user).Error; err != nil {
                return nil, errors.New("failed to create user")
        }

        return user, nil
}

func (s *AuthService) Login(email, password string) (*models.User, error) {
        var user models.User
        if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
                return nil, errors.New("invalid credentials")
        }

        if !user.IsActive {
                return nil, errors.New("account deactivated")
        }

        // Check password
        if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
                return nil, errors.New("invalid credentials")
        }

        return &user, nil
}

func (s *AuthService) GenerateTokens(user *models.User) (string, string, error) {
        // Generate access token (1 hour)
        accessToken, err := utils.GenerateJWT(user.ID, user.Tier, time.Hour)
        if err != nil {
                return "", "", err
        }

        // Generate refresh token (7 days)
        refreshToken, err := utils.GenerateJWT(user.ID, user.Tier, time.Hour*24*7)
        if err != nil {
                return "", "", err
        }

        // Store refresh token in database
        session := &models.UserSession{
                UserID:    user.ID,
                Token:     refreshToken,
                ExpiresAt: time.Now().Add(time.Hour * 24 * 7),
        }

        if err := s.db.Create(session).Error; err != nil {
                return "", "", err
        }

        return accessToken, refreshToken, nil
}

func (s *AuthService) RefreshToken(refreshToken string) (*models.User, string, string, error) {
        // Validate refresh token
        claims, err := utils.ValidateJWT(refreshToken)
        if err != nil {
                return nil, "", "", errors.New("invalid refresh token")
        }

        // Check if session exists and is valid
        var session models.UserSession
        if err := s.db.Where("token = ? AND expires_at > ?", refreshToken, time.Now()).First(&session).Error; err != nil {
                return nil, "", "", errors.New("invalid refresh token")
        }

        // Get user
        var user models.User
        if err := s.db.First(&user, "id = ?", claims.UserID).Error; err != nil {
                return nil, "", "", errors.New("user not found")
        }

        if !user.IsActive {
                return nil, "", "", errors.New("account deactivated")
        }

        // Delete old session
        s.db.Delete(&session)

        // Generate new tokens
        accessToken, newRefreshToken, err := s.GenerateTokens(&user)
        if err != nil {
                return nil, "", "", err
        }

        return &user, accessToken, newRefreshToken, nil
}

func (s *AuthService) Logout(userID uuid.UUID) error {
        // Delete all user sessions
        return s.db.Where("user_id = ?", userID).Delete(&models.UserSession{}).Error
}

func (s *AuthService) GetUserByID(userID uuid.UUID) (*models.User, error) {
        var user models.User
        if err := s.db.Preload("Subscription").First(&user, "id = ?", userID).Error; err != nil {
                return nil, err
        }
        return &user, nil
}

func (s *AuthService) UpdateLastSeen(userID uuid.UUID) error {
        now := time.Now()
        return s.db.Model(&models.User{}).Where("id = ?", userID).Update("last_seen_at", &now).Error
}

func (s *AuthService) UpdateProfile(userID uuid.UUID, username string) (*models.User, error) {
        var user models.User
        if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
                return nil, errors.New("user not found")
        }

        if username != "" {
                // Check if username is already taken
                var existingUser models.User
                if err := s.db.Where("username = ? AND id != ?", username, userID).First(&existingUser).Error; err == nil {
                        return nil, errors.New("username already taken")
                }
                user.Username = username
        }

        if err := s.db.Save(&user).Error; err != nil {
                return nil, errors.New("failed to update profile")
        }

        return &user, nil
}

func (s *AuthService) GetUserSubscription(userID uuid.UUID) (*models.Subscription, error) {
        var subscription models.Subscription
        if err := s.db.Where("user_id = ?", userID).First(&subscription).Error; err != nil {
                return nil, err
        }
        return &subscription, nil
}

func (s *AuthService) ValidateUserTier(userID uuid.UUID, requiredTier models.UserTier) error {
        user, err := s.GetUserByID(userID)
        if err != nil {
                return err
        }

        switch requiredTier {
        case models.TierPremium:
                if !user.IsPremium() {
                        return errors.New("premium subscription required")
                }
        case models.TierAdmin:
                if !user.IsAdmin() {
                        return errors.New("admin privileges required")
                }
        }

        return nil
}
