package models

import (
        "time"

        "github.com/google/uuid"
        "gorm.io/gorm"
)

type ChatRoomType string

const (
        RoomTypePublic  ChatRoomType = "public"
        RoomTypePrivate ChatRoomType = "private"
        RoomTypeVIP     ChatRoomType = "vip"
)

type ChatRoom struct {
        ID          uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
        Name        string          `json:"name" gorm:"not null"`
        Description *string         `json:"description"`
        Type        ChatRoomType    `json:"type" gorm:"default:'public'"`
        CreatedBy   uuid.UUID       `json:"created_by" gorm:"type:uuid;not null"`
        MaxMembers  int             `json:"max_members" gorm:"default:100"`
        IsPrivate   bool            `json:"is_private" gorm:"default:false"`
        Password    *string         `json:"-"` // Hashed password for private rooms
        Settings    ChatRoomSettings `json:"settings" gorm:"embedded"`
        CreatedAt   time.Time       `json:"created_at"`
        UpdatedAt   time.Time       `json:"updated_at"`
        DeletedAt   gorm.DeletedAt  `json:"-" gorm:"index"`

        // Relationships
        Creator     User              `json:"creator" gorm:"foreignKey:CreatedBy"`
        Members     []ChatRoomMember  `json:"members,omitempty" gorm:"foreignKey:ChatRoomID"`
        Messages    []Message         `json:"messages,omitempty" gorm:"foreignKey:ChatRoomID"`
}

type ChatRoomSettings struct {
        AllowFileUploads   bool `json:"allow_file_uploads" gorm:"default:false"`
        AllowImageUploads  bool `json:"allow_image_uploads" gorm:"default:true"`
        AllowAudioUploads  bool `json:"allow_audio_uploads" gorm:"default:false"`
        AllowVideoUploads  bool `json:"allow_video_uploads" gorm:"default:false"`
        MaxFileSize        int64 `json:"max_file_size" gorm:"default:10485760"` // 10MB
        AllowReactions     bool `json:"allow_reactions" gorm:"default:true"`
        AllowThreads       bool `json:"allow_threads" gorm:"default:false"`
        AllowMentions      bool `json:"allow_mentions" gorm:"default:true"`
        MessageHistoryDays int  `json:"message_history_days" gorm:"default:30"`
}

type ChatRoomMember struct {
        ID         uuid.UUID    `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
        ChatRoomID uuid.UUID    `json:"chat_room_id" gorm:"type:uuid;not null"`
        UserID     uuid.UUID    `json:"user_id" gorm:"type:uuid;not null"`
        Role       MemberRole   `json:"role" gorm:"default:'member'"`
        JoinedAt   time.Time    `json:"joined_at"`
        LastReadAt *time.Time   `json:"last_read_at"`
        IsMuted    bool         `json:"is_muted" gorm:"default:false"`
        IsBanned   bool         `json:"is_banned" gorm:"default:false"`

        // Relationships
        ChatRoom ChatRoom `json:"chat_room"`
        User     User     `json:"user"`
}

type MemberRole string

const (
        RoleMember    MemberRole = "member"
        RoleModerator MemberRole = "moderator"
        RoleAdmin     MemberRole = "admin"
)

func (cr *ChatRoom) BeforeCreate(tx *gorm.DB) error {
        if cr.ID == uuid.Nil {
                cr.ID = uuid.New()
        }
        
        // Set default settings based on room type
        switch cr.Type {
        case RoomTypePublic:
                cr.Settings.AllowFileUploads = false
                cr.Settings.AllowThreads = false
                cr.Settings.MessageHistoryDays = 7
        case RoomTypePrivate:
                cr.Settings.AllowFileUploads = true
                cr.Settings.AllowThreads = true
                cr.Settings.AllowAudioUploads = true
                cr.Settings.MessageHistoryDays = 30
        case RoomTypeVIP:
                cr.Settings.AllowFileUploads = true
                cr.Settings.AllowThreads = true
                cr.Settings.AllowAudioUploads = true
                cr.Settings.AllowVideoUploads = true
                cr.Settings.MaxFileSize = 104857600 // 100MB
                cr.Settings.MessageHistoryDays = 365
        }
        
        return nil
}

func (crm *ChatRoomMember) BeforeCreate(tx *gorm.DB) error {
        if crm.ID == uuid.Nil {
                crm.ID = uuid.New()
        }
        return nil
}

func (cr *ChatRoom) CanUserJoin(user *User) bool {
        if user.IsAdmin() {
                return true
        }
        
        return user.CanAccessRoom(cr.Type)
}

func (cr *ChatRoom) GetMemberCount() int {
        return len(cr.Members)
}

func (cr *ChatRoom) IsFull() bool {
        return cr.GetMemberCount() >= cr.MaxMembers
}
