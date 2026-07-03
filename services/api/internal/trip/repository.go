package trip

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrStaleVersion is returned by *WithVersion repo methods when the caller's
// `expectedVersion` does not match the row's current version (LWW conflict).
var ErrStaleVersion = errors.New("stale version")

type IRepository interface {
	WithTx(ctx context.Context, fn func(IRepository) error) error
	Tx() *gorm.DB

	// Users
	FindUserByID(ctx context.Context, id uuid.UUID) (*auth.User, error)
	FindUsersByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error)

	// Trip
	CreateTrip(ctx context.Context, t *Trip) error
	FindTripByID(ctx context.Context, id uuid.UUID) (*Trip, error)
	UpdateTrip(ctx context.Context, id uuid.UUID, patch map[string]any) error
	DeleteTrip(ctx context.Context, id uuid.UUID) error
	ListTrips(ctx context.Context, userID uuid.UUID, q ListTripsQuery) ([]Trip, error)
	CountMembers(ctx context.Context, tripIDs []uuid.UUID) (map[uuid.UUID]int, error)

	// Room
	CreateRoom(ctx context.Context, r *Room) error
	FindRoomByID(ctx context.Context, id uuid.UUID) (*Room, error)
	FindRoomByTripID(ctx context.Context, tripID uuid.UUID) (*Room, error)
	FindRoomByCode(ctx context.Context, code string) (*Room, error)
	UpdateRoomCode(ctx context.Context, roomID uuid.UUID, code string) error
	RoomCodeExists(ctx context.Context, code string) (bool, error)

	// Room members
	AddMember(ctx context.Context, m *RoomMember) error
	RemoveMember(ctx context.Context, roomID, userID uuid.UUID) error
	FindMember(ctx context.Context, roomID, userID uuid.UUID) (*RoomMember, error)
	ListMembers(ctx context.Context, roomID uuid.UUID) ([]RoomMember, error)
	FindMemberByTrip(ctx context.Context, tripID, userID uuid.UUID) (*RoomMember, error)
	CountRoomMembers(ctx context.Context, roomID uuid.UUID) (int, error)
	IsRoomMember(ctx context.Context, tripID, userID uuid.UUID) (bool, error)

	// Itinerary
	CreateItinerary(ctx context.Context, it *Itinerary) error
	FindItineraryByID(ctx context.Context, id uuid.UUID) (*Itinerary, error)
	UpdateItinerary(ctx context.Context, id uuid.UUID, patch map[string]any) error
	UpdateItineraryWithVersion(ctx context.Context, id uuid.UUID, expectedVersion int, patch map[string]any, event *EventMeta) (*Itinerary, error)
	DeleteItinerary(ctx context.Context, id uuid.UUID) error
	ListItinerary(ctx context.Context, tripID uuid.UUID) ([]Itinerary, error)
	ReorderItinerary(ctx context.Context, tripID uuid.UUID, itemIDs []uuid.UUID) error

	// Todo
	CreateTodo(ctx context.Context, todo *Todo) error
	FindTodoByID(ctx context.Context, id uuid.UUID) (*Todo, error)
	UpdateTodo(ctx context.Context, id uuid.UUID, patch map[string]any) error
	UpdateTodoWithVersion(ctx context.Context, id uuid.UUID, expectedVersion int, patch map[string]any, event *EventMeta) (*Todo, error)
	DeleteTodo(ctx context.Context, id uuid.UUID) error
	ListTodos(ctx context.Context, tripID uuid.UUID) ([]Todo, error)
	ReorderTodos(ctx context.Context, tripID uuid.UUID, todoIDs []uuid.UUID) error

	// Outbox
	EnqueueOutbox(ctx context.Context, entry *Outbox) error
	ClaimOutbox(ctx context.Context, limit int) ([]Outbox, error)
	MarkOutboxDispatched(ctx context.Context, ids []uuid.UUID) error
	RecordOutboxFailure(ctx context.Context, id uuid.UUID, errMsg string) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) WithTx(ctx context.Context, fn func(IRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&repository{db: tx})
	})
}

// For cross-module sync layers
func (r *repository) Tx() *gorm.DB {
	return r.db
}

// ---- Users ----

