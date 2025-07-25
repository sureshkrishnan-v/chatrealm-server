package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserTier string

const (
	TierFree    UserTier = "free"
	TierPremium UserTier = "premium"
	TierAdmin   UserTier = "admin"
)

type User struct {
	ID                uuid.UUID          `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Username          string             `json:"username" gorm:"unique;not null"`
	Email             string             `json:"email" gorm:"unique;not null"`
	PasswordHash      string             `json:"-" gorm:"not null"`
	Tier              UserTier           `json:"tier" gorm:"default:'free'"`
	IsActive          bool               `json:"is_active" gorm:"default:true"`
	LastSeenAt        *time.Time         `json:"last_seen_at"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
	DeletedAt         gorm.DeletedAt     `json:"-" gorm:"index"`
	
	// Relationships
	Subscription      *Subscription      `json:"subscription,omitempty"`
	Messages          []Message          `json:"-"`
	ChatRoomMembers   []ChatRoomMember   `json:"-"`
	MessageReactions  []MessageReaction  `json:"-"`
}

type UserSession struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null"`
	Token     string    `json:"token" gorm:"unique;not null"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	
	User User `json:"user"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

func (u *User) IsPremium() bool {
	return u.Tier == TierPremium || u.Tier == TierAdmin
}

func (u *User) IsAdmin() bool {
	return u.Tier == TierAdmin
}

func (u *User) CanAccessRoom(roomType ChatRoomType) bool {
	switch roomType {
	case RoomTypePublic:
		return true
	case RoomTypePrivate:
		return u.IsPremium()
	case RoomTypeVIP:
		return u.Tier == TierPremium || u.Tier == TierAdmin
	default:
		return false
	}
}

func (u *User) GetRateLimit() int {
	switch u.Tier {
	case TierFree:
		return 10 // 10 messages per minute
	case TierPremium:
		return 60 // 60 messages per minute
	case TierAdmin:
		return 0 // Unlimited
	default:
		return 5
	}
}
