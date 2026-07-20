package vote

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ---- HTTP request payloads ----

type CreatePollPayload struct {
	Type             string        `json:"type"`
	Title            string        `json:"title" binding:"required"`
	Description      string        `json:"description"`
	IsAnonymous      bool          `json:"is_anonymous"`
	MaxChoices       int           `json:"max_choices"`
	AllowOptionAdd   bool          `json:"allow_option_add"`
	ResultVisibility string        `json:"result_visibility"`
	DeadlineAt       *time.Time    `json:"deadline_at,omitempty"`
	Options          []OptionInput `json:"options"`
}

type OptionInput struct {
	Text string          `json:"text" binding:"required"`
	Meta json.RawMessage `json:"meta,omitempty"`
}

type AddOptionPayload struct {
	Text string          `json:"text" binding:"required"`
	Meta json.RawMessage `json:"meta,omitempty"`
}

type CastVotePayload struct {
	OptionIDs []uuid.UUID `json:"option_ids" binding:"required"`
}

// ---- HTTP responses ----

type PollDTO struct {
	ID               uuid.UUID        `json:"id"`
	TripID           uuid.UUID        `json:"trip_id"`
	CreatedBy        uuid.UUID        `json:"created_by"`
	Type             PollType         `json:"type"`
	Title            string           `json:"title"`
	Description      string           `json:"description,omitempty"`
	IsAnonymous      bool             `json:"is_anonymous"`
	MaxChoices       int              `json:"max_choices"`
	AllowOptionAdd   bool             `json:"allow_option_add"`
	ResultVisibility ResultVisibility `json:"result_visibility"`
	DeadlineAt       *time.Time       `json:"deadline_at,omitempty"`
	Status           PollStatus       `json:"status"`
	ClosedAt         *time.Time       `json:"closed_at,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	Options          []OptionDTO      `json:"options"`
	// MyChoices lists option ids the caller has already picked. Always
	// populated for the caller regardless of anonymity.
	MyChoices []uuid.UUID `json:"my_choices"`
	// ResultsVisible is true when the current caller is allowed to see
	// aggregate counts under the poll's result_visibility rule.
	ResultsVisible bool `json:"results_visible"`
	TotalVoters    int  `json:"total_voters"`
}

type OptionDTO struct {
	ID        uuid.UUID       `json:"id"`
	Text      string          `json:"text"`
	Meta      json.RawMessage `json:"meta,omitempty"`
	AddedBy   uuid.UUID       `json:"added_by"`
	SortOrder int             `json:"sort_order"`
	CreatedAt time.Time       `json:"created_at"`
	// Count is only populated when ResultsVisible.
	Count int `json:"count"`
	// Voters lists user ids that picked this option. Empty when the poll
	// is anonymous or results aren't visible yet.
	Voters []uuid.UUID `json:"voters,omitempty"`
}

// ---- Realtime broadcast payloads (wrapped in trip realtime channel) ----

// BroadcastKind discriminates the vote-specific WS frames pushed onto the
// trip realtime channel.
type BroadcastKind string

const (
	BroadcastPollCreated BroadcastKind = "VOTE_POLL_CREATED"
	BroadcastPollUpdated BroadcastKind = "VOTE_POLL_UPDATED"
	BroadcastPollClosed  BroadcastKind = "VOTE_POLL_CLOSED"
	BroadcastPollDeleted BroadcastKind = "VOTE_POLL_DELETED"
	BroadcastOptionAdded BroadcastKind = "VOTE_OPTION_ADDED"
	BroadcastVoteCast    BroadcastKind = "VOTE_CAST"
)
