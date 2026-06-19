package audit

import (
	"context"

	"gorm.io/gorm"
)

// Writer is the narrow interface domain services depend on. Production code
// uses *Repository; tests use a fake.
type Writer interface {
	Create(ctx context.Context, log *Log) error
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, log *Log) error {
	return r.db.WithContext(ctx).Create(log).Error
}
