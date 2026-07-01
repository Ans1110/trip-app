package realtime

import (
	"encoding/json"

	"github.com/google/uuid"
)

// OpType is the on-wire op discriminator. CURSOR_MOVE is broadcast-only
// (never persisted); the rest go through version-check + LWW persistence
// before being broadcast.
type OpType string

const (
	OpCursorMove      OpType = "CURSOR_MOVE"
	OpItineraryUpdate OpType = "ITINERARY_UPDATE"
	OpTodoUpdate      OpType = "TODO_UPDATE"
	OpCalendarUpdate  OpType = "CALENDAR_UPDATE"
	OpVoteCast        OpType = "VOTE_CAST"

	// Server -> client only.
	srvAck     = "ACK"
	srvError   = "ERROR"
	srvWelcome = "WELCOME"
)

// ErrorCode is sent to the client when an op is rejected.
type ErrorCode string

const (
	ErrCodeBadOp        ErrorCode = "BAD_OP"
	ErrCodeStaleVersion ErrorCode = "STALE_VERSION"
	ErrCodeNotFound     ErrorCode = "NOT_FOUND"
	ErrCodeForbidden    ErrorCode = "FORBIDDEN"
	ErrCodeInternal     ErrorCode = "INTERNAL"
	ErrCodeUnsupported  ErrorCode = "UNSUPPORTED_OP"
)

type ClientMsg struct {
	Type    OpType          `json:"type"`
	MsgID   string          `json:"msg_id,omitempty"`
	TripID  uuid.UUID       `json:"trip_id"`
	Version int             `json:"version,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type ServerMsg struct {
	Type    string          `json:"type"`
	Seq     int64           `json:"seq,omitempty"`
	From    uuid.UUID       `json:"from,omitempty"`
	TripID  uuid.UUID       `json:"trip_id,omitempty"`
	Version int             `json:"version,omitempty"`
	Ref     string          `json:"ref,omitempty"`
	Code    ErrorCode       `json:"code,omitempty"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// ---- Op-specific data shapes (only what the server needs to interpret) ----

type CursorMoveData struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Selection string  `json:"selection,omitempty"`
}

type ItineraryUpdateData struct {
	ItemID      uuid.UUID `json:"item_id"`
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty"`
	Location    *string   `json:"location,omitempty"`
	Day         *int      `json:"day,omitempty"`
	Latitude    *float64  `json:"latitude,omitempty"`
	Longitude   *float64  `json:"longitude,omitempty"`
}

type TodoUpdateData struct {
	TodoID      uuid.UUID `json:"todo_id"`
	Title       *string   `json:"title,omitempty"`
	IsCompleted *bool     `json:"is_completed,omitempty"`
	Priority    *string   `json:"priority,omitempty"`
}
