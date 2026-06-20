package friend

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IRepository interface {
	WithTx(ctx context.Context, fn func(IRepository) error) error

	FindUserByID(ctx context.Context, id uuid.UUID) (*auth.User, error)
	FindUsersByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error)
	SearchUsers(ctx context.Context, requesterID uuid.UUID, query string, limit int) ([]auth.User, error)

	AreFriends(ctx context.Context, a, b uuid.UUID) (bool, error)
	ListFriends(ctx context.Context, userID uuid.UUID) ([]Friendship, error)
	CreateFriendship(ctx context.Context, a, b uuid.UUID) error
	DeleteFriendship(ctx context.Context, a, b uuid.UUID) error

	FindRequestByID(ctx context.Context, id uuid.UUID) (*Request, error)
	FindPendingRequest(ctx context.Context, senderID, receiverID uuid.UUID) (*Request, error)
	CreateOrReviveRequest(ctx context.Context, req *Request) error
	UpdateRequestStatus(ctx context.Context, id uuid.UUID, status RequestStatus) error
	ListIncomingRequests(ctx context.Context, userID uuid.UUID) ([]Request, error)
	ListOutgoingRequests(ctx context.Context, userID uuid.UUID) ([]Request, error)

	IsBlocked(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error)
	BlockUser(ctx context.Context, blockerID, blockedID uuid.UUID) error
	UnblockUser(ctx context.Context, blockerID, blockedID uuid.UUID) error
	ListBlocks(ctx context.Context, userID uuid.UUID) ([]Block, error)
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

func (r *repository) SearchUsers(ctx context.Context, requesterID uuid.UUID, query string, limit int) ([]auth.User, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	pattern := "%" + strings.ToLower(q) + "%"
	var users []auth.User
	err := r.db.WithContext(ctx).
		Where("id <> ?", requesterID).
		Where("status = ?", auth.UserStatusActive).
		Where("is_blocked = ?", false).
		Where("(LOWER(email) LIKE ? OR LOWER(name) LIKE ?)", pattern, pattern).
		Order("name ASC").
		Limit(limit).
		Find(&users).Error
	return users, err
}

func (r *repository) AreFriends(ctx context.Context, a, b uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Friendship{}).
		Where("user_id = ? AND friend_id = ?", a, b).
		Count(&count).Error
	return count > 0, err
}

func (r *repository) ListFriends(ctx context.Context, userID uuid.UUID) ([]Friendship, error) {
	var fs []Friendship
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").Find(&fs).Error
	return fs, err
}

func (r *repository) CreateFriendship(ctx context.Context, a, b uuid.UUID) error {
	now := time.Now()
	rows := []Friendship{
		{UserID: a, FriendID: b, CreatedAt: now},
		{UserID: b, FriendID: a, CreatedAt: now},
	}
	return r.db.WithContext(ctx).
		Exec("INSERT INTO friend.friends (user_id, friend_id, created_at) VALUES (?, ?, ?), (?, ?, ?) ON CONFLICT DO NOTHING",
			rows[0].UserID, rows[0].FriendID, rows[0].CreatedAt,
			rows[1].UserID, rows[1].FriendID, rows[1].CreatedAt).Error
}

func (r *repository) DeleteFriendship(ctx context.Context, a, b uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)", a, b, b, a).
		Delete(&Friendship{}).Error
}

func (r *repository) FindRequestByID(ctx context.Context, id uuid.UUID) (*Request, error) {
	var req Request
	err := r.db.WithContext(ctx).First(&req, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &req, err
}

func (r *repository) FindPendingRequest(ctx context.Context, senderID, receiverID uuid.UUID) (*Request, error) {
	var req Request
	err := r.db.WithContext(ctx).
		Where("sender_id = ? AND receiver_id = ? AND status = ?", senderID, receiverID, RequestPending).
		First(&req).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &req, err
}

func (r *repository) CreateOrReviveRequest(ctx context.Context, req *Request) error {
	now := time.Now()
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}
	req.Status = RequestPending
	req.CreatedAt = now
	req.UpdatedAt = now
	// Clear any prior row for the unordered pair (e.g., an old accepted/declined
	// row, or a stale row with the reverse direction) before inserting. Both
	// directions are constrained by idx_friend_pair, so without this delete a
	// cross-direction reuse hits a unique-key violation.
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			DELETE FROM friend.invitations
			WHERE (sender_id = ? AND receiver_id = ?)
			   OR (sender_id = ? AND receiver_id = ?)
		`, req.SenderID, req.ReceiverID, req.ReceiverID, req.SenderID).Error; err != nil {
			return err
		}
		return tx.Exec(`
			INSERT INTO friend.invitations (id, sender_id, receiver_id, status, message, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, req.ID, req.SenderID, req.ReceiverID, req.Status, req.Message, req.CreatedAt, req.UpdatedAt).Error
	})
}

func (r *repository) UpdateRequestStatus(ctx context.Context, id uuid.UUID, status RequestStatus) error {
	return r.db.WithContext(ctx).Model(&Request{}).
		Where("id = ?", id).
		Updates(map[string]any{"status": status, "updated_at": time.Now()}).Error
}

func (r *repository) ListIncomingRequests(ctx context.Context, userID uuid.UUID) ([]Request, error) {
	var reqs []Request
	err := r.db.WithContext(ctx).
		Where("receiver_id = ? AND status = ?", userID, RequestPending).
		Order("created_at DESC").Find(&reqs).Error
	return reqs, err
}

func (r *repository) ListOutgoingRequests(ctx context.Context, userID uuid.UUID) ([]Request, error) {
	var reqs []Request
	err := r.db.WithContext(ctx).
		Where("sender_id = ? AND status = ?", userID, RequestPending).
		Order("created_at DESC").Find(&reqs).Error
	return reqs, err
}

func (r *repository) IsBlocked(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Block{}).
		Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).
		Count(&count).Error
	return count > 0, err
}

func (r *repository) BlockUser(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		"INSERT INTO friend.blocks (blocker_id, blocked_id, created_at) VALUES (?, ?, ?) ON CONFLICT DO NOTHING",
		blockerID, blockedID, time.Now()).Error
}

func (r *repository) UnblockUser(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).
		Delete(&Block{}).Error
}

func (r *repository) ListBlocks(ctx context.Context, userID uuid.UUID) ([]Block, error) {
	var bs []Block
	err := r.db.WithContext(ctx).
		Where("blocker_id = ?", userID).
		Order("created_at DESC").Find(&bs).Error
	return bs, err
}
