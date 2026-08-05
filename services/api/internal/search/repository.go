package search

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostHit struct {
	ID          uuid.UUID
	PublishedAt time.Time
}

type Cursor struct {
	PublishedAt time.Time
	ID          uuid.UUID
}

type IRepository interface {
	SearchPosts(ctx context.Context, query string, cursor *Cursor, limit int) ([]PostHit, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) IRepository { return &repository{db: db} }

func (r *repository) SearchPosts(ctx context.Context, query string, cursor *Cursor, limit int) ([]PostHit, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	q := r.db.WithContext(ctx).
		Table("post.posts").
		Select("id, published_at").
		Where("deleted_at IS NULL").
		Where("search_vector @@ plainto_tsquery('simple', ?)", query)
	if cursor != nil {
		q = q.Where("(published_at, id) < (?, ?)", cursor.PublishedAt, cursor.ID)
	}
	var rows []PostHit
	err := q.Order("published_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}
