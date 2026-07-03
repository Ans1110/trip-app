package calendar

import "time"

type CreateEventPayload struct {
	Title       string    `json:"title" binding:"required,min=1,max=200"`
	Description string    `json:"description" binding:"max=2000"`
	Location    string    `json:"location" binding:"max=200"`
	StartAt     time.Time `json:"start_at" binding:"required"`
	EndAt       time.Time `json:"end_at" binding:"required"`
	TimeZone    string    `json:"time_zone" binding:"max=64"`
	AllDay      bool      `json:"all_day"`
	Color       string    `json:"color" binding:"max=16"`
	EventType   string    `json:"event_type" binding:"omitempty,oneof=general trip flight hotel meeting"`
	Visibility  string    `json:"visibility" binding:"required,oneof=private friends"`
}

type UpdateEventPayload struct {
	Title       *string    `json:"title" binding:"omitempty,min=1,max=200"`
	Description *string    `json:"description" binding:"omitempty,max=2000"`
	Location    *string    `json:"location" binding:"omitempty,max=200"`
	StartAt     *time.Time `json:"start_at"`
	EndAt       *time.Time `json:"end_at"`
	TimeZone    *string    `json:"time_zone" binding:"omitempty,max=64"`
	AllDay      *bool      `json:"all_day"`
	Color       *string    `json:"color" binding:"omitempty,max=16"`
	EventType   *string    `json:"event_type" binding:"omitempty,oneof=general trip flight hotel meeting"`
	Visibility  *string    `json:"visibility" binding:"omitempty,oneof=private friends public"`
	Version     int        `json:"version" binding:"required,min=1"`
}

type ListEventsQuery struct {
	From *time.Time
	To   *time.Time
}

type EventResponse struct {
	ID          string    `json:"id"`
	CreatedBy   string    `json:"created_by"`
	SourceType  string    `json:"source_type"`
	SourceID    *string   `json:"source_id,omitempty"`
	EventType   string    `json:"event_type"`
	Visibility  string    `json:"visibility"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Location    string    `json:"location,omitempty"`
	StartAt     time.Time `json:"start_at"`
	EndAt       time.Time `json:"end_at"`
	TimeZone    string    `json:"time_zone,omitempty"`
	AllDay      bool      `json:"all_day"`
	Color       string    `json:"color,omitempty"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
