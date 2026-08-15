package trip

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/internal/audit"
	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/Ans1110/trip-app/internal/calendar"
	"github.com/Ans1110/trip-app/pkg/event"
	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrTripNotFound      = errors.New("trip not found")
	ErrRoomNotFound      = errors.New("room not found")
	ErrItineraryNotFound = errors.New("itinerary item not found")
	ErrTodoNotFound      = errors.New("todo not found")
	ErrUserNotFound      = errors.New("user not found")
	ErrForbidden         = errors.New("forbidden")
	ErrNotMember         = errors.New("not a room member")
	ErrOwnerCannotLeave  = errors.New("trip owner cannot leave; transfer or delete trip")
	ErrCannotRemoveOwner = errors.New("cannot remove trip owner")
	ErrInvalidDateRange  = errors.New("end_date must be on or after start_date")
	ErrInvalidPayload    = errors.New("invalid payload")
	ErrCodeRequired      = errors.New("room code required")
	ErrStatusAutoOnly    = errors.New("'completed' status is set automatically when the trip ends")
)

const dateLayout = "2006-01-02"

type IService interface {
	// Trip
	CreateTrip(ctx context.Context, ownerID uuid.UUID, p CreateTripPayload) (*TripResponse, error)
	GetTrip(ctx context.Context, userID, tripID uuid.UUID) (*TripResponse, error)
	UpdateTrip(ctx context.Context, userID, tripID uuid.UUID, p UpdateTripPayload) (*TripResponse, error)
	DeleteTrip(ctx context.Context, userID, tripID uuid.UUID) error
	ListTrips(ctx context.Context, userID uuid.UUID, q ListTripsQuery) ([]TripResponse, error)

	// Room
	GetRoom(ctx context.Context, userID, tripID uuid.UUID) (*RoomResponse, error)
	PreviewByCode(ctx context.Context, userID uuid.UUID, code string) (*RoomPreview, error)
	JoinRoom(ctx context.Context, userID uuid.UUID, p JoinRoomPayload) (*JoinRoomResponse, error)
	LeaveRoom(ctx context.Context, userID, tripID uuid.UUID) error
	RemoveMember(ctx context.Context, actorID, tripID, targetID uuid.UUID) error
	RegenerateCode(ctx context.Context, userID, tripID uuid.UUID) (*RoomResponse, error)

	// Itinerary
	CreateItinerary(ctx context.Context, userID, tripID uuid.UUID, p CreateItineraryPayload) (*ItineraryResponse, error)
	ListItinerary(ctx context.Context, userID, tripID uuid.UUID) ([]ItineraryResponse, error)
	UpdateItinerary(ctx context.Context, userID, tripID, itemID uuid.UUID, p UpdateItineraryPayload) (*ItineraryResponse, error)
	DeleteItinerary(ctx context.Context, userID, tripID, itemID uuid.UUID) error
	ReorderItinerary(ctx context.Context, userID, tripID uuid.UUID, p ReorderItineraryPayload) error

	// Todos
	CreateTodo(ctx context.Context, userID, tripID uuid.UUID, p CreateTodoPayload) (*TodoResponse, error)
	ListTodos(ctx context.Context, userID, tripID uuid.UUID) ([]TodoResponse, error)
	UpdateTodo(ctx context.Context, userID, tripID, todoID uuid.UUID, p UpdateTodoPayload) (*TodoResponse, error)
	DeleteTodo(ctx context.Context, userID, tripID, todoID uuid.UUID) error
	ReorderTodos(ctx context.Context, userID, tripID uuid.UUID, p ReorderTodosPayload) error
}

type ServiceConfig struct {
	Repo   IRepository
	Logger *zap.Logger
	Audit  audit.Writer
	Bus    *event.Bus

	Cal calendar.TripSyncPort
}

type service struct {
	repo   IRepository
	logger *zap.Logger
	audit  audit.Writer
	bus    *event.Bus
	cal    calendar.TripSyncPort
}

func NewService(cfg ServiceConfig) IService {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &service{
		repo:   cfg.Repo,
		logger: logger.With(zap.String("layer", "trip.service")),
		audit:  cfg.Audit,
		bus:    cfg.Bus,
		cal:    cfg.Cal,
	}
}

// ---- Trip ----

