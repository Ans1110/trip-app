package chat

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrRoomNotFound = errors.New("chat: room not found")
var ErrHideNotAllowed = errors.New("chat: room type does not support hide")

type IRepository interface {
	FindRoomByID(ctx context.Context, id uuid.UUID) (*Room, error)
	FindGroupRoomByTripID(ctx context.Context, tripID uuid.UUID) (*Room, error)
	EnsureGroupRoom(ctx context.Context, tripID uuid.UUID, name string, memberIDs []uuid.UUID) (*Room, error)
	FindOrCreateDMRoom(ctx context.Context, a, b uuid.UUID) (*Room, error)
	AddMembers(ctx context.Context, roomID uuid.UUID, userIDs []uuid.UUID) error
	IsRoomMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
	ListRoomMemberIDs(ctx context.Context, roomID uuid.UUID) ([]uuid.UUID, error)

	ListRoomsForUser(ctx context.Context, userID uuid.UUID) ([]Room, error)
	ListDMPeers(ctx context.Context, roomIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID]uuid.UUID, error)
	HideDMForUser(ctx context.Context, roomID, userID uuid.UUID) error
	UnhideRoomForUser(ctx context.Context, roomID, userID uuid.UUID) error
	UnhideRooms(ctx context.Context, roomIDs []uuid.UUID) error
	InsertMessages(ctx context.Context, msgs []*Message) error
	ListMessages(ctx context.Context, roomID uuid.UUID, before *time.Time, limit int) ([]Message, error)
	LastMessages(ctx context.Context, roomIDs []uuid.UUID) (map[uuid.UUID]Message, error)

	UpsertReadReceipt(ctx context.Context, roomID, userID, lastReadID uuid.UUID) (*ReadReceipt, error)
	ListReadReceipts(ctx context.Context, roomID uuid.UUID) ([]ReadReceipt, error)
	CountUnread(ctx context.Context, roomID, userID uuid.UUID) (int, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) IRepository {
	return &repository{db: db}
}

// ---- Rooms ----

func (r *repository) FindRoomByID(ctx context.Context, id uuid.UUID) (*Room, error) {
	var room Room
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&room).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRoomNotFound
	}
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *repository) FindGroupRoomByTripID(ctx context.Context, tripID uuid.UUID) (*Room, error) {
	var room Room
	err := r.db.WithContext(ctx).
		Where("trip_id = ? AND type = ?", tripID, RoomGroup).
		First(&room).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRoomNotFound
	}
	if err != nil {
		return nil, err
	}
	return &room, nil
}

// EnsureGroupRoom is idempotent: if a group room already exists for the trip
// it returns that; otherwise it creates one and seeds membership.
func (r *repository) EnsureGroupRoom(ctx context.Context, tripID uuid.UUID, name string, memberIDs []uuid.UUID) (*Room, error) {
	if room, err := r.FindGroupRoomByTripID(ctx, tripID); err == nil {
		return room, nil
	} else if !errors.Is(err, ErrRoomNotFound) {
		return nil, err
	}

	var out *Room
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		room := &Room{TripID: &tripID, Name: name, Type: RoomGroup}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(room).Error; err != nil {
			return err
		}
		// Race: another instance created it in the gap.
		if room.ID == uuid.Nil {
			if err := tx.Where("trip_id = ? AND type = ?", tripID, RoomGroup).First(room).Error; err != nil {
				return err
			}
		}
		out = room
		if len(memberIDs) == 0 {
			return nil
		}
		rows := make([]RoomMember, 0, len(memberIDs))
		now := time.Now()
		for _, uid := range memberIDs {
			rows = append(rows, RoomMember{RoomID: room.ID, UserID: uid, JoinedAt: now})
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func dmPairKey(a, b uuid.UUID) string {
	lo, hi := a, b
	if hi.String() < lo.String() {
		lo, hi = hi, lo
	}
	sum := sha1.Sum([]byte(lo.String() + "|" + hi.String()))
	return hex.EncodeToString(sum[:])
}

func (r *repository) FindOrCreateDMRoom(ctx context.Context, a, b uuid.UUID) (*Room, error) {
	if a == b {
		return nil, errors.New("chat: cannot DM yourself")
	}
	key := dmPairKey(a, b)

	var existing Room
	err := r.db.WithContext(ctx).
		Where("type = ? AND dm_pair_key = ?", RoomDM, key).
		First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var out *Room
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		room := &Room{Type: RoomDM, DMPairKey: &key}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "dm_pair_key"}},
			TargetWhere: clause.Where{Exprs: []clause.Expression{
				clause.Eq{Column: "type", Value: string(RoomDM)},
				clause.Expr{SQL: `"dm_pair_key" IS NOT NULL`},
			}},
			DoNothing: true,
		}).Create(room).Error; err != nil {
			return err
		}
		if room.ID == uuid.Nil {
			if err := tx.Where("type = ? AND dm_pair_key = ?", RoomDM, key).First(room).Error; err != nil {
				return err
			}
		}
		out = room
		now := time.Now()
		rows := []RoomMember{
			{RoomID: room.ID, UserID: a, JoinedAt: now},
			{RoomID: room.ID, UserID: b, JoinedAt: now},
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *repository) AddMembers(ctx context.Context, roomID uuid.UUID, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}
	rows := make([]RoomMember, 0, len(userIDs))
	now := time.Now()
	for _, uid := range userIDs {
		rows = append(rows, RoomMember{RoomID: roomID, UserID: uid, JoinedAt: now})
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&rows).Error
}

