package friend

import (
	"time"

	"github.com/google/uuid"
)

type RequestStatus string

const (
	RequestPending   RequestStatus = "pending"
	RequestAccepted  RequestStatus = "accepted"
	RequestDeclined  RequestStatus = "declined"
	RequestCancelled RequestStatus = "cancelled"
)

type Friendship struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey;column:user_id"`
	FriendID  uuid.UUID `gorm:"type:uuid;primaryKey;column:friend_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (Friendship) TableName() string {
	return "friend.friends"
}

type Request struct {
	ID         uuid.UUID     `gorm:"type:uuid;primaryKey"`
	SenderID   uuid.UUID     `gorm:"type:uuid;column:sender_id;not null"`
	ReceiverID uuid.UUID     `gorm:"type:uuid;column:receiver_id;not null"`
	Status     RequestStatus `gorm:"column:status;not null;default:'pending'"`
	Message    string        `gorm:"column:message;not null;default:''"`
	CreatedAt  time.Time     `gorm:"column:created_at"`
	UpdatedAt  time.Time     `gorm:"column:updated_at"`
}

func (Request) TableName() string {
	return "friend.invitations"
}

type Block struct {
	BlockerID uuid.UUID `gorm:"type:uuid;primaryKey;column:blocker_id"`
	BlockedID uuid.UUID `gorm:"type:uuid;primaryKey;column:blocked_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (Block) TableName() string {
	return "friend.blocks"
}

type InviteToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID  `gorm:"type:uuid;column:user_id;not null"`
	TokenHash string     `gorm:"column:token_hash;not null;uniqueIndex"`
	MaxUses   int        `gorm:"column:max_uses;not null;default:1"`
	Uses      int        `gorm:"column:uses;not null;default:0"`
	ExpiresAt time.Time  `gorm:"column:expires_at;not null"`
	RevokedAt *time.Time `gorm:"column:revoked_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
}

func (InviteToken) TableName() string {
	return "friend.invitation_tokens"
}