func (s *service) CreateTrip(ctx context.Context, ownerID uuid.UUID, p CreateTripPayload) (*TripResponse, error) {
	start, end, err := parseDateRange(p.StartDate, p.EndDate)
	if err != nil {
		return nil, err
	}
	tz := normalizeTimeZone(p.TimeZone)
	if err := validateTimeZone(tz); err != nil {
		return nil, err
	}
	owner, err := s.repo.FindUserByID(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if owner == nil || owner.Status != auth.UserStatusActive {
		return nil, ErrUserNotFound
	}

	t := &Trip{
		OwnerID:     ownerID,
		Title:       strings.TrimSpace(p.Title),
		Description: p.Description,
		CoverImage:  p.CoverImage,
		StartDate:   start,
		EndDate:     end,
		TimeZone:    tz,
		Status:      TripPlanning,
		Location:    strings.TrimSpace(p.Location),
		Latitude:    p.Latitude,
		Longitude:   p.Longitude,
	}

	var (
		room *Room
	)
	if err := s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.CreateTrip(ctx, t); err != nil {
			return err
		}
		code, err := s.uniqueRoomCode(ctx, tx)
		if err != nil {
			return err
		}
		room = &Room{TripID: t.ID, RoomCode: code}
		if err := tx.CreateRoom(ctx, room); err != nil {
			return err
		}
		if err := tx.AddMember(ctx, &RoomMember{
			RoomID:   room.ID,
			UserID:   ownerID,
			Role:     RoleAdmin,
			JoinedAt: time.Now(),
		}); err != nil {
			return err
		}
		return s.syncTripCreated(ctx, tx, t, ownerID)
	}); err != nil {
		return nil, err
	}

	s.recordAudit(ctx, AuditTripCreated, audit.Success, ownerID, nil, AuditResourceTrip, t.ID.String(), nil)
	s.publish(ctx, event.EventTripCreated, ownerID, t.ID, map[string]any{
		"trip_id": t.ID.String(),
		"room_id": room.ID.String(),
	})

	return s.buildTripResponse(ctx, *t, room, 1, string(RoleAdmin), *owner)
}

func (s *service) GetTrip(ctx context.Context, userID, tripID uuid.UUID) (*TripResponse, error) {
	t, err := s.repo.FindTripByID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTripNotFound
	}
	member, err := s.requireRoomMember(ctx, tripID, userID, t.OwnerID)
	if err != nil {
		return nil, err
	}
	room, err := s.repo.FindRoomByTripID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, ErrRoomNotFound
	}
	owner, err := s.repo.FindUserByID(ctx, t.OwnerID)
	if err != nil {
		return nil, err
	}
	if owner == nil {
		return nil, ErrUserNotFound
	}
	count, err := s.repo.CountRoomMembers(ctx, room.ID)
	if err != nil {
		return nil, err
	}
	role := ""
	if member != nil {
		role = string(member.Role)
	}
	return s.buildTripResponse(ctx, *t, room, count, role, *owner)
}

func (s *service) UpdateTrip(ctx context.Context, userID, tripID uuid.UUID, p UpdateTripPayload) (*TripResponse, error) {
	t, err := s.repo.FindTripByID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTripNotFound
	}
	if t.OwnerID != userID {
		return nil, ErrForbidden
	}

	patch := map[string]any{}
	if p.Title != nil {
		patch["title"] = strings.TrimSpace(*p.Title)
	}
	if p.Description != nil {
		patch["description"] = *p.Description
	}
	if p.CoverImage != nil {
		patch["cover_image"] = *p.CoverImage
	}
	if p.Status != nil {
		if *p.Status == string(TripCompleted) {
			return nil, ErrStatusAutoOnly
		}
		patch["status"] = *p.Status
	}
	// Detect private→public transition so we can enqueue TRIP_PUBLISHED in
	// the same tx as the visibility flip. Ignore no-ops and reverse flips.
	var publishTransition bool
	if p.Visibility != nil {
		newVis := TripVisibility(*p.Visibility)
		if newVis != effectiveVisibility(*t) {
			patch["visibility"] = string(newVis)
			publishTransition = newVis == TripVisibilityPublic
		}
	}
	if p.StartDate != nil || p.EndDate != nil {
		startStr, endStr := t.StartDate.Format(dateLayout), t.EndDate.Format(dateLayout)
		if p.StartDate != nil {
			startStr = *p.StartDate
		}
		if p.EndDate != nil {
			endStr = *p.EndDate
		}
		start, end, err := parseDateRange(startStr, endStr)
		if err != nil {
			return nil, err
		}
		patch["start_date"] = start
		patch["end_date"] = end
	}
	if p.TimeZone != nil {
		tz := normalizeTimeZone(*p.TimeZone)
		if err := validateTimeZone(tz); err != nil {
			return nil, err
		}
		patch["time_zone"] = tz
	}
	if p.Location != nil {
		patch["location"] = strings.TrimSpace(*p.Location)
	}
	if p.Latitude != nil {
		patch["latitude"] = *p.Latitude
	}
	if p.Longitude != nil {
		patch["longitude"] = *p.Longitude
	}

	if len(patch) > 0 {
		if err := s.repo.WithTx(ctx, func(tx IRepository) error {
			if err := tx.UpdateTrip(ctx, tripID, patch); err != nil {
				return err
			}
			if publishTransition {
				if err := tx.EnqueueTripPublished(ctx, tripID, userID, time.Now(), middleware.GetTraceID(ctx)); err != nil {
					return err
				}
			}
			return s.syncTripUpdated(ctx, tx, tripID, userID)
		}); err != nil {
			return nil, err
		}
	}

	s.recordAudit(ctx, AuditTripUpdated, audit.Success, userID, nil, AuditResourceTrip, tripID.String(), nil)
	s.publish(ctx, event.EventTripUpdated, userID, tripID, map[string]any{"trip_id": tripID.String()})

	return s.GetTrip(ctx, userID, tripID)
}

