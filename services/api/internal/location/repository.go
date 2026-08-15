package location

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IRepository interface {
	CreateSavedPlace(ctx context.Context, sp *SavedPlace) error
	FindSavedPlaceByID(ctx context.Context, id uuid.UUID) (*SavedPlace, error)
	UpdateSavedPlace(ctx context.Context, id uuid.UUID, patch map[string]any) error
	SoftDeleteSavedPlace(ctx context.Context, id uuid.UUID) error
	ListSavedPlacesForUser(ctx context.Context, userID uuid.UUID, tripID *uuid.UUID) ([]SavedPlace, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) IRepository { return &repository{db: db} }

func (r *repository) CreateSavedPlace(ctx context.Context, sp *SavedPlace) error {
	if sp.ID == uuid.Nil {
		sp.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(sp).Error
}

func (r *repository) FindSavedPlaceByID(ctx context.Context, id uuid.UUID) (*SavedPlace, error) {
	var sp SavedPlace
	err := r.db.WithContext(ctx).First(&sp, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &sp, err
}

func (r *repository) UpdateSavedPlace(ctx context.Context, id uuid.UUID, patch map[string]any) error {
	if len(patch) == 0 {
		return nil
	}
	patch["updated_at"] = time.Now()
	return r.db.WithContext(ctx).Model(&SavedPlace{}).Where("id = ?", id).Updates(patch).Error
}

func (r *repository) SoftDeleteSavedPlace(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&SavedPlace{}, "id = ?", id).Error
}

func (r *repository) ListSavedPlacesForUser(ctx context.Context, userID uuid.UUID, tripID *uuid.UUID) ([]SavedPlace, error) {
	tx := r.db.WithContext(ctx).
		Model(&SavedPlace{}).
		Where("user_id = ?", userID)
	if tripID != nil {
		tx = tx.Where("trip_id = ?", *tripID)
	}
	var rows []SavedPlace
	err := tx.Order("created_at DESC").Find(&rows).Error
	return rows, err
}
