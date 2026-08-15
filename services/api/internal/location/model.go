package location

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SavedPlace struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID      `gorm:"type:uuid;column:user_id;not null;index"`
	TripID    *uuid.UUID     `gorm:"type:uuid;column:trip_id"`
	Name      string         `gorm:"column:name;not null"`
	Address   string         `gorm:"column:address;not null;default:''"`
	Lat       float64        `gorm:"column:lat;not null"`
	Lng       float64        `gorm:"column:lng;not null"`
	PlaceID   string         `gorm:"column:place_id;not null;default:''"`
	Category  string         `gorm:"column:category;not null;default:''"`
	Notes     string         `gorm:"column:notes;not null;default:''"`
	Color     string         `gorm:"column:color;not null;default:''"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (SavedPlace) TableName() string { return "location.saved_places" }