func (s *service) DeleteTrip(ctx context.Context, userID, tripID uuid.UUID) error {
	t, err := s.repo.FindTripByID(ctx, tripID)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrTripNotFound
	}
	if t.OwnerID != userID {
		return ErrForbidden
	}
	if err := s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.DeleteTrip(ctx, tripID); err != nil {
			return err
		}
		if s.cal != nil {
			return s.cal.OnTripDeleted(ctx, tx.Tx(), tripID, userID)
		}
		return nil
	}); err != nil {
		return err
	}
	s.recordAudit(ctx, AuditTripDeleted, audit.Success, userID, nil, AuditResourceTrip, tripID.String(), nil)
	s.publish(ctx, event.EventTripDeleted, userID, tripID, map[string]any{"trip_id": tripID.String()})
	return nil
}

func (s *service) ListTrips(ctx context.Context, userID uuid.UUID, q ListTripsQuery) ([]TripResponse, error) {
	trips, err := s.repo.ListTrips(ctx, userID, q)
	if err != nil {
		return nil, err
	}
	if len(trips) == 0 {
		return []TripResponse{}, nil
	}

	tripIDs := make([]uuid.UUID, 0, len(trips))
	ownerIDs := make([]uuid.UUID, 0, len(trips))
	seen := map[uuid.UUID]struct{}{}
	for _, t := range trips {
		tripIDs = append(tripIDs, t.ID)
		if _, ok := seen[t.OwnerID]; !ok {
			seen[t.OwnerID] = struct{}{}
			ownerIDs = append(ownerIDs, t.OwnerID)
		}
	}
	owners, err := s.repo.FindUsersByIDs(ctx, ownerIDs)
	if err != nil {
		return nil, err
	}
	counts, err := s.repo.CountMembers(ctx, tripIDs)
	if err != nil {
		return nil, err
	}

	out := make([]TripResponse, 0, len(trips))
	for _, t := range trips {
		owner := owners[t.OwnerID]
		role := ""
		if t.OwnerID == userID {
			role = string(RoleAdmin)
		} else {
			m, err := s.repo.FindMemberByTrip(ctx, t.ID, userID)
			if err != nil {
				return nil, err
			}
			if m != nil {
				role = string(m.Role)
			}
		}
		room, err := s.repo.FindRoomByTripID(ctx, t.ID)
		if err != nil {
			return nil, err
		}
		resp, err := s.buildTripResponse(ctx, t, room, counts[t.ID], role, owner)
		if err != nil {
			return nil, err
		}
		out = append(out, *resp)
	}
	return out, nil
}

// ---- Room ----

func (s *service) GetRoom(ctx context.Context, userID, tripID uuid.UUID) (*RoomResponse, error) {
	t, err := s.repo.FindTripByID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTripNotFound
	}
	if _, err := s.requireRoomMember(ctx, tripID, userID, t.OwnerID); err != nil {
		return nil, err
	}
	room, err := s.repo.FindRoomByTripID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, ErrRoomNotFound
	}
	members, err := s.repo.ListMembers(ctx, room.ID)
	if err != nil {
		return nil, err
	}
	uids := make([]uuid.UUID, 0, len(members))
	for _, m := range members {
		uids = append(uids, m.UserID)
	}
	users, err := s.repo.FindUsersByIDs(ctx, uids)
	if err != nil {
		return nil, err
	}
	memberDTOs := make([]RoomMemberResponse, 0, len(members))
	for _, m := range members {
		u, ok := users[m.UserID]
		if !ok {
			continue
		}
		memberDTOs = append(memberDTOs, RoomMemberResponse{
			User:     toUserSummary(u),
			Role:     string(m.Role),
			JoinedAt: m.JoinedAt,
		})
	}
	return &RoomResponse{
		ID:        room.ID.String(),
		TripID:    tripID.String(),
		RoomCode:  room.RoomCode,
		Members:   memberDTOs,
		CreatedAt: room.CreatedAt,
	}, nil
}

