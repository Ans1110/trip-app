package chat

import (
	"time"

	"github.com/google/uuid"
)

type RoomType string

const (
	RoomGroup RoomType = "group" // 1:1 with a trip
	RoomDM    RoomType = "dm"    // 2-person direct message
)

type MessageType string

const (
	MessageText   MessageType = "text"
	MessageImage  MessageType = "image"
	MessageVideo  MessageType = "video"
	MessageFile   MessageType = "file"
	MessageSystem MessageType = "system"
)

type Room struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TripID    *uuid.UUID `gorm:"type:uuid;column:trip_id"`
	Name      string     `gorm:"column:name;not null;default:''"`
	Type      RoomType   `gorm:"column:type;not null;default:'group'"`
	DMPairKey *string    `gorm:"column:dm_pair_key"`
	CreatedAt time.Time  `gorm:"column:created_at"`
}

func (Room) TableName() string { return "chat.room" }

type RoomMember struct {
	RoomID   uuid.UUID  `gorm:"type:uuid;primaryKey;column:room_id"`
	UserID   uuid.UUID  `gorm:"type:uuid;primaryKey;column:user_id"`
	JoinedAt time.Time  `gorm:"column:joined_at"`
	HiddenAt *time.Time `gorm:"column:hidden_at"`
}

func (RoomMember) TableName() string { return "chat.room_members" }

type Message struct {
	ID             uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RoomID         uuid.UUID   `gorm:"type:uuid;column:room_id;not null;index"`
	SenderID       uuid.UUID   `gorm:"type:uuid;column:sender_id;not null"`
	Content        string      `gorm:"column:content;not null"`
	Type           MessageType `gorm:"column:type;not null;default:'text'"`
	ReplyTo        *uuid.UUID  `gorm:"type:uuid;column:reply_to"`
	MediaURL       *string     `gorm:"column:media_url"`
	MediaID        *uuid.UUID  `gorm:"type:uuid;column:media_id"`
	MediaMime      *string     `gorm:"column:media_mime"`
	MediaBytes     *int64      `gorm:"column:media_bytes"`
	MediaFilename  *string     `gorm:"column:media_filename"`
	ClientMsgID    *string     `gorm:"column:client_msg_id"`
	IsEdited       bool        `gorm:"column:is_edited;not null;default:false"`
	EditedAt       *time.Time  `gorm:"column:edited_at"`
	IsDeleted      bool        `gorm:"column:is_deleted;not null;default:false"`
	DeletedAt      *time.Time  `gorm:"column:deleted_at"`
	IsPinned       bool        `gorm:"column:is_pinned;not null;default:false"`
	CreatedAt      time.Time   `gorm:"column:created_at"`
}

func (Message) TableName() string { return "chat.message" }

type ReadReceipt struct {
	RoomID     uuid.UUID `gorm:"type:uuid;primaryKey;column:room_id"`
	UserID     uuid.UUID `gorm:"type:uuid;primaryKey;column:user_id"`
	LastReadID uuid.UUID `gorm:"type:uuid;column:last_read_id;not null"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (ReadReceipt) TableName() string { return "chat.read_receipts" }
