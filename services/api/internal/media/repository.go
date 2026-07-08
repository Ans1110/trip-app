package media

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrAssetNotFound   = errors.New("media: asset not found")
	ErrSessionNotFound = errors.New("media: upload session not found")
)

type IRepository interface {
	CreateSession(ctx context.Context, s *UploadSession) error
	FindSession(ctx context.Context, id uuid.UUID) (*UploadSession, error)
	CompleteSession(ctx context.Context, sessionID, mediaID uuid.UUID) error
	ListExpiredPendingSessions(ctx context.Context, now time.Time, limit int) ([]UploadSession, error)
	DeleteSession(ctx context.Context, id uuid.UUID) error

	CreateAsset(ctx context.Context, a *Asset) error
	FindAsset(ctx context.Context, id uuid.UUID) (*Asset, error)
	SoftDeleteAsset(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) IRepository {
	return &repository{db: db}
}

// ---- Sessions ----

func (r *repository) CreateSession(ctx context.Context, s *UploadSession) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *repository) FindSession(ctx context.Context, id uuid.UUID) (*UploadSession, error) {
	var s UploadSession
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *repository) CompleteSession(ctx context.Context, sessionID, mediaID uuid.UUID) error {
	// Complete-then-null-media_id guard: reject a second completion attempt.
	res := r.db.WithContext(ctx).Exec(`
		UPDATE media.upload_sessions
		   SET completed_at = now(), media_id = ?
		 WHERE id = ? AND completed_at IS NULL`,
		mediaID, sessionID,
	)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (r *repository) ListExpiredPendingSessions(ctx context.Context, now time.Time, limit int) ([]UploadSession, error) {
	if limit <= 0 {
		limit = 100
	}
	var rows []UploadSession
	err := r.db.WithContext(ctx).
		Where("completed_at IS NULL AND expires_at < ?", now).
		Order("expires_at ASC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *repository) DeleteSession(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&UploadSession{}).Error
}

// ---- Assets ----

func (r *repository) CreateAsset(ctx context.Context, a *Asset) error {
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *repository) FindAsset(ctx context.Context, id uuid.UUID) (*Asset, error) {
	var a Asset
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAssetNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// SoftDeleteAsset marks the row deleted. It does NOT touch the S3 object any
// chat message that still references the asset stays readable until the FE
// finishes handling the delete broadcast.
func (r *repository) SoftDeleteAsset(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE media.assets SET deleted_at = now()
		 WHERE id = ? AND deleted_at IS NULL`,
		id,
	)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrAssetNotFound
	}
	return nil
}