func (s *service) PreviewByCode(ctx context.Context, userID uuid.UUID, code string) (*RoomPreview, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil, ErrInvalidPayload
	}
	room, err := s.repo.FindRoomByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, ErrRoomNotFound
	}
	t, err := s.repo.FindTripByID(ctx, room.TripID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTripNotFound
	}
	owner, err := s.repo.FindUserByID(ctx, t.OwnerID)
	if err != nil {
		return nil, err
	}
	if owner == nil {
		return nil, ErrUserNotFound
	}
	count, err := s.repo.CountRoomMembers(ctx, room.ID)
	if err != nil {
		return nil, err
	}
	already := false
	if m, err := s.repo.FindMember(ctx, room.ID, userID); err != nil {
		return nil, err
	} else if m != nil {
		already = true
	}
	return &RoomPreview{
		TripID:        t.ID.String(),
		RoomID:        room.ID.String(),
		Title:         t.Title,
		Description:   t.Description,
		CoverImage:    t.CoverImage,
		StartDate:     t.StartDate.Format(dateLayout),
		EndDate:       t.EndDate.Format(dateLayout),
		Status:        string(effectiveStatus(*t)),
		Owner:         toUserSummaryPublic(*owner),
		MemberCount:   count,
		AlreadyJoined: already,
	}, nil
}

func (s *service) JoinRoom(ctx context.Context, userID uuid.UUID, p JoinRoomPayload) (*JoinRoomResponse, error) {
	code := strings.ToUpper(strings.TrimSpace(p.Code))
	if code == "" {
		return nil, ErrCodeRequired
	}

	room, err := s.repo.FindRoomByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, ErrRoomNotFound
	}

	// Idempotent: if already a member, return success with flag rather than 409.
	existing, err := s.repo.FindMember(ctx, room.ID, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return &JoinRoomResponse{
			TripID:        room.TripID.String(),
			RoomID:        room.ID.String(),
			AlreadyMember: true,
		}, nil
	}

	if err := s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.AddMember(ctx, &RoomMember{
			RoomID:   room.ID,
			UserID:   userID,
			Role:     RoleMember,
			JoinedAt: time.Now(),
		}); err != nil {
			return err
		}
		if s.cal != nil {
			return s.cal.OnRoomMemberAdded(ctx, tx.Tx(), room.TripID, userID)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	s.recordAudit(ctx, AuditRoomMemberJoined, audit.Success, userID, nil, AuditResourceRoom, room.ID.String(), nil)
	s.publish(ctx, event.EventRoomMemberJoined, userID, room.ID, map[string]any{
		"room_id": room.ID.String(),
		"trip_id": room.TripID.String(),
		"user_id": userID.String(),
		"via":     "code",
	})

	return &JoinRoomResponse{
		TripID:        room.TripID.String(),
		RoomID:        room.ID.String(),
		AlreadyMember: false,
	}, nil
}

func (s *service) LeaveRoom(ctx context.Context, userID, tripID uuid.UUID) error {
	t, err := s.repo.FindTripByID(ctx, tripID)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrTripNotFound
	}
	if t.OwnerID == userID {
		return ErrOwnerCannotLeave
	}
	room, err := s.repo.FindRoomByTripID(ctx, tripID)
	if err != nil {
		return err
	}
	if room == nil {
		return ErrRoomNotFound
	}
	m, err := s.repo.FindMember(ctx, room.ID, userID)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrNotMember
	}
	if err := s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.RemoveMember(ctx, room.ID, userID); err != nil {
			return err
		}
		if s.cal != nil {
			return s.cal.OnRoomMemberRemoved(ctx, tx.Tx(), tripID, userID)
		}
		return nil
	}); err != nil {
		return err
	}
	s.recordAudit(ctx, AuditRoomMemberLeft, audit.Success, userID, nil, AuditResourceRoom, room.ID.String(), nil)
	s.publish(ctx, event.EventRoomMemberLeft, userID, room.ID, map[string]any{
		"room_id": room.ID.String(),
		"trip_id": tripID.String(),
		"user_id": userID.String(),
	})
	return nil
}

func (s *service) RemoveMember(ctx context.Context, actorID, tripID, targetID uuid.UUID) error {
	t, err := s.repo.FindTripByID(ctx, tripID)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrTripNotFound
	}
	if targetID == t.OwnerID {
		return ErrCannotRemoveOwner
	}
	if err := s.requireAdmin(ctx, tripID, actorID, t.OwnerID); err != nil {
		return err
	}
	room, err := s.repo.FindRoomByTripID(ctx, tripID)
	if err != nil {
		return err
	}
	if room == nil {
		return ErrRoomNotFound
	}
	m, err := s.repo.FindMember(ctx, room.ID, targetID)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrNotMember
	}
	if err := s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.RemoveMember(ctx, room.ID, targetID); err != nil {
			return err
		}
		if s.cal != nil {
			return s.cal.OnRoomMemberRemoved(ctx, tx.Tx(), tripID, targetID)
		}
		return nil
	}); err != nil {
		return err
	}
	s.recordAudit(ctx, AuditRoomMemberRemoved, audit.Success, actorID, &targetID, AuditResourceRoom, room.ID.String(), nil)
	s.publish(ctx, event.EventRoomMemberRemoved, actorID, room.ID, map[string]any{
		"room_id":   room.ID.String(),
		"trip_id":   tripID.String(),
		"target_id": targetID.String(),
	})
	return nil
}

