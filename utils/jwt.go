package utils

import (
	"errors"
	"os"
	"premium-chat-backend/models"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID uuid.UUID       `json:"user_id"`
	Tier   models.UserTier `json:"tier"`
	jwt.RegisteredClaims
}

func GenerateJWT(userID uuid.UUID, tier models.UserTier, duration time.Duration) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your-secret-key" // Fallback for development
	}

	now := time.Now()
	claims := &Claims{
		UserID: userID,
		Tier:   tier,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "premium-chat-backend",
			Subject:   userID.String(),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateJWT(tokenString string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your-secret-key" // Fallback for development
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	// Check if token is expired
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, errors.New("token expired")
	}

	// Check if token is not yet valid
	if claims.NotBefore != nil && claims.NotBefore.Time.After(time.Now()) {
		return nil, errors.New("token not yet valid")
	}

	return claims, nil
}

func RefreshJWT(tokenString string) (string, error) {
	claims, err := ValidateJWT(tokenString)
	if err != nil {
		return "", err
	}

	// Check if token is close to expiration (within 5 minutes)
	if claims.ExpiresAt != nil && time.Until(claims.ExpiresAt.Time) > time.Minute*5 {
		return "", errors.New("token not close to expiration")
	}

	// Generate new token with same claims but extended expiration
	return GenerateJWT(claims.UserID, claims.Tier, time.Hour)
}

func ExtractUserIDFromToken(tokenString string) (uuid.UUID, error) {
	claims, err := ValidateJWT(tokenString)
	if err != nil {
		return uuid.Nil, err
	}
	return claims.UserID, nil
}

func ExtractUserTierFromToken(tokenString string) (models.UserTier, error) {
	claims, err := ValidateJWT(tokenString)
	if err != nil {
		return "", err
	}
	return claims.Tier, nil
}

func IsTokenExpired(tokenString string) bool {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil {
		return true
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return true
	}

	return claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now())
}

func GetTokenRemainingTime(tokenString string) (time.Duration, error) {
	claims, err := ValidateJWT(tokenString)
	if err != nil {
		return 0, err
	}

	if claims.ExpiresAt == nil {
		return 0, errors.New("token has no expiration")
	}

	remaining := time.Until(claims.ExpiresAt.Time)
	if remaining < 0 {
		return 0, errors.New("token expired")
	}

	return remaining, nil
}

func ValidateTokenForTier(tokenString string, requiredTier models.UserTier) error {
	claims, err := ValidateJWT(tokenString)
	if err != nil {
		return err
	}

	switch requiredTier {
	case models.TierPremium:
		if claims.Tier != models.TierPremium && claims.Tier != models.TierAdmin {
			return errors.New("premium tier required")
		}
	case models.TierAdmin:
		if claims.Tier != models.TierAdmin {
			return errors.New("admin tier required")
		}
	}

	return nil
}

func CreatePasswordResetToken(userID uuid.UUID) (string, error) {
	// Create a short-lived token for password resets (15 minutes)
	claims := &Claims{
		UserID: userID,
		Tier:   "", // Not relevant for password reset
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "premium-chat-backend",
			Subject:   "password-reset",
			ID:        uuid.New().String(),
		},
	}

	secret := os.Getenv("JWT_SECRET") + "-password-reset" // Different secret for password resets
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidatePasswordResetToken(tokenString string) (uuid.UUID, error) {
	secret := os.Getenv("JWT_SECRET") + "-password-reset"

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return uuid.Nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return uuid.Nil, errors.New("invalid token")
	}

	// Check if this is a password reset token
	if claims.Subject != "password-reset" {
		return uuid.Nil, errors.New("not a password reset token")
	}

	// Check expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return uuid.Nil, errors.New("password reset token expired")
	}

	return claims.UserID, nil
}

func CreateEmailVerificationToken(userID uuid.UUID, email string) (string, error) {
	// Create a token for email verification (24 hours)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "premium-chat-backend",
			Subject:   "email-verification",
			Audience:  []string{email},
			ID:        uuid.New().String(),
		},
	}

	secret := os.Getenv("JWT_SECRET") + "-email-verification"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateEmailVerificationToken(tokenString string) (uuid.UUID, string, error) {
	secret := os.Getenv("JWT_SECRET") + "-email-verification"

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return uuid.Nil, "", err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return uuid.Nil, "", errors.New("invalid token")
	}

	// Check if this is an email verification token
	if claims.Subject != "email-verification" {
		return uuid.Nil, "", errors.New("not an email verification token")
	}

	// Check expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return uuid.Nil, "", errors.New("email verification token expired")
	}

	var email string
	if len(claims.Audience) > 0 {
		email = claims.Audience[0]
	}

	return claims.UserID, email, nil
}
