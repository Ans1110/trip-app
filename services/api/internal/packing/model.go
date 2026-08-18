package packing

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrItemNotFound   = errors.New("packing item not found")
	ErrForbidden      = errors.New("forbidden")
	ErrInvalidPayload = errors.New("invalid payload")
)

type Item struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TripID    uuid.UUID `gorm:"type:uuid;column:trip_id;not null;index"`
	CreatedBy uuid.UUID `gorm:"type:uuid;column:created_by;not null"`
	Name      string    `gorm:"column:name;not null"`
	Quantity  int       `gorm:"column:quantity;not null;default:1"`
	Category  string    `gorm:"column:category;not null;default:''"`
	Note      string    `gorm:"column:note;not null;default:''"`
	SortOrder int       `gorm:"column:sort_order;not null;default:0"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (Item) TableName() string { return "packing.item" }

type ItemPack struct {
	ItemID   uuid.UUID `gorm:"type:uuid;column:item_id;primaryKey"`
	UserID   uuid.UUID `gorm:"type:uuid;column:user_id;primaryKey"`
	PackedAt time.Time `gorm:"column:packed_at"`
}

func (ItemPack) TableName() string { return "packing.item_pack" }