func (s *service) RegenerateCode(ctx context.Context, userID, tripID uuid.UUID) (*RoomResponse, error) {
	t, err := s.repo.FindTripByID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTripNotFound
	}
	if err := s.requireAdmin(ctx, tripID, userID, t.OwnerID); err != nil {
		return nil, err
	}
	room, err := s.repo.FindRoomByTripID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, ErrRoomNotFound
	}
	code, err := s.uniqueRoomCode(ctx, s.repo)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateRoomCode(ctx, room.ID, code); err != nil {
		return nil, err
	}
	s.recordAudit(ctx, AuditRoomCodeRegenerated, audit.Success, userID, nil, AuditResourceRoom, room.ID.String(), nil)
	s.publish(ctx, event.EventRoomCodeRegenerated, userID, room.ID, map[string]any{
		"room_id":  room.ID.String(),
		"trip_id":  tripID.String(),
		"new_code": code,
	})
	return s.GetRoom(ctx, userID, tripID)
}

// ---- Itinerary ----

func (s *service) CreateItinerary(ctx context.Context, userID, tripID uuid.UUID, p CreateItineraryPayload) (*ItineraryResponse, error) {
	t, err := s.repo.FindTripByID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTripNotFound
	}
	if _, err := s.requireRoomMember(ctx, tripID, userID, t.OwnerID); err != nil {
		return nil, err
	}
	it := &Itinerary{
		TripID:      tripID,
		Day:         p.Day,
		Title:       strings.TrimSpace(p.Title),
		Description: p.Description,
		StartTime:   p.StartTime,
		EndTime:     p.EndTime,
		Location:    p.Location,
		Latitude:    p.Latitude,
		Longitude:   p.Longitude,
		SortOrder:   p.SortOrder,
		CreatedBy:   userID,
	}
	if err := s.repo.CreateItinerary(ctx, it); err != nil {
		return nil, err
	}
	return itineraryToResponse(*it), nil
}

func (s *service) ListItinerary(ctx context.Context, userID, tripID uuid.UUID) ([]ItineraryResponse, error) {
	t, err := s.repo.FindTripByID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTripNotFound
	}
	if _, err := s.requireRoomMember(ctx, tripID, userID, t.OwnerID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListItinerary(ctx, tripID)
	if err != nil {
		return nil, err
	}
	out := make([]ItineraryResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, *itineraryToResponse(r))
	}
	return out, nil
}

func (s *service) UpdateItinerary(ctx context.Context, userID, tripID, itemID uuid.UUID, p UpdateItineraryPayload) (*ItineraryResponse, error) {
	it, err := s.repo.FindItineraryByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if it == nil || it.TripID != tripID {
		return nil, ErrItineraryNotFound
	}
	t, err := s.repo.FindTripByID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTripNotFound
	}
	if err := s.requireWriter(ctx, tripID, userID, t.OwnerID, it.CreatedBy); err != nil {
		return nil, err
	}
	patch := map[string]any{}
	if p.Day != nil {
		patch["day"] = *p.Day
	}
	if p.Title != nil {
		patch["title"] = strings.TrimSpace(*p.Title)
	}
	if p.Description != nil {
		patch["description"] = *p.Description
	}
	if p.StartTime != nil {
		patch["start_time"] = *p.StartTime
	}
	if p.EndTime != nil {
		patch["end_time"] = *p.EndTime
	}
	if p.Location != nil {
		patch["location"] = *p.Location
	}
	if p.Latitude != nil {
		patch["latitude"] = *p.Latitude
	}
	if p.Longitude != nil {
		patch["longitude"] = *p.Longitude
	}
	if p.SortOrder != nil {
		patch["sort_order"] = *p.SortOrder
	}
	if err := s.repo.UpdateItinerary(ctx, itemID, patch); err != nil {
		return nil, err
	}
	updated, err := s.repo.FindItineraryByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	return itineraryToResponse(*updated), nil
}

func (s *service) ReorderItinerary(ctx context.Context, userID, tripID uuid.UUID, p ReorderItineraryPayload) error {
	t, err := s.repo.FindTripByID(ctx, tripID)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrTripNotFound
	}
	if _, err := s.requireRoomMember(ctx, tripID, userID, t.OwnerID); err != nil {
		return err
	}
	ids := make([]uuid.UUID, 0, len(p.ItemIDs))
	for _, raw := range p.ItemIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			return ErrInvalidPayload
		}
		ids = append(ids, id)
	}
	return s.repo.ReorderItinerary(ctx, tripID, ids)
}

