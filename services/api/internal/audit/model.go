package audit

import (
	"time"

	"github.com/google/uuid"
)

// Action is a domain-defined event identifier.
// Each domain declares its own typed constants.
type Action string

// Status is the outcome of an audited event.
type Status string

const (
	Success Status = "success"
	Failure Status = "failure"
)

// Log is one row in the audit table. Persisted to auth.audit_logs (the schema
// name is historical; the table is generic).
type Log struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey"`
	ActorUserID  *uuid.UUID     `gorm:"type:uuid;column:actor_user_id;index"`
	TargetUserID *uuid.UUID     `gorm:"type:uuid;column:target_user_id;index"`
	Action       Action         `gorm:"size:64;not null;index"`
	Status       Status         `gorm:"size:32;not null"`
	ResourceType string         `gorm:"size:64;column:resource_type"`
	ResourceID   string         `gorm:"type:text;column:resource_id"`
	IPAddress    *string        `gorm:"type:inet;column:ip_address"`
	UserAgent    string         `gorm:"type:text;column:user_agent"`
	RequestID    *uuid.UUID     `gorm:"type:uuid;column:request_id"`
	TraceID      string         `gorm:"type:text;column:trace_id"`
	Detail       map[string]any `gorm:"type:jsonb;serializer:json"`
	CreatedAt    time.Time
}

func (Log) TableName() string {
	return "auth.audit_logs"
}
