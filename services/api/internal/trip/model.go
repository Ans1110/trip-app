package trip

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type TodoPriority string

const (
	TodoPriorityLow    TodoPriority = "low"
	TodoPriorityNormal TodoPriority = "normal"
	TodoPriorityHigh   TodoPriority = "high"
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
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerID      uuid.UUID      `gorm:"type:uuid;column:owner_id;not null;index"`
	Title        string         `gorm:"column:title;not null"`
	Description  string         `gorm:"column:description;not null;default:''"`
	CoverImage   string         `gorm:"column:cover_image;not null;default:''"`
	StartDate    time.Time      `gorm:"column:start_date;type:date;not null"`
	EndDate      time.Time      `gorm:"column:end_date;type:date;not null"`
	TimeZone     string         `gorm:"column:time_zone;not null;default:'UTC'"`
	BaseCurrency string         `gorm:"column:base_currency;type:char(3);not null;default:'TWD'"`
	Status       TripStatus     `gorm:"column:status;not null;default:'planning'"`
	CreatedAt    time.Time      `gorm:"column:created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index"`
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
	Latitude    *float64       `gorm:"column:latitude"`
	Longitude   *float64       `gorm:"column:longitude"`
	SortOrder   int            `gorm:"column:sort_order;not null;default:0"`
	Version     int            `gorm:"column:version;not null;default:1"`
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
	Priority    TodoPriority   `gorm:"column:priority;not null;default:'normal'"`
	Tags        pq.StringArray `gorm:"column:tags;type:text[]"`
	SortOrder   int            `gorm:"column:sort_order;not null;default:0"`
	Version     int            `gorm:"column:version;not null;default:1"`
	CreatedBy   uuid.UUID      `gorm:"type:uuid;column:created_by"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (Todo) TableName() string { return "trip.todos" }

type EventMeta struct {
	OpType  string    // wire op type (e.g. "ITINERARY_UPDATE")
	UserID  uuid.UUID // who performed the op
	TraceID string    // request trace id (may be empty)
	Stream  string    // target Redis stream
}

type OutboxEvent struct {
	OpType  string          `json:"op_type"`
	TripID  uuid.UUID       `json:"trip_id"`
	UserID  uuid.UUID       `json:"user_id"`
	Version int             `json:"version,omitempty"`
	TraceID string          `json:"trace_id,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}