// IsRoomMember treats trip-linked group rooms as membership-follows-trip: a
// user is a member of the chat.room iff they are a member of the paired trip
// room. Explicit chat.room_members rows serve as the source of truth for
// DMs and any orphan group rooms.
func (r *repository) IsRoomMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM (
			SELECT 1 FROM chat.room_members
			 WHERE room_id = ? AND user_id = ?
			UNION ALL
			SELECT 1 FROM chat.room cr
			  JOIN trip.trip t ON t.id = cr.trip_id AND t.deleted_at IS NULL
			 WHERE cr.id = ? AND cr.type = 'group' AND t.owner_id = ?
			UNION ALL
			SELECT 1 FROM chat.room cr
			  JOIN trip.rooms tr ON tr.trip_id = cr.trip_id
			  JOIN trip.room_members trm ON trm.room_id = tr.id
			 WHERE cr.id = ? AND cr.type = 'group' AND trm.user_id = ?
		) AS x`,
		roomID, userID,
		roomID, userID,
		roomID, userID,
	).Scan(&n).Error
	return n > 0, err
}

func (r *repository) ListRoomMemberIDs(ctx context.Context, roomID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).Raw(`
		SELECT user_id FROM (
			SELECT user_id FROM chat.room_members WHERE room_id = ?
			UNION
			SELECT t.owner_id AS user_id
			  FROM chat.room cr
			  JOIN trip.trip t ON t.id = cr.trip_id AND t.deleted_at IS NULL
			 WHERE cr.id = ? AND cr.type = 'group'
			UNION
			SELECT trm.user_id
			  FROM chat.room cr
			  JOIN trip.rooms tr ON tr.trip_id = cr.trip_id
			  JOIN trip.room_members trm ON trm.room_id = tr.id
			 WHERE cr.id = ? AND cr.type = 'group'
		) AS m`, roomID, roomID, roomID).Scan(&ids).Error
	return ids, err
}

func (r *repository) ListRoomsForUser(ctx context.Context, userID uuid.UUID) ([]Room, error) {
	var rooms []Room
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT cr.*
		  FROM chat.room cr
		  LEFT JOIN chat.room_members crm
		    ON crm.room_id = cr.id AND crm.user_id = ?
		  LEFT JOIN trip.trip t
		    ON cr.type = 'group' AND cr.trip_id = t.id AND t.deleted_at IS NULL
		  LEFT JOIN trip.rooms tr
		    ON cr.type = 'group' AND cr.trip_id = tr.trip_id
		  LEFT JOIN trip.room_members trm
		    ON tr.id = trm.room_id AND trm.user_id = ?
		 WHERE (crm.user_id IS NOT NULL AND crm.hidden_at IS NULL)
		    OR t.owner_id = ?
		    OR trm.user_id IS NOT NULL
		 ORDER BY cr.created_at DESC`,
		userID, userID, userID,
	).Scan(&rooms).Error
	return rooms, err
}

func (r *repository) ListDMPeers(ctx context.Context, roomIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	out := make(map[uuid.UUID]uuid.UUID, len(roomIDs))
	if len(roomIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		RoomID uuid.UUID `gorm:"column:room_id"`
		UserID uuid.UUID `gorm:"column:user_id"`
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT crm.room_id, crm.user_id
		  FROM chat.room_members crm
		  JOIN chat.room cr ON cr.id = crm.room_id
		 WHERE crm.room_id IN ?
		   AND crm.user_id <> ?
		   AND cr.type = 'dm'`,
		roomIDs, viewerID,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.RoomID] = row.UserID
	}
	return out, nil
}

func (r *repository) HideDMForUser(ctx context.Context, roomID, userID uuid.UUID) error {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE chat.room_members
		   SET hidden_at = now()
		 WHERE room_id = ?
		   AND user_id = ?
		   AND EXISTS (
		     SELECT 1 FROM chat.room WHERE id = ? AND type = 'dm'
		   )`,
		roomID, userID, roomID,
	)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		var room Room
		if err := r.db.WithContext(ctx).Where("id = ?", roomID).First(&room).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRoomNotFound
			}
			return err
		}
		if room.Type != RoomDM {
			return ErrHideNotAllowed
		}
	}
	return nil
}

