package calendar

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/internal/audit"
	"github.com/Ans1110/trip-app/pkg/event"
	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/Ans1110/trip-app/pkg/stream"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrEventNotFound      = errors.New("calendar event not found")
	ErrForbidden          = errors.New("forbidden")
	ErrTripSourceReadOnly = errors.New("trip-sourced events: title/dates/visibility are managed by the trip")
	ErrInvalidDateRange   = errors.New("end_at must be on or after start_at")
	ErrInvalidPayload     = errors.New("invalid payload")
)

type FriendChecker interface {
	AreFriends(ctx context.Context, a, b uuid.UUID) (bool, error)
}

type RoomMemberChecker interface {
	IsRoomMember(ctx context.Context, tripID, userID uuid.UUID) (bool, error)
}

type IService interface {
	CreateEvent(ctx context.Context, userID uuid.UUID, p CreateEventPayload) (*EventResponse, error)
	GetEvent(ctx context.Context, userID, eventID uuid.UUID) (*EventResponse, error)
	UpdateEvent(ctx context.Context, userID, eventID uuid.UUID, p UpdateEventPayload) (*EventResponse, error)
	DeleteEvent(ctx context.Context, userID, eventID uuid.UUID) error
	ListMyEvents(ctx context.Context, userID uuid.UUID, q ListEventsQuery) ([]EventResponse, error)
	ListRoomEvents(ctx context.Context, userID, tripID uuid.UUID, q ListEventsQuery) ([]EventResponse, error)
	ListFriendEvents(ctx context.Context, userID, friendID uuid.UUID, q ListEventsQuery) ([]EventResponse, error)
}

type ServiceConfig struct {
	Repo       IRepository
	Friends    FriendChecker
	RoomMember RoomMemberChecker
	Logger     *zap.Logger
	Audit      audit.Writer
	Bus        *event.Bus
}

type service struct {
	repo    IRepository
	friends FriendChecker
	rooms   RoomMemberChecker
	logger  *zap.Logger
	audit   audit.Writer
	bus     *event.Bus
}

func NewService(cfg ServiceConfig) IService {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &service{
		repo:    cfg.Repo,
		friends: cfg.Friends,
		rooms:   cfg.RoomMember,
		logger:  logger.With(zap.String("layer", "calendar.service")),
		audit:   cfg.Audit,
		bus:     cfg.Bus,
	}
}

// ---- Create ----

func (s *service) CreateEvent(ctx context.Context, userID uuid.UUID, p CreateEventPayload) (*EventResponse, error) {
	if p.EndAt.Before(p.StartAt) {
		return nil, ErrInvalidDateRange
	}
	vis := Visibility(p.Visibility)
	if vis != VisibilityPrivate && vis != VisibilityFriends {
		return nil, ErrInvalidPayload
	}
	if err := ensureValidTimeZone(p.TimeZone); err != nil {
		return nil, err
	}

	et := EventType(strings.TrimSpace(p.EventType))
	if et == "" {
		et = EventTypeGeneral
	}

	e := &Event{
		CreatedBy:   userID,
		SourceType:  SourceUser,
		SourceID:    nil,
		EventType:   et,
		Visibility:  vis,
		Title:       strings.TrimSpace(p.Title),
		Description: p.Description,
		Location:    p.Location,
		StartAt:     p.StartAt.UTC(),
		EndAt:       p.EndAt.UTC(),
		TimeZone:    p.TimeZone,
		AllDay:      p.AllDay,
		Color:       p.Color,
		Version:     1,
	}

	if err := s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.CreateEvent(ctx, e); err != nil {
			return err
		}
		return tx.AddMembers(ctx, e.ID, []EventMember{{
			EventID: e.ID,
			UserID:  userID,
			Role:    MemberRoleOwner,
		}})
	}); err != nil {
		return nil, err
	}

	s.recordAudit(ctx, AuditEventCreated, audit.Success, userID, AuditResourceEvent, e.ID.String(), nil)
	s.publish(ctx, event.EventCalendarEventCreated, userID, e.ID, map[string]any{
		"event_id":   e.ID.String(),
		"visibility": string(vis),
	})
	return toEventResponse(e), nil
}

// ---- Read ----

func (s *service) GetEvent(ctx context.Context, userID, eventID uuid.UUID) (*EventResponse, error) {
	e, err := s.repo.FindEventByID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, ErrEventNotFound
	}
	ok, err := s.canView(ctx, userID, e)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrForbidden
	}
	return toEventResponse(e), nil
}

func (s *service) ListMyEvents(ctx context.Context, userID uuid.UUID, q ListEventsQuery) ([]EventResponse, error) {
	rows, err := s.repo.ListEventsForUser(ctx, userID, q)
	if err != nil {
		return nil, err
	}
	return toEventResponses(rows), nil
}