func (r *repository) FindUserByID(ctx context.Context, id uuid.UUID) (*auth.User, error) {
	var u auth.User
	err := r.db.WithContext(ctx).First(&u, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (r *repository) FindUsersByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
	out := make(map[uuid.UUID]auth.User, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var users []auth.User
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, u := range users {
		out[u.ID] = u
	}
	return out, nil
}

// ---- Trip ----

func (r *repository) CreateTrip(ctx context.Context, t *Trip) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *repository) FindTripByID(ctx context.Context, id uuid.UUID) (*Trip, error) {
	var t Trip
	err := r.db.WithContext(ctx).First(&t, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &t, err
}

func (r *repository) UpdateTrip(ctx context.Context, id uuid.UUID, patch map[string]any) error {
	if len(patch) == 0 {
		return nil
	}
	patch["updated_at"] = time.Now()
	return r.db.WithContext(ctx).Model(&Trip{}).Where("id = ?", id).Updates(patch).Error
}

func (r *repository) DeleteTrip(ctx context.Context, id uuid.UUID) error {
	// GORM soft delete: sets deleted_at, cascades to rooms via FK ON DELETE
	// only on hard-delete — so we soft-delete the trip but also explicitly
	// soft-delete its dependent itineraries/todos.
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&Trip{}, "id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Where("trip_id = ?", id).Delete(&Itinerary{}).Error; err != nil {
			return err
		}
		return tx.Where("trip_id = ?", id).Delete(&Todo{}).Error
	})
}

func (r *repository) ListTrips(ctx context.Context, userID uuid.UUID, q ListTripsQuery) ([]Trip, error) {
	limit := q.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	tx := r.db.WithContext(ctx).
		Model(&Trip{}).
		Distinct("trip.trip.*").
		Joins("LEFT JOIN trip.rooms r ON r.trip_id = trip.trip.id").
		Joins("LEFT JOIN trip.room_members rm ON rm.room_id = r.id AND rm.user_id = ?", userID)

	switch strings.ToLower(q.Role) {
	case "owner":
		tx = tx.Where("trip.trip.owner_id = ?", userID)
	case "member":
		tx = tx.Where("rm.user_id = ? AND trip.trip.owner_id <> ?", userID, userID)
	default:
		tx = tx.Where("trip.trip.owner_id = ? OR rm.user_id = ?", userID, userID)
	}

	if q.Status != "" {
		switch strings.ToLower(q.Status) {
		case string(TripCompleted):
			tx = tx.Where("trip.trip.status = ? OR trip.trip.end_date < CURRENT_DATE", TripCompleted)
		case string(TripPlanning), string(TripOngoing):
			tx = tx.Where("trip.trip.status = ? AND trip.trip.end_date >= CURRENT_DATE", q.Status)
		default:
			tx = tx.Where("trip.trip.status = ?", q.Status)
		}
	}
	if s := strings.TrimSpace(q.Q); s != "" {
		tx = tx.Where("LOWER(trip.trip.title) LIKE ?", "%"+strings.ToLower(s)+"%")
	}

	var trips []Trip
	err := tx.Order("trip.trip.start_date DESC").
		Limit(limit).Offset(q.Offset).
		Find(&trips).Error
	return trips, err
}

