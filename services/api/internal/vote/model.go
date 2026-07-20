package vote

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type PollType string

const (
	PollLocation PollType = "location"
	PollTime     PollType = "time"
	PollCustom   PollType = "custom"
)

type PollStatus string

const (
	StatusOpen   PollStatus = "open"
	StatusClosed PollStatus = "closed"
)

type ResultVisibility string

const (
	VisibilityAlways        ResultVisibility = "always"
	VisibilityAfterVote     ResultVisibility = "after_vote"
	VisibilityAfterDeadline ResultVisibility = "after_deadline"
)

var (
	ErrPollNotFound   = errors.New("poll not found")
	ErrOptionNotFound = errors.New("option not found")
	ErrForbidden      = errors.New("forbidden")
	ErrPollClosed     = errors.New("poll is closed")
	ErrInvalidPayload = errors.New("invalid payload")
	ErrTooManyChoices = errors.New("selection exceeds max_choices")
	ErrDuplicateOption = errors.New("an option with the same text already exists on this poll")
)

type Poll struct {
	ID               uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TripID           uuid.UUID        `gorm:"type:uuid;column:trip_id;not null;index"`
	CreatedBy        uuid.UUID        `gorm:"type:uuid;column:created_by;not null"`
	Type             PollType         `gorm:"column:type;not null;default:'custom'"`
	Title            string           `gorm:"column:title;not null"`
	Description      string           `gorm:"column:description;not null;default:''"`
	IsAnonymous      bool             `gorm:"column:is_anonymous;not null;default:false"`
	MaxChoices       int              `gorm:"column:max_choices;not null;default:1"`
	AllowOptionAdd   bool             `gorm:"column:allow_option_add;not null;default:false"`
	ResultVisibility ResultVisibility `gorm:"column:result_visibility;not null;default:'always'"`
	DeadlineAt       *time.Time       `gorm:"column:deadline_at"`
	Status           PollStatus       `gorm:"column:status;not null;default:'open'"`
	ClosedAt         *time.Time       `gorm:"column:closed_at"`
	CreatedAt        time.Time        `gorm:"column:created_at"`
	UpdatedAt        time.Time        `gorm:"column:updated_at"`
}

func (Poll) TableName() string { return "vote.poll" }

type Option struct {
	ID        uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PollID    uuid.UUID       `gorm:"type:uuid;column:poll_id;not null;index"`
	Text      string          `gorm:"column:text;not null"`
	Meta      json.RawMessage `gorm:"column:meta;type:jsonb;not null;default:'{}'"`
	AddedBy   uuid.UUID       `gorm:"type:uuid;column:added_by;not null"`
	SortOrder int             `gorm:"column:sort_order;not null;default:0"`
	CreatedAt time.Time       `gorm:"column:created_at"`
}

func (Option) TableName() string { return "vote.option" }

type Ballot struct {
	PollID    uuid.UUID `gorm:"type:uuid;column:poll_id;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;column:user_id;primaryKey"`
	OptionID  uuid.UUID `gorm:"type:uuid;column:option_id;primaryKey"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (Ballot) TableName() string { return "vote.ballot" }