func (s *service) ListRoomEvents(ctx context.Context, userID, tripID uuid.UUID, q ListEventsQuery) ([]EventResponse, error) {
	if s.rooms == nil {
		return nil, ErrForbidden
	}
	ok, err := s.rooms.IsRoomMember(ctx, tripID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrForbidden
	}
	rows, err := s.repo.ListRoomEventsForTrip(ctx, tripID, q)
	if err != nil {
		return nil, err
	}
	return toEventResponses(rows), nil
}

func (s *service) ListFriendEvents(ctx context.Context, userID, friendID uuid.UUID, q ListEventsQuery) ([]EventResponse, error) {
	if userID == friendID {
		return s.ListMyEvents(ctx, userID, q)
	}
	if s.friends == nil {
		return nil, ErrForbidden
	}
	ok, err := s.friends.AreFriends(ctx, userID, friendID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrForbidden
	}
	rows, err := s.repo.ListFriendEventsForUser(ctx, friendID, q)
	if err != nil {
		return nil, err
	}
	return toEventResponses(rows), nil
}

// ---- Update ----

func (s *service) UpdateEvent(ctx context.Context, userID, eventID uuid.UUID, p UpdateEventPayload) (*EventResponse, error) {
	e, err := s.repo.FindEventByID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, ErrEventNotFound
	}
	if err := s.requireWriter(ctx, userID, e); err != nil {
		return nil, err
	}

	patch, err := s.buildUpdatePatch(e, p)
	if err != nil {
		return nil, err
	}
	if len(patch) == 0 {
		if e.SourceType == SourceTrip {
			return nil, ErrTripSourceReadOnly
		}
		return toEventResponse(e), nil
	}

	meta := s.outboxMetaFor(ctx, e, userID, AuditEventUpdated)
	updated, err := s.repo.UpdateEventWithVersion(ctx, eventID, p.Version, patch, meta)
	if err != nil {
		if errors.Is(err, ErrStaleVersion) {
			return nil, ErrStaleVersion
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}

	s.recordAudit(ctx, AuditEventUpdated, audit.Success, userID, AuditResourceEvent, eventID.String(), nil)
	return toEventResponse(updated), nil
}

// ---- Delete ----

func (s *service) DeleteEvent(ctx context.Context, userID, eventID uuid.UUID) error {
	e, err := s.repo.FindEventByID(ctx, eventID)
	if err != nil {
		return err
	}
	if e == nil {
		return ErrEventNotFound
	}
	if e.SourceType == SourceTrip {
		// Trip-sourced events are only deleted by the trip module
		return ErrTripSourceReadOnly
	}
	if err := s.requireWriter(ctx, userID, e); err != nil {
		return err
	}
	if err := s.repo.WithTx(ctx, func(tx IRepository) error {
		return tx.SoftDeleteEvent(ctx, eventID)
	}); err != nil {
		return err
	}
	s.recordAudit(ctx, AuditEventDeleted, audit.Success, userID, AuditResourceEvent, eventID.String(), nil)
	return nil
}

// ---- Helpers ----

func (s *service) canView(ctx context.Context, viewerID uuid.UUID, e *Event) (bool, error) {
	if e.CreatedBy == viewerID {
		return true, nil
	}
	switch e.Visibility {
	case VisibilityPublic:
		return true, nil
	case VisibilityPrivate:
		return s.repo.IsEventMember(ctx, e.ID, viewerID)
	case VisibilityFriends:
		if s.friends == nil {
			return false, nil
		}
		return s.friends.AreFriends(ctx, viewerID, e.CreatedBy)
	case VisibilityRoom:
		if e.SourceType != SourceTrip || e.SourceID == nil || s.rooms == nil {
			return false, nil
		}
		return s.rooms.IsRoomMember(ctx, *e.SourceID, viewerID)
	}
	return false, nil
}

func (s *service) requireWriter(ctx context.Context, viewerID uuid.UUID, e *Event) error {
	if e.CreatedBy == viewerID {
		return nil
	}
	return ErrForbidden
}

