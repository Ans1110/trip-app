package packing

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ItemPackInfo struct {
	PackedByMe  bool
	PackedCount int
}

type IRepository interface {
	WithTx(ctx context.Context, fn func(IRepository) error) error

	CreateItem(ctx context.Context, i *Item) error
	FindItem(ctx context.Context, id uuid.UUID) (*Item, error)
	ListItemsByTrip(ctx context.Context, tripID uuid.UUID) ([]Item, error)
	UpdateItem(ctx context.Context, id uuid.UUID, patch map[string]any) (*Item, error)
	DeleteItem(ctx context.Context, id uuid.UUID) error
	NextSortOrder(ctx context.Context, tripID uuid.UUID) (int, error)

	// Per-user pack state
	SetPacked(ctx context.Context, itemID, userID uuid.UUID, packed bool) error
	PackInfoForItems(ctx context.Context, itemIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]ItemPackInfo, error)
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

func (r *repository) CreateItem(ctx context.Context, i *Item) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(i).Error
}

func (r *repository) FindItem(ctx context.Context, id uuid.UUID) (*Item, error) {
	var i Item
	err := r.db.WithContext(ctx).First(&i, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &i, err
}

func (r *repository) ListItemsByTrip(ctx context.Context, tripID uuid.UUID) ([]Item, error) {
	var rows []Item
	err := r.db.WithContext(ctx).
		Where("trip_id = ?", tripID).
		Order("sort_order ASC, created_at ASC").
		Find(&rows).Error
	return rows, err
}

func (r *repository) UpdateItem(ctx context.Context, id uuid.UUID, patch map[string]any) (*Item, error) {
	if len(patch) == 0 {
		return r.FindItem(ctx, id)
	}
	patch["updated_at"] = time.Now()
	res := r.db.WithContext(ctx).Model(&Item{}).Where("id = ?", id).Updates(patch)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrItemNotFound
	}
	return r.FindItem(ctx, id)
}

func (r *repository) DeleteItem(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&Item{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrItemNotFound
	}
	return nil
}

func (r *repository) NextSortOrder(ctx context.Context, tripID uuid.UUID) (int, error) {
	var max *int
	err := r.db.WithContext(ctx).
		Model(&Item{}).
		Where("trip_id = ?", tripID).
		Select("MAX(sort_order)").
		Scan(&max).Error
	if err != nil {
		return 0, err
	}
	if max == nil {
		return 0, nil
	}
	return *max + 1, nil
}

func (r *repository) SetPacked(ctx context.Context, itemID, userID uuid.UUID, packed bool) error {
	if packed {
		return r.db.WithContext(ctx).
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&ItemPack{ItemID: itemID, UserID: userID, PackedAt: time.Now()}).
			Error
	}
	return r.db.WithContext(ctx).
		Where("item_id = ? AND user_id = ?", itemID, userID).
		Delete(&ItemPack{}).Error
}

func (r *repository) PackInfoForItems(ctx context.Context, itemIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]ItemPackInfo, error) {
	out := make(map[uuid.UUID]ItemPackInfo, len(itemIDs))
	if len(itemIDs) == 0 {
		return out, nil
	}

	type row struct {
		ItemID     uuid.UUID
		Count      int
		PackedByMe bool
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Table("packing.item_pack").
		Select("item_id, COUNT(*) AS count, BOOL_OR(user_id = ?) AS packed_by_me", userID).
		Where("item_id IN ?", itemIDs).
		Group("item_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ItemID] = ItemPackInfo{
			PackedByMe:  r.PackedByMe,
			PackedCount: r.Count,
		}
	}
	return out, nil
}