func (s *service) DeleteItinerary(ctx context.Context, userID, tripID, itemID uuid.UUID) error {
	it, err := s.repo.FindItineraryByID(ctx, itemID)
	if err != nil {
		return err
	}
	if it == nil || it.TripID != tripID {
		return ErrItineraryNotFound
	}
	t, err := s.repo.FindTripByID(ctx, tripID)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrTripNotFound
	}
	if err := s.requireWriter(ctx, tripID, userID, t.OwnerID, it.CreatedBy); err != nil {
		return err
	}
	return s.repo.DeleteItinerary(ctx, itemID)
}

// ---- Todos ----

func (s *service) CreateTodo(ctx context.Context, userID, tripID uuid.UUID, p CreateTodoPayload) (*TodoResponse, error) {
	t, err := s.repo.FindTripByID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTripNotFound
	}
	if _, err := s.requireRoomMember(ctx, tripID, userID, t.OwnerID); err != nil {
		return nil, err
	}
	td := &Todo{
		TripID:    tripID,
		Title:     strings.TrimSpace(p.Title),
		DueDate:   p.DueDate,
		Priority:  TodoPriorityNormal,
		Tags:      normalizeTags(p.Tags),
		CreatedBy: userID,
	}
	if p.Priority != nil {
		td.Priority = TodoPriority(*p.Priority)
	}
	if p.AssigneeID != nil {
		aid, err := uuid.Parse(*p.AssigneeID)
		if err != nil {
			return nil, ErrInvalidPayload
		}
		td.AssigneeID = &aid
	}
	if err := s.repo.CreateTodo(ctx, td); err != nil {
		return nil, err
	}
	return todoToResponse(*td), nil
}

func (s *service) ListTodos(ctx context.Context, userID, tripID uuid.UUID) ([]TodoResponse, error) {
	t, err := s.repo.FindTripByID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTripNotFound
	}
	if _, err := s.requireRoomMember(ctx, tripID, userID, t.OwnerID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListTodos(ctx, tripID)
	if err != nil {
		return nil, err
	}
	out := make([]TodoResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, *todoToResponse(r))
	}
	return out, nil
}

func (s *service) UpdateTodo(ctx context.Context, userID, tripID, todoID uuid.UUID, p UpdateTodoPayload) (*TodoResponse, error) {
	td, err := s.repo.FindTodoByID(ctx, todoID)
	if err != nil {
		return nil, err
	}
	if td == nil || td.TripID != tripID {
		return nil, ErrTodoNotFound
	}
	t, err := s.repo.FindTripByID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTripNotFound
	}
	// Toggling completion / reassigning is allowed for any room member.
	// Edits to title/due_date and delete are gated by requireWriter at their
	// own entry points. Permission here: must be a room member.
	if _, err := s.requireRoomMember(ctx, tripID, userID, t.OwnerID); err != nil {
		return nil, err
	}
	// Title edits, however, are restricted to creator/admin/owner.
	if p.Title != nil {
		if err := s.requireWriter(ctx, tripID, userID, t.OwnerID, td.CreatedBy); err != nil {
			return nil, err
		}
	}
	patch := map[string]any{}
	if p.Title != nil {
		patch["title"] = strings.TrimSpace(*p.Title)
	}
	if p.IsCompleted != nil {
		patch["is_completed"] = *p.IsCompleted
	}
	if p.DueDate != nil {
		patch["due_date"] = *p.DueDate
	}
	if p.AssigneeID != nil {
		aid, err := uuid.Parse(*p.AssigneeID)
		if err != nil {
			return nil, ErrInvalidPayload
		}
		patch["assignee_id"] = aid
	}
	if p.Priority != nil {
		patch["priority"] = *p.Priority
	}
	if p.Tags != nil {
		patch["tags"] = normalizeTags(*p.Tags)
	}
	if p.SortOrder != nil {
		patch["sort_order"] = *p.SortOrder
	}
	if err := s.repo.UpdateTodo(ctx, todoID, patch); err != nil {
		return nil, err
	}
	updated, err := s.repo.FindTodoByID(ctx, todoID)
	if err != nil {
		return nil, err
	}
	return todoToResponse(*updated), nil
}

func (s *service) ReorderTodos(ctx context.Context, userID, tripID uuid.UUID, p ReorderTodosPayload) error {
	t, err := s.repo.FindTripByID(ctx, tripID)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrTripNotFound
	}
	if _, err := s.requireRoomMember(ctx, tripID, userID, t.OwnerID); err != nil {
		return err
	}
	ids := make([]uuid.UUID, 0, len(p.TodoIDs))
	for _, raw := range p.TodoIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			return ErrInvalidPayload
		}
		ids = append(ids, id)
	}
	return s.repo.ReorderTodos(ctx, tripID, ids)
}