// buildUpdatePatch translates the pointer-y UpdateEventPayload into the map
// gorm expects, applying trip-source immutability.
func (s *service) buildUpdatePatch(e *Event, p UpdateEventPayload) (map[string]any, error) {
	trip := e.SourceType == SourceTrip
	patch := map[string]any{}

	if p.Title != nil && !trip {
		patch["title"] = strings.TrimSpace(*p.Title)
	}
	if p.Description != nil && !trip {
		patch["description"] = *p.Description
	}
	if p.Location != nil && !trip {
		patch["location"] = *p.Location
	}

	if p.StartAt != nil || p.EndAt != nil {
		if trip {
			// Silently drop — trip module owns dates.
		} else {
			start := e.StartAt
			end := e.EndAt
			if p.StartAt != nil {
				start = p.StartAt.UTC()
			}
			if p.EndAt != nil {
				end = p.EndAt.UTC()
			}
			if end.Before(start) {
				return nil, ErrInvalidDateRange
			}
			patch["start_at"] = start
			patch["end_at"] = end
		}
	}

	if p.TimeZone != nil && !trip {
		if err := ensureValidTimeZone(*p.TimeZone); err != nil {
			return nil, err
		}
		patch["time_zone"] = *p.TimeZone
	}
	if p.AllDay != nil && !trip {
		patch["all_day"] = *p.AllDay
	}
	if p.Color != nil {
		// Presentation-only — allowed even for trip-sourced events.
		patch["color"] = *p.Color
	}
	if p.EventType != nil && !trip {
		patch["event_type"] = *p.EventType
	}
	if p.Visibility != nil && !trip {
		v := Visibility(*p.Visibility)
		if v == VisibilityRoom {
			return nil, ErrInvalidPayload
		}
		patch["visibility"] = v
	}
	return patch, nil
}

func (s *service) outboxMetaFor(ctx context.Context, e *Event, userID uuid.UUID, op audit.Action) *OutboxMeta {
	if e.SourceType != SourceTrip || e.SourceID == nil {
		return nil
	}
	return &OutboxMeta{
		OpType:  wireOpFor(op),
		UserID:  userID,
		TraceID: middleware.GetTraceID(ctx),
		Stream:  stream.StreamTripEvents,
		TripID:  *e.SourceID,
	}
}

// wireOpFor maps audit-action constants to the realtime op wire strings.
func wireOpFor(a audit.Action) string {
	switch a {
	case AuditEventCreated:
		return "CALENDAR_CREATED"
	case AuditEventUpdated:
		return "CALENDAR_UPDATED"
	case AuditEventDeleted:
		return "CALENDAR_DELETED"
	}
	return ""
}

func (s *service) recordAudit(
	ctx context.Context,
	action audit.Action,
	status audit.Status,
	actorID uuid.UUID,
	resourceType, resourceID string,
	detail map[string]any,
) {
	if s.audit == nil {
		return
	}
	log := &audit.Log{
		ID:           uuid.New(),
		Action:       action,
		Status:       status,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		TraceID:      middleware.GetTraceID(ctx),
		Detail:       detail,
	}
	if actorID != uuid.Nil {
		a := actorID
		log.ActorUserID = &a
	}
	if rid := middleware.RequestIDFromContext(ctx); rid != uuid.Nil {
		log.RequestID = &rid
	}
	meta := auditMetaFromCtx(ctx)
	if meta.IPAddress != "" {
		ip := meta.IPAddress
		log.IPAddress = &ip
	}
	if meta.UserAgent != "" {
		log.UserAgent = meta.UserAgent
	}
	if err := s.audit.Create(ctx, log); err != nil {
		s.logger.Warn("audit log write failed",
			zap.Error(err),
			zap.String("action", string(action)),
		)
	}
}

func (s *service) publish(ctx context.Context, evtType string, userID, targetID uuid.UUID, payload map[string]any) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(ctx, event.Event{
		Type:      evtType,
		Payload:   payload,
		UserID:    userID,
		TargetID:  targetID,
		Timestamp: time.Now(),
	})
}

func ensureValidTimeZone(tz string) error {
	if tz == "" {
		return nil
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return ErrInvalidPayload
	}
	return nil
}

// ---- Mappers ----

func toEventResponse(e *Event) *EventResponse {
	resp := &EventResponse{
		ID:          e.ID.String(),
		CreatedBy:   e.CreatedBy.String(),
		SourceType:  string(e.SourceType),
		EventType:   string(e.EventType),
		Visibility:  string(e.Visibility),
		Title:       e.Title,
		Description: e.Description,
		Location:    e.Location,
		StartAt:     e.StartAt,
		EndAt:       e.EndAt,
		TimeZone:    e.TimeZone,
		AllDay:      e.AllDay,
		Color:       e.Color,
		Version:     e.Version,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
	if e.SourceID != nil {
		s := e.SourceID.String()
		resp.SourceID = &s
	}
	return resp
}

func toEventResponses(es []Event) []EventResponse {
	out := make([]EventResponse, 0, len(es))
	for i := range es {
		out = append(out, *toEventResponse(&es[i]))
	}
	return out
}
