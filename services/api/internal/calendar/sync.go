package calendar

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/Ans1110/trip-app/pkg/stream"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TripSyncPort is the port trip.Service depends on to keep the shared trip
// event in step with the trip row itself. Every method receives the trip's
// *gorm.DB tx so calendar writes commit atomically with the trip write.
//
// Owned by the calendar package (not trip) so calendar remains the source of
// truth for its own domain shape.
type TripSyncPort interface {
	OnTripCreated(ctx context.Context, tx *gorm.DB, snap TripSnapshot) error
	OnTripUpdated(ctx context.Context, tx *gorm.DB, snap TripSnapshot) error
	OnTripDeleted(ctx context.Context, tx *gorm.DB, tripID, actorID uuid.UUID) error
	OnRoomMemberAdded(ctx context.Context, tx *gorm.DB, tripID, userID uuid.UUID) error
	OnRoomMemberRemoved(ctx context.Context, tx *gorm.DB, tripID, userID uuid.UUID) error
}

type TripSnapshot struct {
	TripID      uuid.UUID
	OwnerID     uuid.UUID
	Title       string
	Description string
	StartDate   time.Time // date-only (00:00:00 in some tz)
	EndDate     time.Time // date-only
	TimeZone    string    // IANA name; "" or "UTC" both mean UTC
	MemberIDs   []uuid.UUID
	ActorID     uuid.UUID // the user whose action triggered the sync
}

type TripSync struct {
	logger *zap.Logger
}

func NewTripSync(logger *zap.Logger) *TripSync {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &TripSync{logger: logger.With(zap.String("layer", "calendar.sync"))}
}

func (s *TripSync) OnTripCreated(ctx context.Context, tx *gorm.DB, snap TripSnapshot) error {
	repo := NewRepositoryWithTx(tx)

	start, end := tripDayBounds(snap.StartDate, snap.EndDate, snap.TimeZone)
	e := &Event{
		CreatedBy:   snap.OwnerID,
		SourceType:  SourceTrip,
		SourceID:    &snap.TripID,
		EventType:   EventTypeTrip,
		Visibility:  VisibilityRoom,
		Title:       snap.Title,
		Description: snap.Description,
		StartAt:     start,
		EndAt:       end,
		TimeZone:    snap.TimeZone,
		AllDay:      true,
		Version:     1,
	}
	if err := repo.CreateEvent(ctx, e); err != nil {
		return err
	}

	members := buildMemberRows(e.ID, snap.OwnerID, snap.MemberIDs)
	if err := repo.AddMembers(ctx, e.ID, members); err != nil {
		return err
	}
	return s.enqueue(ctx, repo, e, snap.ActorID, "CALENDAR_CREATED")
}

// OnTripUpdated propagates title/description/date-range changes to the trip
// event. Members are NOT touched here. If the event is missing
// re-create it so the invariant "each trip has one calendar event" holds.
func (s *TripSync) OnTripUpdated(ctx context.Context, tx *gorm.DB, snap TripSnapshot) error {
	repo := NewRepositoryWithTx(tx)
	e, err := repo.FindEventBySource(ctx, SourceTrip, snap.TripID)
	if err != nil {
		return err
	}
	if e == nil {
		return s.OnTripCreated(ctx, tx, snap)
	}

	start, end := tripDayBounds(snap.StartDate, snap.EndDate, snap.TimeZone)
	patch := map[string]any{
		"title":       snap.Title,
		"description": snap.Description,
		"start_at":    start,
		"end_at":      end,
		"time_zone":   snap.TimeZone,
	}
	meta := &OutboxMeta{
		OpType:  "CALENDAR_UPDATED",
		UserID:  snap.ActorID,
		TraceID: middleware.GetTraceID(ctx),
		Stream:  stream.StreamTripEvents,
		TripID:  snap.TripID,
	}
	if _, err := repo.UpdateEventWithVersion(ctx, e.ID, e.Version, patch, meta); err != nil {
		// already re-synced; treat as no-op.
		if err == ErrStaleVersion {
			s.logger.Debug("trip sync stale version, ignoring",
				zap.String("trip_id", snap.TripID.String()))
			return nil
		}
		return err
	}
	return nil
}

func (s *TripSync) OnTripDeleted(ctx context.Context, tx *gorm.DB, tripID, actorID uuid.UUID) error {
	repo := NewRepositoryWithTx(tx)
	e, err := repo.FindEventBySource(ctx, SourceTrip, tripID)
	if err != nil {
		return err
	}
	if e == nil {
		return nil
	}
	if err := repo.SoftDeleteEventBySource(ctx, SourceTrip, tripID); err != nil {
		return err
	}
	return s.enqueue(ctx, repo, e, actorID, "CALENDAR_DELETED")
}

func (s *TripSync) OnRoomMemberAdded(ctx context.Context, tx *gorm.DB, tripID, userID uuid.UUID) error {
	repo := NewRepositoryWithTx(tx)
	e, err := repo.FindEventBySource(ctx, SourceTrip, tripID)
	if err != nil {
		return err
	}
	if e == nil {
		// Silent skip — trip module will populate on next update.
		return nil
	}
	return repo.AddMembers(ctx, e.ID, []EventMember{{
		EventID: e.ID,
		UserID:  userID,
		Role:    MemberRoleMember,
	}})
}

func (s *TripSync) OnRoomMemberRemoved(ctx context.Context, tx *gorm.DB, tripID, userID uuid.UUID) error {
	repo := NewRepositoryWithTx(tx)
	return repo.RemoveMemberBySource(ctx, SourceTrip, tripID, userID)
}

// enqueue marshals the event and writes an outbox row referencing the trip.
func (s *TripSync) enqueue(ctx context.Context, repo IRepository, e *Event, actorID uuid.UUID, opType string) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	tripID := uuid.Nil
	if e.SourceID != nil {
		tripID = *e.SourceID
	}
	if opType == "CALENDAR_DELETED" {
		data = nil
	}
	payload, err := json.Marshal(outboxEvent{
		OpType:  opType,
		TripID:  tripID,
		UserID:  actorID,
		Version: e.Version,
		TraceID: middleware.GetTraceID(ctx),
		Data:    data,
	})
	if err != nil {
		return err
	}
	return repo.EnqueueOutbox(ctx, tripID, payload, middleware.GetTraceID(ctx), stream.StreamTripEvents)
}

// buildMemberRows builds the initial event_members set.
func buildMemberRows(eventID, ownerID uuid.UUID, memberIDs []uuid.UUID) []EventMember {
	seen := map[uuid.UUID]struct{}{}
	rows := make([]EventMember, 0, len(memberIDs)+1)
	rows = append(rows, EventMember{EventID: eventID, UserID: ownerID, Role: MemberRoleOwner})
	seen[ownerID] = struct{}{}
	for _, uid := range memberIDs {
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		rows = append(rows, EventMember{EventID: eventID, UserID: uid, Role: MemberRoleMember})
	}
	return rows
}

func tripDayBounds(startDate, endDate time.Time, tz string) (time.Time, time.Time) {
	loc := time.UTC
	if tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		}
	}
	start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, loc)
	end := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, loc).Add(24 * time.Hour)
	return start.UTC(), end.UTC()
}