func (r *repository) CountMembers(ctx context.Context, tripIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	out := make(map[uuid.UUID]int, len(tripIDs))
	if len(tripIDs) == 0 {
		return out, nil
	}
	type row struct {
		TripID uuid.UUID
		N      int
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Raw(`SELECT r.trip_id AS trip_id, COUNT(rm.user_id) AS n
		     FROM trip.rooms r
		     LEFT JOIN trip.room_members rm ON rm.room_id = r.id
		     WHERE r.trip_id IN ?
		     GROUP BY r.trip_id`, tripIDs).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, x := range rows {
		out[x.TripID] = x.N
	}
	return out, nil
}

// ---- Room ----

func (r *repository) CreateRoom(ctx context.Context, room *Room) error {
	if room.ID == uuid.Nil {
		room.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(room).Error
}

func (r *repository) FindRoomByID(ctx context.Context, id uuid.UUID) (*Room, error) {
	var room Room
	err := r.db.WithContext(ctx).First(&room, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &room, err
}

func (r *repository) FindRoomByTripID(ctx context.Context, tripID uuid.UUID) (*Room, error) {
	var room Room
	err := r.db.WithContext(ctx).First(&room, "trip_id = ?", tripID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &room, err
}

func (r *repository) FindRoomByCode(ctx context.Context, code string) (*Room, error) {
	var room Room
	err := r.db.WithContext(ctx).First(&room, "room_code = ?", code).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &room, err
}

func (r *repository) UpdateRoomCode(ctx context.Context, roomID uuid.UUID, code string) error {
	return r.db.WithContext(ctx).Model(&Room{}).
		Where("id = ?", roomID).
		Updates(map[string]any{"room_code": code, "updated_at": time.Now()}).Error
}

func (r *repository) RoomCodeExists(ctx context.Context, code string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Room{}).Where("room_code = ?", code).Count(&n).Error
	return n > 0, err
}

// ---- Room members ----

func (r *repository) AddMember(ctx context.Context, m *RoomMember) error {
	if m.JoinedAt.IsZero() {
		m.JoinedAt = time.Now()
	}
	return r.db.WithContext(ctx).
		Exec(`INSERT INTO trip.room_members (room_id, user_id, role, joined_at)
		      VALUES (?, ?, ?, ?)
		      ON CONFLICT (room_id, user_id) DO NOTHING`,
			m.RoomID, m.UserID, m.Role, m.JoinedAt).Error
}

func (r *repository) RemoveMember(ctx context.Context, roomID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Delete(&RoomMember{}).Error
}

func (r *repository) FindMember(ctx context.Context, roomID, userID uuid.UUID) (*RoomMember, error) {
	var m RoomMember
	err := r.db.WithContext(ctx).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &m, err
}

func (r *repository) ListMembers(ctx context.Context, roomID uuid.UUID) ([]RoomMember, error) {
	var rows []RoomMember
	err := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Order("joined_at ASC").Find(&rows).Error
	return rows, err
}

func (r *repository) FindMemberByTrip(ctx context.Context, tripID, userID uuid.UUID) (*RoomMember, error) {
	var m RoomMember
	err := r.db.WithContext(ctx).
		Raw(`SELECT rm.* FROM trip.room_members rm
		     JOIN trip.rooms r ON r.id = rm.room_id
		     WHERE r.trip_id = ? AND rm.user_id = ?`, tripID, userID).
		Scan(&m).Error
	if err != nil {
		return nil, err
	}
	if m.UserID == uuid.Nil {
		return nil, nil
	}
	return &m, nil
}

func (r *repository) CountRoomMembers(ctx context.Context, roomID uuid.UUID) (int, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&RoomMember{}).
		Where("room_id = ?", roomID).Count(&n).Error
	return int(n), err
}

func (r *repository) IsRoomMember(ctx context.Context, tripID, userID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM (
			SELECT 1 FROM trip.trip WHERE id = ? AND owner_id = ? AND deleted_at IS NULL
			UNION ALL
			SELECT 1 FROM trip.room_members rm
			  JOIN trip.rooms r ON r.id = rm.room_id
			 WHERE r.trip_id = ? AND rm.user_id = ?
		) AS x`, tripID, userID, tripID, userID).Scan(&n).Error
	return n > 0, err
}

// ---- Itinerary ----

func (r *repository) CreateItinerary(ctx context.Context, it *Itinerary) error {
	if it.ID == uuid.Nil {
		it.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(it).Error
}

func (r *repository) FindItineraryByID(ctx context.Context, id uuid.UUID) (*Itinerary, error) {
	var it Itinerary
	err := r.db.WithContext(ctx).First(&it, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &it, err
}

func (r *repository) UpdateItinerary(ctx context.Context, id uuid.UUID, patch map[string]any) error {
	if len(patch) == 0 {
		return nil
	}
	patch["updated_at"] = time.Now()
	return r.db.WithContext(ctx).Model(&Itinerary{}).Where("id = ?", id).Updates(patch).Error
}

func (r *repository) UpdateItineraryWithVersion(ctx context.Context, id uuid.UUID, expectedVersion int, patch map[string]any, event *EventMeta) (*Itinerary, error) {
	var updated Itinerary
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		u, err := updateItineraryReturning(tx, id, expectedVersion, patch)
		if err != nil {
			return err
		}
		updated = *u
		if event == nil {
			return nil
		}
		return insertOutboxRow(tx, event, updated.TripID, updated.Version, &updated)
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func updateItineraryReturning(db *gorm.DB, id uuid.UUID, expectedVersion int, patch map[string]any) (*Itinerary, error) {
	cp := make(map[string]any, len(patch)+2)
	for k, v := range patch {
		cp[k] = v
	}
	cp["version"] = gorm.Expr("version + 1")
	cp["updated_at"] = time.Now()

	var updated Itinerary
	res := db.Model(&updated).
		Clauses(clause.Returning{}).
		Where("id = ? AND version = ?", id, expectedVersion).
		Updates(cp)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		var probe Itinerary
		err := db.Select("id").First(&probe, "id = ?", id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrItineraryNotFound
		}
		if err != nil {
			return nil, err
		}
		return nil, ErrStaleVersion
	}
	return &updated, nil
}

func (r *repository) DeleteItinerary(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Itinerary{}, "id = ?", id).Error
}

func (r *repository) ListItinerary(ctx context.Context, tripID uuid.UUID) ([]Itinerary, error) {
	var rows []Itinerary
	err := r.db.WithContext(ctx).
		Where("trip_id = ?", tripID).
		Order("day ASC, sort_order ASC, start_time ASC").Find(&rows).Error
	return rows, err
}

// The trip_id guard prevents one trip's payload from
// touching another trip's rows even if a malicious id slips through.
func (r *repository) ReorderItinerary(ctx context.Context, tripID uuid.UUID, itemIDs []uuid.UUID) error {
	if len(itemIDs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, id := range itemIDs {
			if err := tx.Model(&Itinerary{}).
				Where("id = ? AND trip_id = ?", id, tripID).
				Update("sort_order", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ---- Todo ----

func (r *repository) CreateTodo(ctx context.Context, t *Todo) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *repository) FindTodoByID(ctx context.Context, id uuid.UUID) (*Todo, error) {
	var t Todo
	err := r.db.WithContext(ctx).First(&t, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &t, err
}

func (r *repository) UpdateTodo(ctx context.Context, id uuid.UUID, patch map[string]any) error {
	if len(patch) == 0 {
		return nil
	}
	patch["updated_at"] = time.Now()
	return r.db.WithContext(ctx).Model(&Todo{}).Where("id = ?", id).Updates(patch).Error
}

func (r *repository) UpdateTodoWithVersion(ctx context.Context, id uuid.UUID, expectedVersion int, patch map[string]any, event *EventMeta) (*Todo, error) {
	var updated Todo
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		u, err := updateTodoReturning(tx, id, expectedVersion, patch)
		if err != nil {
			return err
		}
		updated = *u
		if event == nil {
			return nil
		}
		return insertOutboxRow(tx, event, updated.TripID, updated.Version, &updated)
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

// updateTodoReturning mirrors updateItineraryReturning for todos.
func updateTodoReturning(db *gorm.DB, id uuid.UUID, expectedVersion int, patch map[string]any) (*Todo, error) {
	cp := make(map[string]any, len(patch)+2)
	for k, v := range patch {
		cp[k] = v
	}
	cp["version"] = gorm.Expr("version + 1")
	cp["updated_at"] = time.Now()

	var updated Todo
	res := db.Model(&updated).
		Clauses(clause.Returning{}).
		Where("id = ? AND version = ?", id, expectedVersion).
		Updates(cp)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		var probe Todo
		err := db.Select("id").First(&probe, "id = ?", id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTodoNotFound
		}
		if err != nil {
			return nil, err
		}
		return nil, ErrStaleVersion
	}
	return &updated, nil
}

func insertOutboxRow(tx *gorm.DB, event *EventMeta, tripID uuid.UUID, version int, entity any) error {
	data, err := json.Marshal(entity)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(OutboxEvent{
		OpType:  event.OpType,
		TripID:  tripID,
		UserID:  event.UserID,
		Version: version,
		TraceID: event.TraceID,
		Data:    data,
	})
	if err != nil {
		return err
	}
	return tx.Create(&Outbox{
		AggregateID: tripID,
		Stream:      event.Stream,
		TraceID:     event.TraceID,
		Payload:     payload,
	}).Error
}

// ---- Outbox ----

func (r *repository) EnqueueOutbox(ctx context.Context, entry *Outbox) error {
	if entry == nil {
		return nil
	}
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *repository) ClaimOutbox(ctx context.Context, limit int) ([]Outbox, error) {
	if limit <= 0 {
		limit = 32
	}
	var rows []Outbox
	err := r.db.WithContext(ctx).
		Raw(`SELECT * FROM trip.outbox
		     WHERE dispatched_at IS NULL
		     ORDER BY created_at ASC
		     LIMIT ?
		     FOR UPDATE SKIP LOCKED`, limit).
		Scan(&rows).Error
	return rows, err
}

func (r *repository) MarkOutboxDispatched(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&Outbox{}).
		Where("id IN ?", ids).
		Updates(map[string]any{"dispatched_at": now}).Error
}

func (r *repository) RecordOutboxFailure(ctx context.Context, id uuid.UUID, errMsg string) error {
	// Truncate error blob defensively — a repeated 10 MB stringified error
	// would balloon the row.
	const maxErrLen = 512
	if len(errMsg) > maxErrLen {
		errMsg = errMsg[:maxErrLen]
	}
	return r.db.WithContext(ctx).
		Model(&Outbox{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"attempts":   gorm.Expr("attempts + 1"),
			"last_error": errMsg,
		}).Error
}

func (r *repository) DeleteTodo(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Todo{}, "id = ?", id).Error
}

func (r *repository) ListTodos(ctx context.Context, tripID uuid.UUID) ([]Todo, error) {
	var rows []Todo
	err := r.db.WithContext(ctx).
		Where("trip_id = ?", tripID).
		Order("is_completed ASC, sort_order ASC, due_date ASC NULLS LAST, created_at DESC").
		Find(&rows).Error
	return rows, err
}

func (r *repository) ReorderTodos(ctx context.Context, tripID uuid.UUID, todoIDs []uuid.UUID) error {
	if len(todoIDs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, id := range todoIDs {
			if err := tx.Model(&Todo{}).
				Where("id = ? AND trip_id = ?", id, tripID).
				Update("sort_order", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
