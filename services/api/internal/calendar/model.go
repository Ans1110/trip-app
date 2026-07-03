package calendar

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SourceType identifies what produced an event.
type SourceType string

const (
	SourceUser SourceType = "user"
	SourceTrip SourceType = "trip"
)

// EventType is a display / filter tag. It carries no authorization meaning —
// clients use it to pick an icon or group items in the calendar UI.
type EventType string

const (
	EventTypeGeneral EventType = "general"
	EventTypeTrip    EventType = "trip"
	EventTypeFlight  EventType = "flight"
	EventTypeHotel   EventType = "hotel"
	EventTypeMeeting EventType = "meeting"
)

// Visibility is the single authorization axis. event_members controls whose
// calendar an event appears on; it does NOT grant view rights.
type Visibility string

const (
	VisibilityPrivate Visibility = "private"
	VisibilityRoom    Visibility = "room"
	VisibilityFriends Visibility = "friends"
	VisibilityPublic  Visibility = "public"
)

type MemberRole string

const (
	MemberRoleOwner  MemberRole = "owner"
	MemberRoleMember MemberRole = "member"
)

type Event struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedBy   uuid.UUID      `gorm:"type:uuid;column:created_by;not null"`
	SourceType  SourceType     `gorm:"column:source_type;not null;default:'user'"`
	SourceID    *uuid.UUID     `gorm:"type:uuid;column:source_id"`
	EventType   EventType      `gorm:"column:event_type;not null;default:'general'"`
	Visibility  Visibility     `gorm:"column:visibility;not null"`
	Title       string         `gorm:"column:title;not null"`
	Description string         `gorm:"column:description;not null;default:''"`
	Location    string         `gorm:"column:location;not null;default:''"`
	StartAt     time.Time      `gorm:"column:start_at;not null"`
	EndAt       time.Time      `gorm:"column:end_at;not null"`
	TimeZone    string         `gorm:"column:time_zone;not null;default:''"`
	AllDay      bool           `gorm:"column:all_day;not null;default:false"`
	Color       string         `gorm:"column:color;not null;default:''"`
	Version     int            `gorm:"column:version;not null;default:1"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (Event) TableName() string { return "calendar.events" }

type EventMember struct {
	EventID  uuid.UUID  `gorm:"type:uuid;primaryKey;column:event_id"`
	UserID   uuid.UUID  `gorm:"type:uuid;primaryKey;column:user_id"`
	Role     MemberRole `gorm:"column:role;not null;default:'member'"`
	JoinedAt time.Time  `gorm:"column:joined_at"`
}

func (EventMember) TableName() string { return "calendar.event_members" }

type outboxEvent struct {
	OpType  string          `json:"op_type"`
	TripID  uuid.UUID       `json:"trip_id,omitempty"`
	UserID  uuid.UUID       `json:"user_id"`
	Version int             `json:"version,omitempty"`
	TraceID string          `json:"trace_id,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type OutboxMeta struct {
	OpType  string
	UserID  uuid.UUID
	TraceID string
	Stream  string
	TripID  uuid.UUID // fanout - 0 means don't publish
}
