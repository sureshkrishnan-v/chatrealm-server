package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MessageType string

const (
	MessageTypeText  MessageType = "text"
	MessageTypeImage MessageType = "image"
	MessageTypeFile  MessageType = "file"
	MessageTypeAudio MessageType = "audio"
	MessageTypeVideo MessageType = "video"
)

type Message struct {
	ID         uuid.UUID    `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ChatRoomID uuid.UUID    `json:"chat_room_id" gorm:"type:uuid;not null"`
	UserID     uuid.UUID    `json:"user_id" gorm:"type:uuid;not null"`
	Content    string       `json:"content" gorm:"type:text"`
	Type       MessageType  `json:"type" gorm:"default:'text'"`
	FileURL    *string      `json:"file_url,omitempty"`
	FileName   *string      `json:"file_name,omitempty"`
	FileSize   *int64       `json:"file_size,omitempty"`
	ParentID   *uuid.UUID   `json:"parent_id,omitempty" gorm:"type:uuid"` // For thread replies
	Mentions   []uuid.UUID  `json:"mentions" gorm:"type:uuid[]"`
	IsEdited   bool         `json:"is_edited" gorm:"default:false"`
	EditedAt   *time.Time   `json:"edited_at,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	User       User               `json:"user"`
	ChatRoom   ChatRoom           `json:"chat_room"`
	Parent     *Message           `json:"parent,omitempty"`
	Replies    []Message          `json:"replies,omitempty" gorm:"foreignKey:ParentID"`
	Reactions  []MessageReaction  `json:"reactions,omitempty"`
}

type MessageReaction struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MessageID uuid.UUID `json:"message_id" gorm:"type:uuid;not null"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null"`
	Emoji     string    `json:"emoji" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`

	// Relationships
	Message Message `json:"message"`
	User    User    `json:"user"`
}

func (m *Message) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

func (mr *MessageReaction) BeforeCreate(tx *gorm.DB) error {
	if mr.ID == uuid.Nil {
		mr.ID = uuid.New()
	}
	return nil
}

// MessageRetentionPolicy defines how long messages are kept based on user tier
func GetMessageRetentionDays(userTier UserTier) int {
	switch userTier {
	case TierFree:
		return 7 // 1 week
	case TierPremium:
		return 365 // 1 year
	case TierAdmin:
		return 0 // Unlimited
	default:
		return 1
	}
}