func (s *service) DeleteTodo(ctx context.Context, userID, tripID, todoID uuid.UUID) error {
	td, err := s.repo.FindTodoByID(ctx, todoID)
	if err != nil {
		return err
	}
	if td == nil || td.TripID != tripID {
		return ErrTodoNotFound
	}
	t, err := s.repo.FindTripByID(ctx, tripID)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrTripNotFound
	}
	if err := s.requireWriter(ctx, tripID, userID, t.OwnerID, td.CreatedBy); err != nil {
		return err
	}
	return s.repo.DeleteTodo(ctx, todoID)
}

// ---- Helpers ----

// requireRoomMember returns the room_members row for (tripID, userID).
// The trip owner is implicitly treated as a room admin even if the
// room_members row is missing (it shouldn't be, but defense in depth).
func (s *service) requireRoomMember(ctx context.Context, tripID, userID, ownerID uuid.UUID) (*RoomMember, error) {
	m, err := s.repo.FindMemberByTrip(ctx, tripID, userID)
	if err != nil {
		return nil, err
	}
	if m != nil {
		return m, nil
	}
	if userID == ownerID {
		return &RoomMember{UserID: userID, Role: RoleAdmin}, nil
	}
	return nil, ErrForbidden
}

func (s *service) requireAdmin(ctx context.Context, tripID, userID, ownerID uuid.UUID) error {
	m, err := s.requireRoomMember(ctx, tripID, userID, ownerID)
	if err != nil {
		return err
	}
	if m.Role != RoleAdmin && userID != ownerID {
		return ErrForbidden
	}
	return nil
}

// requireWriter: itinerary/todo edits and deletes allowed only for the
// creator, the trip owner, or a room admin.
func (s *service) requireWriter(ctx context.Context, tripID, userID, ownerID, createdBy uuid.UUID) error {
	if userID == ownerID || userID == createdBy {
		return nil
	}
	m, err := s.repo.FindMemberByTrip(ctx, tripID, userID)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrForbidden
	}
	if m.Role == RoleAdmin {
		return nil
	}
	return ErrForbidden
}

