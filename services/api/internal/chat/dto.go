package chat

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// -------- WebSocket wire types --------

// OpType is the discriminator for client→server frames.
type OpType string

const (
	OpSendMessage OpType = "SEND_MESSAGE"
	OpMarkRead    OpType = "MARK_READ"
	OpTyping      OpType = "TYPING"
)

// server→client frame types
const (
	SrvWelcome = "WELCOME"
	SrvMessage = "MESSAGE"      // broadcast on successful persistence
	SrvAck     = "ACK"          // sender-only receipt
	SrvRead    = "READ_UPDATED" // someone's last_read_id advanced
	SrvTyping  = "TYPING"       // ephemeral typing indicator (not persisted)
	SrvError   = "ERROR"
)

type ErrorCode string

const (
	ErrCodeBadOp       ErrorCode = "BAD_OP"
	ErrCodeForbidden   ErrorCode = "FORBIDDEN"
	ErrCodeNotFound    ErrorCode = "NOT_FOUND"
	ErrCodeInternal    ErrorCode = "INTERNAL"
	ErrCodeUnsupported ErrorCode = "UNSUPPORTED_OP"
	ErrCodeOverloaded  ErrorCode = "OVERLOADED" // writer queue full
)

// ClientMsg is the raw envelope pulled off the WS. RoomID is validated against
// the connection's bound room_id — a mismatch is rejected as FORBIDDEN.
type ClientMsg struct {
	Type   OpType          `json:"type"`
	MsgID  string          `json:"msg_id,omitempty"` // client-side correlation id echoed on ACK/ERROR
	RoomID uuid.UUID       `json:"room_id"`
	Data   json.RawMessage `json:"data,omitempty"`
}

type ServerMsg struct {
	Type    string          `json:"type"`
	Ref     string          `json:"ref,omitempty"` // echoes MsgID on ACK/ERROR
	RoomID  uuid.UUID       `json:"room_id,omitempty"`
	Code    ErrorCode       `json:"code,omitempty"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// ---- op data shapes ----

// SendMessageData: what the client can *choose*. sender_id/created_at/id are
// server-assigned and appear on the broadcast, not here.
//
// MediaID is the ONLY media-related field the client supplies.
type SendMessageData struct {
	Content     string      `json:"content"`
	Type        MessageType `json:"type,omitempty"`
	ReplyTo     *uuid.UUID  `json:"reply_to,omitempty"`
	ClientMsgID string      `json:"client_msg_id,omitempty"`
	MediaID     *uuid.UUID  `json:"media_id,omitempty"`
}

type MarkReadData struct {
	LastReadID uuid.UUID `json:"last_read_id"`
}

type TypingData struct {
	IsTyping bool `json:"is_typing"`
}

// -------- Broadcast/HTTP DTOs (server → client) --------

// MessageDTO is the shape emitted on both the "list history" HTTP response
// AND the WS MESSAGE broadcast
type MessageDTO struct {
	ID       uuid.UUID   `json:"id"`
	RoomID   uuid.UUID   `json:"room_id"`
	SenderID uuid.UUID   `json:"sender_id"`
	Content  string      `json:"content"`
	Type     MessageType `json:"type"`
	ReplyTo  *uuid.UUID  `json:"reply_to,omitempty"`
	// MediaID is the FK into media.assets.
	MediaID     *uuid.UUID `json:"media_id,omitempty"`
	MediaMime   *string    `json:"media_mime,omitempty"`
	MediaBytes  *int64     `json:"media_bytes,omitempty"`
	ClientMsgID *string    `json:"client_msg_id,omitempty"`
	IsEdited    bool       `json:"is_edited"`
	IsDeleted   bool       `json:"is_deleted"`
	IsPinned    bool       `json:"is_pinned"`
	CreatedAt   time.Time  `json:"created_at"`
}

func MessageToDTO(m *Message) MessageDTO {
	return MessageDTO{
		ID:          m.ID,
		RoomID:      m.RoomID,
		SenderID:    m.SenderID,
		Content:     m.Content,
		Type:        m.Type,
		ReplyTo:     m.ReplyTo,
		MediaID:     m.MediaID,
		MediaMime:   m.MediaMime,
		MediaBytes:  m.MediaBytes,
		ClientMsgID: m.ClientMsgID,
		IsEdited:    m.IsEdited,
		IsDeleted:   m.IsDeleted,
		IsPinned:    m.IsPinned,
		CreatedAt:   m.CreatedAt,
	}
}

type RoomDTO struct {
	ID          uuid.UUID   `json:"id"`
	TripID      *uuid.UUID  `json:"trip_id,omitempty"`
	Name        string      `json:"name"`
	Type        RoomType    `json:"type"`
	LastMessage *MessageDTO `json:"last_message,omitempty"`
	UnreadCount int         `json:"unread_count"`
	CreatedAt   time.Time   `json:"created_at"`
}

type ReadReceiptDTO struct {
	RoomID     uuid.UUID `json:"room_id"`
	UserID     uuid.UUID `json:"user_id"`
	LastReadID uuid.UUID `json:"last_read_id"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// -------- HTTP request DTOs --------

type EnsureDMRequest struct {
	PeerID string `json:"peer_id" binding:"required"`
}

type TicketResponse struct {
	Ticket    string `json:"ticket"`
	ExpiresIn int    `json:"expires_in"`
}
