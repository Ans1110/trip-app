package feed

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FeedCursor struct {
	PublishedAt time.Time
	ID          uuid.UUID
}

type IRepository interface {
	InsertMany(ctx context.Context, items []FeedItem) error
	DeleteBySubject(ctx context.Context, subjectType string, subjectID uuid.UUID) error
	ListForUser(ctx context.Context, userID uuid.UUID, cursor *FeedCursor, limit int) ([]FeedItem, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) IRepository { return &repository{db: db} }

// ON CONFLICT DO NOTHING so consumer retries after transient failure are idempotent.
func (r *repository) InsertMany(ctx context.Context, items []FeedItem) error {
	if len(items) == 0 {
		return nil
	}
	for i := range items {
		if items[i].ID == uuid.Nil {
			items[i].ID = uuid.New()
		}
	}
	sql := `INSERT INTO feed.feed_items
	          (id, user_id, actor_id, event_type, subject_type, subject_id, published_at)
	        VALUES ` + placeholders(len(items), 7) + `
	        ON CONFLICT (user_id, event_type, subject_id) DO NOTHING`
	args := make([]any, 0, len(items)*7)
	for _, it := range items {
		args = append(args, it.ID, it.UserID, it.ActorID, it.EventType, it.SubjectType, it.SubjectID, it.PublishedAt)
	}
	return r.db.WithContext(ctx).Exec(sql, args...).Error
}

func (r *repository) DeleteBySubject(ctx context.Context, subjectType string, subjectID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("subject_type = ? AND subject_id = ?", subjectType, subjectID).
		Delete(&FeedItem{}).Error
}

func (r *repository) ListForUser(ctx context.Context, userID uuid.UUID, cursor *FeedCursor, limit int) ([]FeedItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if cursor != nil {
		q = q.Where("(published_at, id) < (?, ?)", cursor.PublishedAt, cursor.ID)
	}
	var rows []FeedItem
	err := q.Order("published_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func placeholders(rows, cols int) string {
	if rows == 0 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < rows; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString("(")
		for j := 0; j < cols; j++ {
			if j > 0 {
				b.WriteString(",")
			}
			b.WriteString("?")
		}
		b.WriteString(")")
	}
	return b.String()
}
