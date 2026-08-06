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

	like := "%" + escapeLike(query) + "%"
	q := r.db.WithContext(ctx).
		Table("post.posts AS p").
		Select("p.id, p.published_at").
		Where("p.deleted_at IS NULL").
		Where(`(
			p.search_vector @@ plainto_tsquery('simple', ?)
			OR EXISTS (
				SELECT 1 FROM auth.users u
				LEFT JOIN profile.profiles pr ON pr.user_id = u.id
				WHERE u.id = p.author_id
				  AND (u.name ILIKE ? ESCAPE '\' OR pr.username::text ILIKE ? ESCAPE '\')
			)
		)`, query, like, like)
	if cursor != nil {
		q = q.Where("(p.published_at, p.id) < (?, ?)", cursor.PublishedAt, cursor.ID)
	}
	var rows []PostHit
	err := q.Order("p.published_at DESC, p.id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func escapeLike(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' || c == '%' || c == '_' {
			out = append(out, '\\')
		}
		out = append(out, c)
	}
	return string(out)
}