func (s *service) uniqueRoomCode(ctx context.Context, r IRepository) (string, error) {
	for i := 0; i < 8; i++ {
		code, err := randomRoomCode()
		if err != nil {
			return "", err
		}
		exists, err := r.RoomCodeExists(ctx, code)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
	return "", fmt.Errorf("could not allocate unique room code after retries")
}

func (s *service) buildTripResponse(ctx context.Context, t Trip, room *Room, memberCount int, role string, owner auth.User) (*TripResponse, error) {
	resp := &TripResponse{
		ID:          t.ID.String(),
		Owner:       toUserSummary(owner),
		Title:       t.Title,
		Description: t.Description,
		CoverImage:  t.CoverImage,
		StartDate:   t.StartDate.Format(dateLayout),
		EndDate:     t.EndDate.Format(dateLayout),
		TimeZone:    t.TimeZone,
		Status:      string(effectiveStatus(t)),
		Visibility:  string(effectiveVisibility(t)),
		Location:    t.Location,
		Latitude:    t.Latitude,
		Longitude:   t.Longitude,
		MemberCount: memberCount,
		MyRole:      role,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
	if room != nil {
		resp.Room = &RoomBrief{ID: room.ID.String(), RoomCode: room.RoomCode}
	}
	return resp, nil
}

// syncTripCreated fans a freshly-created trip out to the calendar module.
// Called from within the trip creation tx so both rows commit together.
// The owner is the sole initial member.
func (s *service) syncTripCreated(ctx context.Context, tx IRepository, t *Trip, actorID uuid.UUID) error {
	if s.cal == nil {
		return nil
	}
	return s.cal.OnTripCreated(ctx, tx.Tx(), calendar.TripSnapshot{
		TripID:      t.ID,
		OwnerID:     t.OwnerID,
		Title:       t.Title,
		Description: t.Description,
		StartDate:   t.StartDate,
		EndDate:     t.EndDate,
		TimeZone:    t.TimeZone,
		MemberIDs:   []uuid.UUID{t.OwnerID},
		ActorID:     actorID,
	})
}

// syncTripUpdated re-reads the trip inside the tx so the snapshot reflects
// post-patch values, then hands it to the calendar sync.
func (s *service) syncTripUpdated(ctx context.Context, tx IRepository, tripID, actorID uuid.UUID) error {
	if s.cal == nil {
		return nil
	}
	t, err := tx.FindTripByID(ctx, tripID)
	if err != nil {
		return err
	}
	if t == nil {
		return nil
	}
	return s.cal.OnTripUpdated(ctx, tx.Tx(), calendar.TripSnapshot{
		TripID:      t.ID,
		OwnerID:     t.OwnerID,
		Title:       t.Title,
		Description: t.Description,
		StartDate:   t.StartDate,
		EndDate:     t.EndDate,
		TimeZone:    t.TimeZone,
		ActorID:     actorID,
	})
}

func (s *service) recordAudit(
	ctx context.Context,
	action audit.Action,
	status audit.Status,
	actorID uuid.UUID,
	targetID *uuid.UUID,
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
	if targetID != nil && *targetID != uuid.Nil {
		t := *targetID
		log.TargetUserID = &t
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

// ---- Pure helpers ----

// effectiveVisibility returns the stored visibility, defaulting to private
// for rows that predate the visibility column being added.
func effectiveVisibility(t Trip) TripVisibility {
	if t.Visibility == "" {
		return TripVisibilityPrivate
	}
	return t.Visibility
}

// effectiveStatus returns TripCompleted once the trip's end date has passed,
// regardless of the stored status. Otherwise it returns the stored status.
// Owners cannot set "completed" manually; it is derived from time.
func effectiveStatus(t Trip) TripStatus {
	if t.Status == TripCompleted {
		return TripCompleted
	}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if today.After(t.EndDate) {
		return TripCompleted
	}
	return t.Status
}

func normalizeTimeZone(tz string) string {
	tz = strings.TrimSpace(tz)
	if tz == "" {
		return "UTC"
	}
	return tz
}

func validateTimeZone(tz string) error {
	if _, err := time.LoadLocation(tz); err != nil {
		return ErrInvalidPayload
	}
	return nil
}

func parseDateRange(startStr, endStr string) (time.Time, time.Time, error) {
	start, err := time.Parse(dateLayout, startStr)
	if err != nil {
		return time.Time{}, time.Time{}, ErrInvalidPayload
	}
	end, err := time.Parse(dateLayout, endStr)
	if err != nil {
		return time.Time{}, time.Time{}, ErrInvalidPayload
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, ErrInvalidDateRange
	}
	return start, end, nil
}

// randomRoomCode returns a human-friendly 8-char code (A-Z + 2-9, no
// ambiguous chars). 56^8 ≈ 9.6e13 possibilities — collisions rare but
// caller still retries on conflict.
func randomRoomCode() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, 8)
	for i, v := range b {
		out[i] = alphabet[int(v)%len(alphabet)]
	}
	return string(out), nil
}

func toUserSummary(u auth.User) UserSummary {
	return UserSummary{
		ID:        u.ID.String(),
		Email:     u.Email,
		Name:      u.Name,
		AvatarURL: u.AvatarURL,
	}
}

// toUserSummaryPublic omits email — used in unauthenticated/preview responses.
func toUserSummaryPublic(u auth.User) UserSummary {
	return UserSummary{
		ID:        u.ID.String(),
		Name:      u.Name,
		AvatarURL: u.AvatarURL,
	}
}

func itineraryToResponse(it Itinerary) *ItineraryResponse {
	return &ItineraryResponse{
		ID:          it.ID.String(),
		TripID:      it.TripID.String(),
		Day:         it.Day,
		Title:       it.Title,
		Description: it.Description,
		StartTime:   it.StartTime,
		EndTime:     it.EndTime,
		Location:    it.Location,
		Latitude:    it.Latitude,
		Longitude:   it.Longitude,
		SortOrder:   it.SortOrder,
		CreatedBy:   it.CreatedBy.String(),
		CreatedAt:   it.CreatedAt,
		UpdatedAt:   it.UpdatedAt,
	}
}

func todoToResponse(td Todo) *TodoResponse {
	priority := string(td.Priority)
	if priority == "" {
		priority = string(TodoPriorityNormal)
	}
	tags := []string(td.Tags)
	if tags == nil {
		tags = []string{}
	}
	resp := &TodoResponse{
		ID:          td.ID.String(),
		TripID:      td.TripID.String(),
		Title:       td.Title,
		IsCompleted: td.IsCompleted,
		DueDate:     td.DueDate,
		Priority:    priority,
		Tags:        tags,
		SortOrder:   td.SortOrder,
		CreatedBy:   td.CreatedBy.String(),
		CreatedAt:   td.CreatedAt,
		UpdatedAt:   td.UpdatedAt,
	}
	if td.AssigneeID != nil {
		s := td.AssigneeID.String()
		resp.AssigneeID = &s
	}
	return resp
}

// normalizeTags trims whitespace, drops empties, lowercases, and de-duplicates
// while preserving first-seen order. Keeps tag storage canonical so filtering
// in the UI is stable.
func normalizeTags(raw []string) []string {
	if len(raw) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, t := range raw {
		v := strings.ToLower(strings.TrimSpace(t))
		if v == "" {
			continue
		}
		if _, dup := seen[v]; dup {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