func (r *repository) UnhideRoomForUser(ctx context.Context, roomID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE chat.room_members
		   SET hidden_at = NULL
		 WHERE room_id = ? AND user_id = ? AND hidden_at IS NOT NULL`,
		roomID, userID,
	).Error
}

func (r *repository) UnhideRooms(ctx context.Context, roomIDs []uuid.UUID) error {
	if len(roomIDs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Exec(`
		UPDATE chat.room_members
		   SET hidden_at = NULL
		 WHERE room_id IN ? AND hidden_at IS NOT NULL`,
		roomIDs,
	).Error
}

// ---- Messages ----

func (r *repository) InsertMessages(ctx context.Context, msgs []*Message) error {
	if len(msgs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "room_id"}, {Name: "sender_id"}, {Name: "client_msg_id"}},
			TargetWhere: clause.Where{Exprs: []clause.Expression{
				clause.Expr{SQL: `"client_msg_id" IS NOT NULL`},
			}},
			DoNothing: true,
		}).
		Create(&msgs).Error
}

func (r *repository) ListMessages(ctx context.Context, roomID uuid.UUID, before *time.Time, limit int) ([]Message, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := r.db.WithContext(ctx).
		Where("room_id = ? AND is_deleted = FALSE", roomID).
		Order("created_at DESC").
		Limit(limit)
	if before != nil {
		q = q.Where("created_at < ?", *before)
	}
	var out []Message
	if err := q.Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *repository) LastMessages(ctx context.Context, roomIDs []uuid.UUID) (map[uuid.UUID]Message, error) {
	if len(roomIDs) == 0 {
		return map[uuid.UUID]Message{}, nil
	}
	var rows []Message
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT ON (room_id) *
		  FROM chat.message
		 WHERE room_id IN ? AND is_deleted = FALSE
		 ORDER BY room_id, created_at DESC`,
		roomIDs,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]Message, len(rows))
	for _, m := range rows {
		out[m.RoomID] = m
	}
	return out, nil
}

// ---- Read receipts ----

func (r *repository) UpsertReadReceipt(ctx context.Context, roomID, userID, lastReadID uuid.UUID) (*ReadReceipt, error) {
	rec := &ReadReceipt{
		RoomID:     roomID,
		UserID:     userID,
		LastReadID: lastReadID,
		UpdatedAt:  time.Now(),
	}

	err := r.db.WithContext(ctx).Exec(`
		INSERT INTO chat.read_receipts (room_id, user_id, last_read_id, updated_at)
		VALUES (?, ?, ?, now())
		ON CONFLICT (room_id, user_id) DO UPDATE
		  SET last_read_id = EXCLUDED.last_read_id,
		      updated_at   = EXCLUDED.updated_at
		  WHERE (
		    SELECT created_at FROM chat.message WHERE id = EXCLUDED.last_read_id
		  ) >= (
		    SELECT created_at FROM chat.message WHERE id = chat.read_receipts.last_read_id
		  )`,
		roomID, userID, lastReadID,
	).Error
	if err != nil {
		return nil, err
	}
	return rec, nil
}

func (r *repository) ListReadReceipts(ctx context.Context, roomID uuid.UUID) ([]ReadReceipt, error) {
	var out []ReadReceipt
	err := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Find(&out).Error
	return out, err
}

func (r *repository) CountUnread(ctx context.Context, roomID, userID uuid.UUID) (int, error) {
	var n int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		  FROM chat.message m
		  LEFT JOIN chat.read_receipts rr
		    ON rr.room_id = m.room_id AND rr.user_id = ?
		  LEFT JOIN chat.message anchor ON anchor.id = rr.last_read_id
		 WHERE m.room_id = ?
		   AND m.is_deleted = FALSE
		   AND m.sender_id <> ?
		   AND (anchor.created_at IS NULL OR m.created_at > anchor.created_at)`,
		userID, roomID, userID,
	).Scan(&n).Error
	return int(n), err
}
