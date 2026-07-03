package calendar

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrStaleVersion = errors.New("stale version")

type IRepository interface {
	WithTx(ctx context.Context, fn func(IRepository) error) error

	// Events
	CreateEvent(ctx context.Context, e *Event) error
	FindEventByID(ctx context.Context, id uuid.UUID) (*Event, error)
	FindEventBySource(ctx context.Context, srcType SourceType, srcID uuid.UUID) (*Event, error)
	UpdateEvent(ctx context.Context, id uuid.UUID, patch map[string]any) error
	UpdateEventWithVersion(ctx context.Context, id uuid.UUID, expectedVersion int, patch map[string]any, meta *OutboxMeta) (*Event, error)
	SoftDeleteEvent(ctx context.Context, id uuid.UUID) error
	SoftDeleteEventBySource(ctx context.Context, srcType SourceType, srcID uuid.UUID) error

	// Queries
	ListEventsForUser(ctx context.Context, userID uuid.UUID, q ListEventsQuery) ([]Event, error)
	ListRoomEventsForTrip(ctx context.Context, tripID uuid.UUID, q ListEventsQuery) ([]Event, error)
	ListFriendEventsForUser(ctx context.Context, friendID uuid.UUID, q ListEventsQuery) ([]Event, error)

	// Members
	AddMembers(ctx context.Context, eventID uuid.UUID, members []EventMember) error
	RemoveMember(ctx context.Context, eventID, userID uuid.UUID) error
	RemoveMemberBySource(ctx context.Context, srcType SourceType, srcID uuid.UUID, userID uuid.UUID) error
	ListEventMembers(ctx context.Context, eventID uuid.UUID) ([]EventMember, error)
	IsEventMember(ctx context.Context, eventID, userID uuid.UUID) (bool, error)

	// Outbox
	EnqueueOutbox(ctx context.Context, tripID uuid.UUID, payload []byte, traceID, stream string) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) IRepository {
	return &repository{db: db}
}

// NewRepositoryWithTx wraps an existing *gorm.DB tx. Trip.CalendarSync uses
// this to share the same transaction as trip mutations — the row inserted here
// and the trip row commit or roll back together.
func NewRepositoryWithTx(tx *gorm.DB) IRepository {
	return &repository{db: tx}
}

func (r *repository) WithTx(ctx context.Context, fn func(IRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&repository{db: tx})
	})
}

// ---- Events ----

func (r *repository) CreateEvent(ctx context.Context, e *Event) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *repository) FindEventByID(ctx context.Context, id uuid.UUID) (*Event, error) {
	var e Event
	err := r.db.WithContext(ctx).First(&e, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &e, err
}

func (r *repository) FindEventBySource(ctx context.Context, srcType SourceType, srcID uuid.UUID) (*Event, error) {
	var e Event
	err := r.db.WithContext(ctx).
		Where("source_type = ? AND source_id = ?", srcType, srcID).
		First(&e).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &e, err
}

func (r *repository) UpdateEvent(ctx context.Context, id uuid.UUID, patch map[string]any) error {
	if len(patch) == 0 {
		return nil
	}
	patch["updated_at"] = time.Now()
	return r.db.WithContext(ctx).Model(&Event{}).Where("id = ?", id).Updates(patch).Error
}

func (r *repository) UpdateEventWithVersion(ctx context.Context, id uuid.UUID, expectedVersion int, patch map[string]any, meta *OutboxMeta) (*Event, error) {
	updated, err := updateEventReturning(r.db.WithContext(ctx), id, expectedVersion, patch)
	if err != nil {
		return nil, err
	}

	if meta != nil {
		if err := insertCalendarOutboxRow(r.db, meta, updated); err != nil {
			return nil, err
		}
	}
	return updated, nil
}

func updateEventReturning(db *gorm.DB, id uuid.UUID, expectedVersion int, patch map[string]any) (*Event, error) {
	cp := make(map[string]any, len(patch)+2)
	for k, v := range patch {
		cp[k] = v
	}
	cp["version"] = gorm.Expr("version + 1")
	cp["updated_at"] = time.Now()

	var updated Event
	res := db.Model(&updated).
		Clauses(clause.Returning{}).
		Where("id = ? AND version = ? AND deleted_at IS NULL", id, expectedVersion).
		Updates(cp)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		var probe Event
		err := db.Select("id").First(&probe, "id = ?", id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		if err != nil {
			return nil, err
		}
		return nil, ErrStaleVersion
	}
	return &updated, nil
}

func (r *repository) SoftDeleteEvent(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Event{}, "id = ?", id).Error
}

func (r *repository) SoftDeleteEventBySource(ctx context.Context, srcType SourceType, srcID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("source_type = ? AND source_id = ?", srcType, srcID).
		Delete(&Event{}).Error
}

// ---- Queries ----

