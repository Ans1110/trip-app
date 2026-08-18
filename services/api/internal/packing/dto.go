package packing

import (
	"time"

	"github.com/google/uuid"
)

// ---- HTTP request payloads ----

type CreateItemPayload struct {
	Name      string `json:"name" binding:"required,min=1,max=200"`
	Quantity  int    `json:"quantity" binding:"omitempty,min=1,max=999"`
	Category  string `json:"category" binding:"max=32"`
	Note      string `json:"note" binding:"max=2000"`
	SortOrder int    `json:"sort_order"`
}

type UpdateItemPayload struct {
	Name      *string `json:"name" binding:"omitempty,min=1,max=200"`
	Quantity  *int    `json:"quantity" binding:"omitempty,min=1,max=999"`
	Category  *string `json:"category" binding:"omitempty,max=32"`
	Note      *string `json:"note" binding:"omitempty,max=2000"`
	SortOrder *int    `json:"sort_order"`
}

// ---- HTTP responses ----

type ItemDTO struct {
	ID          uuid.UUID `json:"id"`
	TripID      uuid.UUID `json:"trip_id"`
	CreatedBy   uuid.UUID `json:"created_by"`
	Name        string    `json:"name"`
	Quantity    int       `json:"quantity"`
	Category    string    `json:"category"`
	Note        string    `json:"note,omitempty"`
	PackedByMe  bool      `json:"packed_by_me"`
	PackedCount int       `json:"packed_count"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ---- Realtime broadcast ----

type BroadcastKind string

const (
	BroadcastItemCreated BroadcastKind = "PACKING_ITEM_CREATED"
	BroadcastItemUpdated BroadcastKind = "PACKING_ITEM_UPDATED"
	BroadcastItemDeleted BroadcastKind = "PACKING_ITEM_DELETED"
)
