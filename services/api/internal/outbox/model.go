package outbox

import (
	"time"

	"github.com/google/uuid"
)

// Outbox is a shared transactional outbox row. Producers insert one row per
// domain event inside the same tx as the mutation; the realtime dispatcher
// polls and forwards to the row's target Redis stream.
type Outbox struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AggregateType string     `gorm:"column:aggregate_type;not null"`
	AggregateID   uuid.UUID  `gorm:"type:uuid;column:aggregate_id"`
	OpType        string     `gorm:"column:op_type;not null"`
	Stream        string     `gorm:"column:stream;not null"`
	ActorID       *uuid.UUID `gorm:"type:uuid;column:actor_id"`
	TraceID       string     `gorm:"column:trace_id;not null;default:''"`
	Payload       []byte     `gorm:"column:payload;type:jsonb;not null"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	DispatchedAt  *time.Time `gorm:"column:dispatched_at"`
	Attempts      int        `gorm:"column:attempts;not null;default:0"`
	LastError     string     `gorm:"column:last_error;not null;default:''"`
}

func (Outbox) TableName() string { return "platform.outbox" }
