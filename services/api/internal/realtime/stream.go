package realtime

import (
	"time"

	"github.com/Ans1110/trip-app/internal/trip"
)

func outboxEventValues(ev *trip.OutboxEvent) map[string]any {
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