// ListEventsForUser returns every event whose event_members row contains userID.
// Visibility is intentionally NOT filtered here
func (r *repository) ListEventsForUser(ctx context.Context, userID uuid.UUID, q ListEventsQuery) ([]Event, error) {
	tx := r.db.WithContext(ctx).
		Model(&Event{}).
		Joins("JOIN calendar.event_members m ON m.event_id = calendar.events.id").
		Where("m.user_id = ?", userID)
	tx = applyRangeFilter(tx, q)
	var events []Event
	err := tx.Order("start_at ASC").Find(&events).Error
	return events, err
}

// ListRoomEventsForTrip returns the trip's room-visible events.
func (r *repository) ListRoomEventsForTrip(ctx context.Context, tripID uuid.UUID, q ListEventsQuery) ([]Event, error) {
	tx := r.db.WithContext(ctx).
		Model(&Event{}).
		Where("source_type = ? AND source_id = ?", SourceTrip, tripID).
		Where("visibility = ?", VisibilityRoom)
	tx = applyRangeFilter(tx, q)
	var events []Event
	err := tx.Order("start_at ASC").Find(&events).Error
	return events, err
}

// ListFriendEventsForUser returns the friend's `visibility=friends` events.
func (r *repository) ListFriendEventsForUser(ctx context.Context, friendID uuid.UUID, q ListEventsQuery) ([]Event, error) {
	tx := r.db.WithContext(ctx).
		Model(&Event{}).
		Where("created_by = ? AND visibility = ?", friendID, VisibilityFriends)
	tx = applyRangeFilter(tx, q)
	var events []Event
	err := tx.Order("start_at ASC").Find(&events).Error
	return events, err
}

func applyRangeFilter(tx *gorm.DB, q ListEventsQuery) *gorm.DB {
	if q.From != nil {
		tx = tx.Where("end_at >= ?", *q.From)
	}
	if q.To != nil {
		tx = tx.Where("start_at <= ?", *q.To)
	}
	return tx
}

// ---- Members ----

func (r *repository) AddMembers(ctx context.Context, eventID uuid.UUID, members []EventMember) error {
	if len(members) == 0 {
		return nil
	}
	now := time.Now()
	rows := make([]EventMember, 0, len(members))
	for _, m := range members {
		if m.EventID == uuid.Nil {
			m.EventID = eventID
		}
		if m.JoinedAt.IsZero() {
			m.JoinedAt = now
		}
		if m.Role == "" {
			m.Role = MemberRoleMember
		}
		rows = append(rows, m)
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&rows).Error
}

func (r *repository) RemoveMember(ctx context.Context, eventID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("event_id = ? AND user_id = ?", eventID, userID).
		Delete(&EventMember{}).Error
}

// RemoveMemberBySource is used by the trip sync layer: when a room member is
// kicked/leaves, look up the trip event by (SourceTrip, tripID) and delete
// the corresponding event_members row in a single JOIN'd DELETE.
func (r *repository) RemoveMemberBySource(ctx context.Context, srcType SourceType, srcID uuid.UUID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(`
		DELETE FROM calendar.event_members m
		USING calendar.events e
		WHERE m.event_id = e.id
		  AND e.source_type = ?
		  AND e.source_id = ?
		  AND m.user_id = ?
	`, srcType, srcID, userID).Error
}

func (r *repository) ListEventMembers(ctx context.Context, eventID uuid.UUID) ([]EventMember, error) {
	var rows []EventMember
	err := r.db.WithContext(ctx).
		Where("event_id = ?", eventID).
		Order("joined_at ASC").
		Find(&rows).Error
	return rows, err
}

func (r *repository) IsEventMember(ctx context.Context, eventID, userID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&EventMember{}).
		Where("event_id = ? AND user_id = ?", eventID, userID).
		Count(&n).Error
	return n > 0, err
}

// ---- Outbox ----

func (r *repository) EnqueueOutbox(ctx context.Context, tripID uuid.UUID, payload []byte, traceID, stream string) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO trip.outbox (id, aggregate_id, stream, trace_id, payload, created_at, attempts, last_error)
		VALUES (?, ?, ?, ?, ?, now(), 0, '')
	`, uuid.New(), tripID, stream, traceID, payload).Error
}

// insertCalendarOutboxRow marshals the outbox event and writes it inside the
// same tx that mutated the event.
func insertCalendarOutboxRow(tx *gorm.DB, meta *OutboxMeta, e *Event) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(outboxEvent{
		OpType:  meta.OpType,
		TripID:  meta.TripID,
		UserID:  meta.UserID,
		Version: e.Version,
		TraceID: meta.TraceID,
		Data:    data,
	})
	if err != nil {
		return err
	}
	return tx.Exec(`
		INSERT INTO trip.outbox (id, aggregate_id, stream, trace_id, payload, created_at, attempts, last_error)
		VALUES (?, ?, ?, ?, ?, now(), 0, '')
	`, uuid.New(), meta.TripID, meta.Stream, meta.TraceID, payload).Error
}
