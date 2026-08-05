package feed

import (
	"time"

	"github.com/google/uuid"
)

type FeedItem struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID       uuid.UUID `gorm:"type:uuid;column:user_id;not null;index"`
	ActorID      uuid.UUID `gorm:"type:uuid;column:actor_id;not null"`
	EventType    string    `gorm:"column:event_type;not null"`
	SubjectType  string    `gorm:"column:subject_type;not null"`
	SubjectID    uuid.UUID `gorm:"type:uuid;column:subject_id;not null"`
	PublishedAt  time.Time `gorm:"column:published_at;not null"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (FeedItem) TableName() string { return "feed.feed_items" }
