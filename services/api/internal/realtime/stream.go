package realtime

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type outboxPayload struct {
	OpType  string          `json:"op_type"`
	TripID  uuid.UUID       `json:"trip_id,omitempty"`
	UserID  uuid.UUID       `json:"user_id"`
	Version int             `json:"version,omitempty"`
	TraceID string          `json:"trace_id,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func outboxEventValues(ev *outboxPayload) map[string]any {
	values := map[string]any{
		"op_type": ev.OpType,
		"trip_id": ev.TripID.String(),
		"user_id": ev.UserID.String(),
		"ts":      time.Now().UnixMilli(),
	}
	if ev.Version > 0 {
		values["version"] = ev.Version
	}
	if ev.TraceID != "" {
		values["trace_id"] = ev.TraceID
	}
	if len(ev.Data) > 0 {
		values["data"] = string(ev.Data)
	}
	return values
}
