package trip

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TripStatus string

const (
	TripPlanning  TripStatus = "planning"
	TripOngoing   TripStatus = "ongoing"
	TripCompleted TripStatus = "completed"
)

type RoomMemberRole string

const (
	RoleAdmin  RoomMemberRole = "admin"
	RoleMember RoomMemberRole = "member"
)

type Trip struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerID     uuid.UUID      `gorm:"type:uuid;column:owner_id;not null;index"`
	Title       string         `gorm:"column:title;not null"`
	Description string         `gorm:"column:description;not null;default:''"`
	CoverImage  string         `gorm:"column:cover_image;not null;default:''"`
	StartDate   time.Time      `gorm:"column:start_date;type:date;not null"`
	EndDate     time.Time      `gorm:"column:end_date;type:date;not null"`
	Status      TripStatus     `gorm:"column:status;not null;default:'planning'"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (Trip) TableName() string { return "trip.trip" }

type Room struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TripID    uuid.UUID `gorm:"type:uuid;column:trip_id;not null;uniqueIndex"`
	RoomCode  string    `gorm:"column:room_code;not null;uniqueIndex"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (Room) TableName() string { return "trip.rooms" }

type RoomMember struct {
	RoomID   uuid.UUID      `gorm:"type:uuid;primaryKey;column:room_id"`
	UserID   uuid.UUID      `gorm:"type:uuid;primaryKey;column:user_id"`
	Role     RoomMemberRole `gorm:"column:role;not null;default:'member'"`
	JoinedAt time.Time      `gorm:"column:joined_at"`
}

func (RoomMember) TableName() string { return "trip.room_members" }

type Itinerary struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TripID      uuid.UUID      `gorm:"type:uuid;column:trip_id;not null;index"`
	Day         int            `gorm:"column:day;not null"`
	Title       string         `gorm:"column:title;not null;default:''"`
	Description string         `gorm:"column:description;not null;default:''"`
	StartTime   *time.Time     `gorm:"column:start_time"`
	EndTime     *time.Time     `gorm:"column:end_time"`
	Location    string         `gorm:"column:location;not null;default:''"`
	SortOrder   int            `gorm:"column:sort_order;not null;default:0"`
	CreatedBy   uuid.UUID      `gorm:"type:uuid;column:created_by"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (Itinerary) TableName() string { return "trip.itineraries" }

type Todo struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TripID      uuid.UUID      `gorm:"type:uuid;column:trip_id;not null;index"`
	AssigneeID  *uuid.UUID     `gorm:"type:uuid;column:assignee_id"`
	Title       string         `gorm:"column:title;not null"`
	IsCompleted bool           `gorm:"column:is_completed;not null;default:false"`
	DueDate     *time.Time     `gorm:"column:due_date"`
	CreatedBy   uuid.UUID      `gorm:"type:uuid;column:created_by"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (Todo) TableName() string { return "trip.todos" }
