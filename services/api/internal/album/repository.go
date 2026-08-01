package album

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrPhotoNotFound      = errors.New("album: photo not found")
	ErrShareTokenNotFound = errors.New("album: share token not found")
	ErrShareTokenInactive = errors.New("album: share token revoked or expired")
	ErrForbidden          = errors.New("album: forbidden")
)

type IRepository interface {
	WithTx(ctx context.Context, fn func(IRepository) error) error

	// Photos
	CreatePhoto(ctx context.Context, p *Photo) error
	UpdatePhoto(ctx context.Context, id uuid.UUID, patch map[string]any) (*Photo, error)
	FindPhoto(ctx context.Context, id uuid.UUID) (*Photo, error)
	ListPhotosByTrip(ctx context.Context, tripID uuid.UUID) ([]Photo, error)
	SoftDeletePhoto(ctx context.Context, id uuid.UUID) error

	// Share tokens
	CreateShareToken(ctx context.Context, s *ShareToken) error
	FindShareTokenByHash(ctx context.Context, hash []byte) (*ShareToken, error)
	FindShareToken(ctx context.Context, id uuid.UUID) (*ShareToken, error)
	ListShareTokensByTrip(ctx context.Context, tripID uuid.UUID) ([]ShareToken, error)
	RevokeShareToken(ctx context.Context, id, revokedBy uuid.UUID) error
	TouchShareTokenAccess(ctx context.Context, id uuid.UUID, at time.Time) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) IRepository { return &repository{db: db} }

func (r *repository) WithTx(ctx context.Context, fn func(IRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&repository{db: tx})
	})
}

// ---- Photos ----

func (r *repository) CreatePhoto(ctx context.Context, p *Photo) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *repository) UpdatePhoto(ctx context.Context, id uuid.UUID, patch map[string]any) (*Photo, error) {
	if len(patch) == 0 {
		return r.FindPhoto(ctx, id)
	}
	res := r.db.WithContext(ctx).
		Model(&Photo{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(patch)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrPhotoNotFound
	}
	return r.FindPhoto(ctx, id)
}

func (r *repository) FindPhoto(ctx context.Context, id uuid.UUID) (*Photo, error) {
	var p Photo
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPhotoNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) ListPhotosByTrip(ctx context.Context, tripID uuid.UUID) ([]Photo, error) {
	var rows []Photo
	err := r.db.WithContext(ctx).
		Where("trip_id = ? AND deleted_at IS NULL", tripID).
		Order("taken_at DESC").
		Find(&rows).Error
	return rows, err
}

func (r *repository) SoftDeletePhoto(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE album.album_photo SET deleted_at = now()
		 WHERE id = ? AND deleted_at IS NULL`,
		id,
	)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPhotoNotFound
	}
	return nil
}

// ---- Share tokens ----

func (r *repository) CreateShareToken(ctx context.Context, s *ShareToken) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *repository) FindShareTokenByHash(ctx context.Context, hash []byte) (*ShareToken, error) {
	var s ShareToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrShareTokenNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *repository) FindShareToken(ctx context.Context, id uuid.UUID) (*ShareToken, error) {
	var s ShareToken
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrShareTokenNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *repository) ListShareTokensByTrip(ctx context.Context, tripID uuid.UUID) ([]ShareToken, error) {
	var rows []ShareToken
	err := r.db.WithContext(ctx).
		Where("trip_id = ?", tripID).
		Order("created_at DESC").
		Find(&rows).Error
	return rows, err
}

func (r *repository) RevokeShareToken(ctx context.Context, id, revokedBy uuid.UUID) error {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE album.share_token
		   SET revoked_at = now(), revoked_by = ?
		 WHERE id = ? AND revoked_at IS NULL`,
		revokedBy, id,
	)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrShareTokenNotFound
	}
	return nil
}

// TouchShareTokenAccess is a fire-and-forget audit hook. Best-effort — the
// public read path should not fail if this update loses to a race.
func (r *repository) TouchShareTokenAccess(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE album.share_token
		   SET last_accessed_at = ?
		 WHERE id = ?`,
		at, id,
	).Error
}
