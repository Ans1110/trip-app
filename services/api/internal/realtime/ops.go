package realtime

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Ans1110/trip-app/internal/trip"
	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/Ans1110/trip-app/pkg/stream"
	"github.com/google/uuid"
)

func (s *Service) applyItineraryUpdate(ctx context.Context, tripID, userID uuid.UUID, expectedVersion int, data json.RawMessage) (int, json.RawMessage, *opError) {
	id, patch, oe := decodeVersionedPatch(expectedVersion, data, func(p ItineraryUpdateData) (uuid.UUID, map[string]any, *opError) {
		if p.ItemID == uuid.Nil {
			return uuid.Nil, nil, &opError{code: ErrCodeBadOp, msg: "item_id required"}
		}
		patch := map[string]any{}
		if p.Title != nil {
			patch["title"] = *p.Title
		}
		if p.Description != nil {
			patch["description"] = *p.Description
		}
		if p.Location != nil {
			patch["location"] = *p.Location
		}
		if p.Day != nil {
			if *p.Day < 1 {
				return uuid.Nil, nil, &opError{code: ErrCodeBadOp, msg: "day must be >= 1"}
			}
			patch["day"] = *p.Day
		}
		if p.Latitude != nil {
			patch["latitude"] = *p.Latitude
		}
		if p.Longitude != nil {
			patch["longitude"] = *p.Longitude
		}
		return p.ItemID, patch, nil
	})
	if oe != nil {
		return 0, nil, oe
	}

	existing, err := s.tripRepo.FindItineraryByID(ctx, id)
	if oe := precheckEntity(existing, err, tripID, func(it *trip.Itinerary) uuid.UUID { return it.TripID }, "itinerary"); oe != nil {
		return 0, nil, oe
	}

	return s.runVersionedUpdate(ctx, userID, OpItineraryUpdate, trip.ErrItineraryNotFound, "itinerary",
		func(event *trip.EventMeta) (any, int, error) {
			u, err := s.tripRepo.UpdateItineraryWithVersion(ctx, id, expectedVersion, patch, event)
			if err != nil {
				return nil, 0, err
			}
			return u, u.Version, nil
		})
}

func (s *Service) applyTodoUpdate(ctx context.Context, tripID, userID uuid.UUID, expectedVersion int, data json.RawMessage) (int, json.RawMessage, *opError) {
	id, patch, oe := decodeVersionedPatch(expectedVersion, data, func(p TodoUpdateData) (uuid.UUID, map[string]any, *opError) {
		if p.TodoID == uuid.Nil {
			return uuid.Nil, nil, &opError{code: ErrCodeBadOp, msg: "todo_id required"}
		}
		patch := map[string]any{}
		if p.Title != nil {
			patch["title"] = *p.Title
		}
		if p.IsCompleted != nil {
			patch["is_completed"] = *p.IsCompleted
		}
		if p.Priority != nil {
			switch *p.Priority {
			case "low", "normal", "high":
				patch["priority"] = *p.Priority
			default:
				return uuid.Nil, nil, &opError{code: ErrCodeBadOp, msg: "invalid priority"}
			}
		}
		return p.TodoID, patch, nil
	})
	if oe != nil {
		return 0, nil, oe
	}

	existing, err := s.tripRepo.FindTodoByID(ctx, id)
	if oe := precheckEntity(existing, err, tripID, func(t *trip.Todo) uuid.UUID { return t.TripID }, "todo"); oe != nil {
		return 0, nil, oe
	}

	return s.runVersionedUpdate(ctx, userID, OpTodoUpdate, trip.ErrTodoNotFound, "todo",
		func(event *trip.EventMeta) (any, int, error) {
			u, err := s.tripRepo.UpdateTodoWithVersion(ctx, id, expectedVersion, patch, event)
			if err != nil {
				return nil, 0, err
			}
			return u, u.Version, nil
		})
}

// decodeVersionedPatch runs the checks shared by every versioned entity update
func decodeVersionedPatch[P any](expectedVersion int, data json.RawMessage, parse func(P) (uuid.UUID, map[string]any, *opError)) (uuid.UUID, map[string]any, *opError) {
	if expectedVersion <= 0 {
		return uuid.Nil, nil, &opError{code: ErrCodeBadOp, msg: "version required"}
	}
	var p P
	if err := json.Unmarshal(data, &p); err != nil {
		return uuid.Nil, nil, &opError{code: ErrCodeBadOp, msg: "invalid data"}
	}
	id, patch, oe := parse(p)
	if oe != nil {
		return uuid.Nil, nil, oe
	}
	if len(patch) == 0 {
		return uuid.Nil, nil, &opError{code: ErrCodeBadOp, msg: "empty patch"}
	}
	return id, patch, nil
}

// precheckEntity funnels the repeated fetch/nil/ownership triangle through a
// single generic helper so callers don't spell out three near-identical error
// branches per op.
func precheckEntity[T any](entity *T, err error, tripID uuid.UUID, getTripID func(*T) uuid.UUID, label string) *opError {
	if err != nil {
		return &opError{code: ErrCodeInternal, msg: "lookup failed", err: err}
	}
	if entity == nil {
		return &opError{code: ErrCodeNotFound, msg: label + " not found"}
	}
	if getTripID(entity) != tripID {
		return &opError{code: ErrCodeForbidden, msg: label + " belongs to another trip"}
	}
	return nil
}

func (s *Service) runVersionedUpdate(
	ctx context.Context,
	userID uuid.UUID,
	opType OpType,
	notFoundErr error,
	label string,
	apply func(event *trip.EventMeta) (any, int, error),
) (int, json.RawMessage, *opError) {
	event := &trip.EventMeta{
		OpType:  string(opType),
		UserID:  userID,
		TraceID: middleware.GetTraceID(ctx),
		Stream:  stream.StreamTripEvents,
	}
	updated, version, err := apply(event)
	if err != nil {
		switch {
		case errors.Is(err, trip.ErrStaleVersion):
			return 0, nil, &opError{code: ErrCodeStaleVersion, msg: "stale version"}
		case errors.Is(err, notFoundErr):
			return 0, nil, &opError{code: ErrCodeNotFound, msg: label + " not found"}
		default:
			return 0, nil, &opError{code: ErrCodeInternal, msg: "update failed", err: err}
		}
	}
	out, err := json.Marshal(updated)
	if err != nil {
		return 0, nil, &opError{code: ErrCodeInternal, msg: "marshal failed", err: err}
	}
	return version, out, nil
}

type opError struct {
	code ErrorCode
	msg  string
	err  error
}
